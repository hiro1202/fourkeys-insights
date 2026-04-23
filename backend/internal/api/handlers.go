package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiro1202/fourkeys-insights/internal/db"
	gh "github.com/hiro1202/fourkeys-insights/internal/github"
	"github.com/hiro1202/fourkeys-insights/internal/jobs"
	"github.com/hiro1202/fourkeys-insights/internal/metrics"
	"go.uber.org/zap"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Store  db.Store
	GitHub *gh.Client
	Queue  *jobs.Queue
	Logger *zap.Logger
}

// --- Auth ---

func (h *Handler) ValidateAuth(w http.ResponseWriter, r *http.Request) {
	login, err := h.GitHub.ValidateToken(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid token or insufficient permissions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"login": login})
}

// --- Repos ---

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	repoInfos, err := h.GitHub.ListAccessibleRepos(r.Context())
	if err != nil {
		h.Logger.Error("failed to list repos", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	var result []*db.Repo
	for _, ri := range repoInfos {
		id, err := h.Store.UpsertRepo(r.Context(), &db.Repo{
			Owner:         ri.Owner,
			Name:          ri.Name,
			FullName:      ri.FullName,
			DefaultBranch: ri.DefaultBranch,
		})
		if err != nil {
			h.Logger.Error("failed to upsert repo", zap.String("repo", ri.FullName), zap.Error(err))
			continue
		}
		repo, _ := h.Store.GetRepo(r.Context(), id)
		if repo != nil {
			result = append(result, repo)
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetRepoSettings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid repo ID")
		return
	}

	repo, err := h.Store.GetRepo(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Repo not found")
		return
	}

	writeJSON(w, http.StatusOK, db.RepoSettings{
		IncidentRules: repo.IncidentRules.String,
		LeadTimeStart: repo.LeadTimeStart,
		MTTRStart:     repo.MTTRStart.String,
	})
}

func (h *Handler) UpdateRepoSettings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid repo ID")
		return
	}

	var settings db.RepoSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.Store.UpdateRepoSettings(r.Context(), id, &settings); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update settings")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Groups ---

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.Store.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list groups")
		return
	}
	if groups == nil {
		groups = []*db.Group{}
	}
	writeJSON(w, http.StatusOK, groups)
}

type createGroupRequest struct {
	Name            string  `json:"name"`
	AggregationUnit string  `json:"aggregation_unit"`
	RepoIDs         []int64 `json:"repo_ids"`
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.AggregationUnit == "" {
		req.AggregationUnit = "weekly"
	}
	if req.AggregationUnit != "weekly" && req.AggregationUnit != "monthly" {
		writeError(w, http.StatusBadRequest, "aggregation_unit must be 'weekly' or 'monthly'")
		return
	}
	if len(req.RepoIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one repo is required")
		return
	}

	id, err := h.Store.CreateGroup(r.Context(), req.Name, req.AggregationUnit, req.RepoIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create group")
		return
	}

	group, _ := h.Store.GetGroup(r.Context(), id)
	writeJSON(w, http.StatusCreated, group)
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	if err := h.Store.DeleteGroup(r.Context(), id); err != nil {
		h.Logger.Error("failed to delete group", zap.Int64("group_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Metrics ---

// buildRulesMap creates the per-repo incident rules map using group defaults.
func (h *Handler) buildRulesMap(group *db.Group) map[int64]metrics.IncidentRules {
	groupRules := metrics.ParseIncidentRules(group.IncidentRules)
	rulesMap := make(map[int64]metrics.IncidentRules)
	for _, repo := range group.Repos {
		if repo.IncidentRules.Valid && repo.IncidentRules.String != "" {
			rulesMap[repo.ID] = metrics.ParseIncidentRules(repo.IncidentRules.String)
		} else {
			rulesMap[repo.ID] = groupRules
		}
	}
	return rulesMap
}

func (h *Handler) GetGroupMetrics(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}

	now := time.Now().UTC()
	period := metrics.CalcPeriod(now, group.AggregationUnit)

	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, period.Start)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}
	// Filter to period range
	var periodPRs []*db.PullRequest
	for _, pr := range prs {
		if !pr.MergedAt.After(period.End.Add(time.Second)) {
			periodPRs = append(periodPRs, pr)
		}
	}

	rulesMap := h.buildRulesMap(group)

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           periodPRs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: group.LeadTimeStart,
		MTTRStart:     group.MTTRStart,
		PeriodDays:    period.Days,
	})

	// Previous period for comparison
	prevPeriod := metrics.CalcPreviousPeriod(period, group.AggregationUnit)
	prevPRs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, prevPeriod.Start)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get previous period PRs")
		return
	}
	var prevOnly []*db.PullRequest
	for _, pr := range prevPRs {
		if !pr.MergedAt.Before(prevPeriod.Start) && !pr.MergedAt.After(prevPeriod.End.Add(time.Second)) {
			prevOnly = append(prevOnly, pr)
		}
	}

	var prevResult *metrics.FourKeysResult
	if len(prevOnly) > 0 {
		prev := metrics.Calculate(metrics.CalculateInput{
			PRs:           prevOnly,
			RepoRulesMap:  rulesMap,
			LeadTimeStart: group.LeadTimeStart,
			MTTRStart:     group.MTTRStart,
			PeriodDays:    prevPeriod.Days,
		})
		prevResult = &prev
	}

	lastJob, _ := h.Store.GetLatestJobByGroup(r.Context(), id)

	type extendedResult struct {
		metrics.FourKeysResult
		PreviousPeriod  *metrics.FourKeysResult `json:"previous_period,omitempty"`
		LastSyncAt      *time.Time              `json:"last_sync_at,omitempty"`
		PeriodStart     string                  `json:"period_start"`
		PeriodEnd       string                  `json:"period_end"`
		AggregationUnit string                  `json:"aggregation_unit"`
		LeadTimeStart   string                  `json:"lead_time_start"`
		MTTRStart       string                  `json:"mttr_start"`
	}

	resp := extendedResult{
		FourKeysResult:  result,
		PreviousPeriod:  prevResult,
		PeriodStart:     period.Start.Format("2006-01-02"),
		PeriodEnd:       period.End.Format("2006-01-02"),
		AggregationUnit: group.AggregationUnit,
		LeadTimeStart:   group.LeadTimeStart,
		MTTRStart:       group.MTTRStart,
	}
	if lastJob != nil && lastJob.CompletedAt.Valid {
		t := lastJob.CompletedAt.Time
		resp.LastSyncAt = &t
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Trends ---

type trendDataPoint struct {
	PeriodStart       string   `json:"period_start"`
	PeriodEnd         string   `json:"period_end"`
	LeadTimeHours     float64  `json:"lead_time_hours"`
	DeployFrequency   float64  `json:"deploy_frequency"`
	ChangeFailureRate float64  `json:"change_failure_rate"`
	MTTRHours         *float64 `json:"mttr_hours"`
	TotalPRs          int      `json:"total_prs"`
	IncidentPRs       int      `json:"incident_prs"`
}

func (h *Handler) GetGroupTrends(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}

	// Parse query params
	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = group.AggregationUnit
	}

	now := time.Now().UTC()
	since := now.AddDate(0, -6, 0) // default: 6 months
	until := now

	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			since = t
		}
	}
	if u := r.URL.Query().Get("until"); u != "" {
		if t, err := time.Parse("2006-01-02", u); err == nil {
			until = t
		}
	}

	// Get all PRs for the range
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	rulesMap := h.buildRulesMap(group)
	periods := metrics.CalcTrendPeriods(since, until, unit)

	var points []trendDataPoint
	for _, p := range periods {
		var periodPRs []*db.PullRequest
		for _, pr := range prs {
			if !pr.MergedAt.Before(p.Start) && !pr.MergedAt.After(p.End.Add(time.Second)) {
				periodPRs = append(periodPRs, pr)
			}
		}

		point := trendDataPoint{
			PeriodStart: p.Start.Format("2006-01-02"),
			PeriodEnd:   p.End.Format("2006-01-02"),
			TotalPRs:    len(periodPRs),
		}

		if len(periodPRs) > 0 {
			result := metrics.Calculate(metrics.CalculateInput{
				PRs:           periodPRs,
				RepoRulesMap:  rulesMap,
				LeadTimeStart: group.LeadTimeStart,
				MTTRStart:     group.MTTRStart,
				PeriodDays:    p.Days,
			})
			point.LeadTimeHours = math.Round(result.LeadTimeHours*10) / 10
			point.DeployFrequency = math.Round(result.DeployFrequency*100) / 100
			point.ChangeFailureRate = math.Round(result.ChangeFailureRate*10) / 10
			point.MTTRHours = result.MTTRHours
			point.IncidentPRs = result.IncidentPRs
		}

		points = append(points, point)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data_points": points,
		"unit":        unit,
		"since":       since.Format("2006-01-02"),
		"until":       until.Format("2006-01-02"),
	})
}

// --- Group Settings ---

// repoFallbackStats holds per-repo fallback counts for lead time and MTTR.
type repoFallbackStats struct {
	RepoID            int64 `json:"repo_id"`
	TotalPRs          int   `json:"total_prs"`
	LeadTimeFallbacks int   `json:"lead_time_fallbacks"`
	MTTRFallbacks     int   `json:"mttr_fallbacks"`
}

func (h *Handler) GetGroupSettings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}

	// Compute per-repo fallback stats
	since := time.Now().AddDate(-1, 0, 0)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, since)
	fallbackMap := make(map[int64]*repoFallbackStats)
	if err == nil {
		for _, pr := range prs {
			stats, ok := fallbackMap[pr.RepoID]
			if !ok {
				stats = &repoFallbackStats{RepoID: pr.RepoID}
				fallbackMap[pr.RepoID] = stats
			}
			stats.TotalPRs++
			lt := metrics.CalculateLeadTime(pr, group.LeadTimeStart)
			if lt.UsedFallback {
				stats.LeadTimeFallbacks++
			}
			mttr := metrics.CalculateLeadTime(pr, group.MTTRStart)
			if mttr.UsedFallback {
				stats.MTTRFallbacks++
			}
		}
	}

	fallbackStats := make([]repoFallbackStats, 0, len(fallbackMap))
	for _, s := range fallbackMap {
		fallbackStats = append(fallbackStats, *s)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":             group.Name,
		"aggregation_unit": group.AggregationUnit,
		"lead_time_start":  group.LeadTimeStart,
		"mttr_start":       group.MTTRStart,
		"incident_rules":   group.IncidentRules,
		"repos":            group.Repos,
		"fallback_stats":   fallbackStats,
	})
}

type updateGroupSettingsRequest struct {
	Name            string `json:"name"`
	AggregationUnit string `json:"aggregation_unit"`
	LeadTimeStart   string `json:"lead_time_start"`
	MTTRStart       string `json:"mttr_start"`
	IncidentRules   string `json:"incident_rules"`
}

func (h *Handler) UpdateGroupSettings(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	var req updateGroupSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate aggregation_unit
	if req.AggregationUnit != "" && req.AggregationUnit != "weekly" && req.AggregationUnit != "monthly" {
		writeError(w, http.StatusBadRequest, "aggregation_unit must be 'weekly' or 'monthly'")
		return
	}

	// Validate lead_time_start and mttr_start
	validStarts := map[string]bool{"first_commit_at": true, "issue.created_at": true, "pr_created_at": true}
	if req.LeadTimeStart != "" && !validStarts[req.LeadTimeStart] {
		writeError(w, http.StatusBadRequest, "Invalid lead_time_start value")
		return
	}
	if req.MTTRStart != "" && !validStarts[req.MTTRStart] {
		writeError(w, http.StatusBadRequest, "Invalid mttr_start value")
		return
	}

	// Validate incident_rules JSON if provided
	if req.IncidentRules != "" {
		var tmp interface{}
		if err := json.Unmarshal([]byte(req.IncidentRules), &tmp); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid incident_rules JSON")
			return
		}
	}

	// Fill defaults from existing group
	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}
	if req.Name == "" {
		req.Name = group.Name
	}
	if req.AggregationUnit == "" {
		req.AggregationUnit = group.AggregationUnit
	}
	if req.LeadTimeStart == "" {
		req.LeadTimeStart = group.LeadTimeStart
	}
	if req.MTTRStart == "" {
		req.MTTRStart = group.MTTRStart
	}

	if err := h.Store.UpdateGroup(r.Context(), id, req.Name, req.AggregationUnit, req.LeadTimeStart, req.MTTRStart, req.IncidentRules); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update group settings")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Pulls ---

func (h *Handler) ListGroupPulls(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	sortBy := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	prs, total, err := h.Store.ListPullRequestsByGroup(r.Context(), id, db.PullRequestListOpts{
		Page:    page,
		PerPage: perPage,
		SortBy:  sortBy,
		Order:   order,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list PRs")
		return
	}

	if prs == nil {
		prs = []*db.PullRequest{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pulls": prs,
		"total": total,
		"page":  page,
	})
}

// --- Sync ---

func (h *Handler) StartSync(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	if h.Queue == nil {
		writeError(w, http.StatusServiceUnavailable, "Sync unavailable: no GitHub token configured. Set GITHUB_TOKEN or github.token in config.")
		return
	}

	jobID, err := h.Queue.StartSync(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start sync")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]int64{"job_id": jobID})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.Store.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if h.Queue == nil {
		writeError(w, http.StatusServiceUnavailable, "Sync unavailable")
		return
	}

	if err := h.Queue.CancelJob(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

// --- Badge ---

func (h *Handler) GetBadge(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}

	now := time.Now().UTC()
	period := metrics.CalcPeriod(now, group.AggregationUnit)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, period.Start)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	rulesMap := h.buildRulesMap(group)

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           prs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: group.LeadTimeStart,
		MTTRStart:     group.MTTRStart,
		PeriodDays:    period.Days,
	})

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(renderBadgeSVG(result.OverallLevel)))
}

func renderBadgeSVG(level string) string {
	colors := map[string]string{
		"elite":  "#22c55e",
		"high":   "#3b82f6",
		"medium": "#eab308",
		"low":    "#ef4444",
	}
	color := colors[level]
	if color == "" {
		color = "#6b7280"
	}

	label := "DORA"
	value := level
	if level == "elite" {
		value = "Elite"
	} else if level == "high" {
		value = "High"
	} else if level == "medium" {
		value = "Medium"
	} else {
		value = "Low"
	}

	return `<svg xmlns="http://www.w3.org/2000/svg" width="106" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a"><rect width="106" height="20" rx="3" fill="#fff"/></mask>
  <g mask="url(#a)">
    <rect width="46" height="20" fill="#555"/>
    <rect x="46" width="60" height="20" fill="` + color + `"/>
    <rect width="106" height="20" fill="url(#b)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="23" y="15" fill="#010101" fill-opacity=".3">` + label + `</text>
    <text x="23" y="14">` + label + `</text>
    <text x="75" y="15" fill="#010101" fill-opacity=".3">` + value + `</text>
    <text x="75" y="14">` + value + `</text>
  </g>
</svg>`
}

// --- Export ---

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	group, err := h.Store.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found")
		return
	}

	now := time.Now().UTC()
	period := metrics.CalcPeriod(now, group.AggregationUnit)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, period.Start)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	rulesMap := h.buildRulesMap(group)

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           prs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: group.LeadTimeStart,
		MTTRStart:     group.MTTRStart,
		PeriodDays:    period.Days,
	})

	zipData, err := buildExportZIP(group, prs, result, rulesMap, group.LeadTimeStart)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate export")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fourkeys-export.zip\"")
	w.Write(zipData)
}

// --- Helpers ---

func parseID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

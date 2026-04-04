package api

import (
	"encoding/json"
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
	Store    db.Store
	GitHub   *gh.Client
	Queue    *jobs.Queue
	Logger   *zap.Logger
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
	// Fetch from GitHub API
	repoInfos, err := h.GitHub.ListAccessibleRepos(r.Context())
	if err != nil {
		h.Logger.Error("failed to list repos", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	// Upsert into DB and return with IDs
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
		PeriodDays:    repo.PeriodDays,
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
	Name       string  `json:"name"`
	PeriodDays int     `json:"period_days"`
	RepoIDs    []int64 `json:"repo_ids"`
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
	if req.PeriodDays <= 0 {
		req.PeriodDays = 30
	}
	if len(req.RepoIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one repo is required")
		return
	}

	id, err := h.Store.CreateGroup(r.Context(), req.Name, req.PeriodDays, req.RepoIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create group")
		return
	}

	group, _ := h.Store.GetGroup(r.Context(), id)
	writeJSON(w, http.StatusCreated, group)
}

type updateGroupRequest struct {
	Name       string `json:"name"`
	PeriodDays int    `json:"period_days"`
}

func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.Store.UpdateGroup(r.Context(), id, req.Name, req.PeriodDays); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update group")
		return
	}

	group, _ := h.Store.GetGroup(r.Context(), id)
	writeJSON(w, http.StatusOK, group)
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	if err := h.Store.DeleteGroup(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Metrics ---

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

	since := time.Now().UTC().AddDate(0, 0, -group.PeriodDays)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	// Build repo rules map
	rulesMap := make(map[int64]metrics.IncidentRules)
	for _, repo := range group.Repos {
		rulesMap[repo.ID] = metrics.ParseIncidentRules(repo.IncidentRules.String)
	}

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           prs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: group.Repos[0].LeadTimeStart, // Use first repo's setting as group default
		PeriodDays:    group.PeriodDays,
	})

	writeJSON(w, http.StatusOK, result)
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

	since := time.Now().UTC().AddDate(0, 0, -group.PeriodDays)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	rulesMap := make(map[int64]metrics.IncidentRules)
	for _, repo := range group.Repos {
		rulesMap[repo.ID] = metrics.ParseIncidentRules(repo.IncidentRules.String)
	}

	leadTimeStart := "first_commit_at"
	if len(group.Repos) > 0 {
		leadTimeStart = group.Repos[0].LeadTimeStart
	}

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           prs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: leadTimeStart,
		PeriodDays:    group.PeriodDays,
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

	since := time.Now().UTC().AddDate(0, 0, -group.PeriodDays)
	prs, err := h.Store.GetMergedPRsByGroup(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get PRs")
		return
	}

	rulesMap := make(map[int64]metrics.IncidentRules)
	for _, repo := range group.Repos {
		rulesMap[repo.ID] = metrics.ParseIncidentRules(repo.IncidentRules.String)
	}

	leadTimeStart := "first_commit_at"
	if len(group.Repos) > 0 {
		leadTimeStart = group.Repos[0].LeadTimeStart
	}

	result := metrics.Calculate(metrics.CalculateInput{
		PRs:           prs,
		RepoRulesMap:  rulesMap,
		LeadTimeStart: leadTimeStart,
		PeriodDays:    group.PeriodDays,
	})

	zipData, err := buildExportZIP(group, prs, result, rulesMap, leadTimeStart)
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

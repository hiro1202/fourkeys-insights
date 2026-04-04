package metrics

import (
	"sort"
	"time"

	"github.com/hiro1202/fourkeys-insights/internal/db"
)

// FourKeysResult holds the calculated Four Keys metrics for a group.
type FourKeysResult struct {
	// Lead Time for Changes (median, in hours)
	LeadTimeHours float64 `json:"lead_time_hours"`
	// Deploy Frequency (merges per day)
	DeployFrequency float64 `json:"deploy_frequency"`
	// Change Failure Rate (percentage)
	ChangeFailureRate float64 `json:"change_failure_rate"`
	// Mean Time to Restore (median of incident PR lead times, in hours)
	MTTRHours *float64 `json:"mttr_hours"` // nil when no incidents
	// DORA level for each metric
	LeadTimeLevel          string  `json:"lead_time_level"`
	DeployFrequencyLevel   string  `json:"deploy_frequency_level"`
	ChangeFailureRateLevel string  `json:"change_failure_rate_level"`
	MTTRLevel              *string `json:"mttr_level"` // nil when no incidents
	// Overall DORA level (lowest of all four)
	OverallLevel string `json:"overall_level"`
	// Metadata
	TotalPRs      int `json:"total_prs"`
	IncidentPRs   int `json:"incident_prs"`
	PeriodDays    int `json:"period_days"`
	FallbackCount int `json:"fallback_count"`
}

// CalculateInput holds the inputs for Four Keys calculation.
type CalculateInput struct {
	PRs           []*db.PullRequest
	RepoRulesMap  map[int64]IncidentRules // repoID -> rules
	LeadTimeStart string
	MTTRStart     string // separate start point for MTTR; falls back to LeadTimeStart if empty
	PeriodDays    int
}

// Calculate computes the Four Keys metrics from a set of PRs.
func Calculate(input CalculateInput) FourKeysResult {
	result := FourKeysResult{
		TotalPRs:   len(input.PRs),
		PeriodDays: input.PeriodDays,
	}

	if len(input.PRs) == 0 {
		result.LeadTimeLevel = "low"
		result.DeployFrequencyLevel = "low"
		result.ChangeFailureRateLevel = "elite"
		result.OverallLevel = "low"
		return result
	}

	// Classify PRs and calculate lead times
	var allLeadTimes []time.Duration
	var incidentLeadTimes []time.Duration
	incidentCount := 0

	mttrStart := input.MTTRStart
	if mttrStart == "" {
		mttrStart = input.LeadTimeStart
	}

	for _, pr := range input.PRs {
		// Lead time
		lt := CalculateLeadTime(pr, input.LeadTimeStart)
		allLeadTimes = append(allLeadTimes, lt.LeadTime)
		if lt.UsedFallback {
			result.FallbackCount++
		}

		// Incident detection
		rules, ok := input.RepoRulesMap[pr.RepoID]
		if !ok {
			rules = DefaultIncidentRules()
		}
		if IsIncident(pr, rules) {
			incidentCount++
			// Use MTTRStart for incident PR lead time (MTTR calculation)
			mttrLt := CalculateLeadTime(pr, mttrStart)
			incidentLeadTimes = append(incidentLeadTimes, mttrLt.LeadTime)
		}
	}

	result.IncidentPRs = incidentCount

	// Lead Time (median)
	medianLT := median(allLeadTimes)
	result.LeadTimeHours = medianLT.Hours()
	result.LeadTimeLevel = classifyLeadTime(medianLT)

	// Deploy Frequency
	if input.PeriodDays > 0 {
		result.DeployFrequency = float64(len(input.PRs)) / float64(input.PeriodDays)
	}
	result.DeployFrequencyLevel = classifyDeployFrequency(result.DeployFrequency)

	// Change Failure Rate
	if len(input.PRs) > 0 {
		result.ChangeFailureRate = float64(incidentCount) / float64(len(input.PRs)) * 100
	}
	result.ChangeFailureRateLevel = classifyChangeFailureRate(result.ChangeFailureRate)

	// MTTR (median of incident lead times)
	if len(incidentLeadTimes) > 0 {
		medianMTTR := median(incidentLeadTimes)
		hours := medianMTTR.Hours()
		result.MTTRHours = &hours
		level := classifyMTTR(medianMTTR)
		result.MTTRLevel = &level
	}

	// Overall level = lowest of all four
	result.OverallLevel = overallLevel(result)

	return result
}

func median(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// DORA Level Thresholds (from 2023 State of DevOps Report)

func classifyLeadTime(d time.Duration) string {
	switch {
	case d < 24*time.Hour:
		return "elite"
	case d < 7*24*time.Hour:
		return "high"
	case d < 30*24*time.Hour:
		return "medium"
	default:
		return "low"
	}
}

func classifyDeployFrequency(perDay float64) string {
	switch {
	case perDay >= 1.0: // multiple per day or daily
		return "elite"
	case perDay >= 1.0/7: // weekly to daily
		return "high"
	case perDay >= 1.0/30: // monthly to weekly
		return "medium"
	default:
		return "low"
	}
}

func classifyChangeFailureRate(pct float64) string {
	switch {
	case pct <= 5:
		return "elite"
	case pct <= 10:
		return "high"
	case pct <= 15:
		return "medium"
	default:
		return "low"
	}
}

func classifyMTTR(d time.Duration) string {
	switch {
	case d < 1*time.Hour:
		return "elite"
	case d < 24*time.Hour:
		return "high"
	case d < 7*24*time.Hour:
		return "medium"
	default:
		return "low"
	}
}

var levelOrder = map[string]int{"elite": 3, "high": 2, "medium": 1, "low": 0}

func overallLevel(r FourKeysResult) string {
	lowest := "elite"
	check := func(level string) {
		if levelOrder[level] < levelOrder[lowest] {
			lowest = level
		}
	}
	check(r.LeadTimeLevel)
	check(r.DeployFrequencyLevel)
	check(r.ChangeFailureRateLevel)
	if r.MTTRLevel != nil {
		check(*r.MTTRLevel)
	}
	return lowest
}

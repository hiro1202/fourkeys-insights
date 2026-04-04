package metrics

import (
	"time"

	"github.com/hiro1202/fourkeys-insights/internal/db"
)

// LeadTimeResult holds the lead time calculation for a single PR.
type LeadTimeResult struct {
	PRNumber     int
	LeadTime     time.Duration
	StartPoint   string // which start point was actually used
	UsedFallback bool   // true if the selected start point wasn't available
}

// CalculateLeadTime computes lead time for a single PR.
// Follows the fallback chain defined in DESIGN.md:
//   - first_commit_at: no fallback needed (pr_created_at as last resort)
//   - issue.created_at -> first_commit_at -> pr_created_at
//   - pr_created_at: always available
func CalculateLeadTime(pr *db.PullRequest, selectedStart string) LeadTimeResult {
	result := LeadTimeResult{PRNumber: pr.PRNumber}

	var startTime time.Time

	switch selectedStart {
	case "issue.created_at":
		if pr.LinkedIssueCreatedAt.Valid {
			startTime = pr.LinkedIssueCreatedAt.Time
			result.StartPoint = "issue.created_at"
		} else if pr.FirstCommitAt.Valid {
			startTime = pr.FirstCommitAt.Time
			result.StartPoint = "first_commit_at"
			result.UsedFallback = true
		} else {
			startTime = pr.CreatedAt
			result.StartPoint = "pr_created_at"
			result.UsedFallback = true
		}

	case "first_commit_at":
		if pr.FirstCommitAt.Valid {
			startTime = pr.FirstCommitAt.Time
			result.StartPoint = "first_commit_at"
		} else {
			startTime = pr.CreatedAt
			result.StartPoint = "pr_created_at"
			result.UsedFallback = true
		}

	case "pr_created_at":
		startTime = pr.CreatedAt
		result.StartPoint = "pr_created_at"

	default:
		// Default to first_commit_at
		if pr.FirstCommitAt.Valid {
			startTime = pr.FirstCommitAt.Time
			result.StartPoint = "first_commit_at"
		} else {
			startTime = pr.CreatedAt
			result.StartPoint = "pr_created_at"
			result.UsedFallback = true
		}
	}

	result.LeadTime = pr.MergedAt.Sub(startTime)
	if result.LeadTime < 0 {
		result.LeadTime = 0
	}

	return result
}

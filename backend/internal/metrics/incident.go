package metrics

import (
	"encoding/json"
	"strings"

	"github.com/hiro1202/fourkeys-insights/internal/db"
)

// IncidentRules defines the rules for detecting incident PRs.
// Evaluated at query time, not persisted in PR data.
type IncidentRules struct {
	TitleKeywords  []string `json:"title_keywords"`
	BranchKeywords []string `json:"branch_keywords"`
	Labels         []string `json:"labels"`
}

// DefaultIncidentRules returns the default incident detection rules.
func DefaultIncidentRules() IncidentRules {
	return IncidentRules{
		TitleKeywords:  []string{"revert", "hotfix"},
		BranchKeywords: []string{"hotfix"},
		Labels:         []string{"incident", "bug"},
	}
}

// ParseIncidentRules parses JSON rules from the repo's incident_rules column.
// Returns default rules if the input is empty or invalid.
func ParseIncidentRules(raw string) IncidentRules {
	if raw == "" {
		return DefaultIncidentRules()
	}
	var rules IncidentRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return DefaultIncidentRules()
	}
	return rules
}

// IsIncident evaluates whether a PR is an incident based on the given rules.
// Logic: title matches ANY keyword OR branch matches ANY keyword OR labels contain ANY label.
func IsIncident(pr *db.PullRequest, rules IncidentRules) bool {
	title := strings.ToLower(pr.Title)
	for _, kw := range rules.TitleKeywords {
		if strings.Contains(title, strings.ToLower(kw)) {
			return true
		}
	}

	if pr.BranchName.Valid {
		branch := strings.ToLower(pr.BranchName.String)
		for _, kw := range rules.BranchKeywords {
			if strings.Contains(branch, strings.ToLower(kw)) {
				return true
			}
		}
	}

	if pr.Labels.Valid {
		var prLabels []string
		if err := json.Unmarshal([]byte(pr.Labels.String), &prLabels); err == nil {
			for _, prLabel := range prLabels {
				for _, ruleLabel := range rules.Labels {
					if strings.EqualFold(prLabel, ruleLabel) {
						return true
					}
				}
			}
		}
	}

	return false
}

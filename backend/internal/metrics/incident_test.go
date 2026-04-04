package metrics

import (
	"testing"

	"github.com/hiro1202/fourkeys-insights/internal/db"
)

func TestIsIncident(t *testing.T) {
	rules := DefaultIncidentRules()

	tests := []struct {
		name string
		pr   *db.PullRequest
		want bool
	}{
		{
			name: "revert in title",
			pr:   &db.PullRequest{Title: "Revert: fix login"},
			want: true,
		},
		{
			name: "hotfix in title",
			pr:   &db.PullRequest{Title: "hotfix for broken deploy"},
			want: true,
		},
		{
			name: "hotfix in branch",
			pr: &db.PullRequest{
				Title:      "Fix auth",
				BranchName: db.NewJSONNullString("hotfix/auth-fix", true),
			},
			want: true,
		},
		{
			name: "bug label",
			pr: &db.PullRequest{
				Title:  "Update handler",
				Labels: db.NewJSONNullString(`["bug","enhancement"]`, true),
			},
			want: true,
		},
		{
			name: "incident label",
			pr: &db.PullRequest{
				Title:  "Fix outage",
				Labels: db.NewJSONNullString(`["incident"]`, true),
			},
			want: true,
		},
		{
			name: "normal PR",
			pr: &db.PullRequest{
				Title:      "Add new feature",
				BranchName: db.NewJSONNullString("feature/new-thing", true),
				Labels:     db.NewJSONNullString(`["enhancement"]`, true),
			},
			want: false,
		},
		{
			name: "no labels no branch",
			pr:   &db.PullRequest{Title: "Regular update"},
			want: false,
		},
		{
			name: "case insensitive title",
			pr:   &db.PullRequest{Title: "HOTFIX: critical fix"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIncident(tt.pr, rules)
			if got != tt.want {
				t.Errorf("IsIncident() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIncidentRules(t *testing.T) {
	// Valid JSON
	rules := ParseIncidentRules(`{"title_keywords":["rollback"],"branch_keywords":[],"labels":["p0"]}`)
	if len(rules.TitleKeywords) != 1 || rules.TitleKeywords[0] != "rollback" {
		t.Errorf("unexpected title_keywords: %v", rules.TitleKeywords)
	}

	// Empty returns defaults
	defaults := ParseIncidentRules("")
	if len(defaults.TitleKeywords) != 2 {
		t.Errorf("expected 2 default title_keywords, got %d", len(defaults.TitleKeywords))
	}

	// Invalid JSON returns defaults
	invalid := ParseIncidentRules("{bad json}")
	if len(invalid.TitleKeywords) != 2 {
		t.Errorf("expected defaults on invalid JSON, got %d keywords", len(invalid.TitleKeywords))
	}
}

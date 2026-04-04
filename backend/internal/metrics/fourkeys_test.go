package metrics

import (
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/hiro1202/fourkeys-insights/internal/db"
)

func TestCalculateBasic(t *testing.T) {
	now := time.Now().UTC()
	prs := []*db.PullRequest{
		{
			RepoID:        1,
			PRNumber:      1,
			Title:         "Add feature A",
			FirstCommitAt: sql.NullTime{Time: now.Add(-48 * time.Hour), Valid: true},
			CreatedAt:     now.Add(-47 * time.Hour),
			MergedAt:      now.Add(-24 * time.Hour),
		},
		{
			RepoID:        1,
			PRNumber:      2,
			Title:         "Add feature B",
			FirstCommitAt: sql.NullTime{Time: now.Add(-72 * time.Hour), Valid: true},
			CreatedAt:     now.Add(-70 * time.Hour),
			MergedAt:      now.Add(-48 * time.Hour),
		},
		{
			RepoID:        1,
			PRNumber:      3,
			Title:         "Revert feature A",
			FirstCommitAt: sql.NullTime{Time: now.Add(-12 * time.Hour), Valid: true},
			CreatedAt:     now.Add(-11 * time.Hour),
			MergedAt:      now.Add(-6 * time.Hour),
		},
	}

	result := Calculate(CalculateInput{
		PRs:           prs,
		RepoRulesMap:  map[int64]IncidentRules{1: DefaultIncidentRules()},
		LeadTimeStart: "first_commit_at",
		PeriodDays:    30,
	})

	// 3 PRs in 30 days = 0.1/day
	if result.TotalPRs != 3 {
		t.Errorf("expected 3 total PRs, got %d", result.TotalPRs)
	}
	if result.IncidentPRs != 1 {
		t.Errorf("expected 1 incident PR, got %d", result.IncidentPRs)
	}

	// Deploy frequency: 3/30 = 0.1
	expectedDF := 0.1
	if math.Abs(result.DeployFrequency-expectedDF) > 0.01 {
		t.Errorf("expected deploy frequency ~%.2f, got %.2f", expectedDF, result.DeployFrequency)
	}

	// Change failure rate: 1/3 * 100 = 33.3%
	expectedCFR := 33.33
	if math.Abs(result.ChangeFailureRate-expectedCFR) > 1.0 {
		t.Errorf("expected CFR ~%.1f%%, got %.1f%%", expectedCFR, result.ChangeFailureRate)
	}
	if result.ChangeFailureRateLevel != "low" {
		t.Errorf("expected CFR level 'low', got '%s'", result.ChangeFailureRateLevel)
	}

	// MTTR should be set (1 incident PR)
	if result.MTTRHours == nil {
		t.Fatal("expected MTTR to be set")
	}

	// Lead time median: sorted lead times = [6h, 24h, 24h], median = 24h
	if result.LeadTimeHours < 1 {
		t.Errorf("expected positive lead time, got %.1f", result.LeadTimeHours)
	}
}

func TestCalculateZeroPRs(t *testing.T) {
	result := Calculate(CalculateInput{
		PRs:           nil,
		RepoRulesMap:  map[int64]IncidentRules{},
		LeadTimeStart: "first_commit_at",
		PeriodDays:    30,
	})

	if result.TotalPRs != 0 {
		t.Errorf("expected 0 total PRs, got %d", result.TotalPRs)
	}
	if result.MTTRHours != nil {
		t.Errorf("expected nil MTTR, got %v", result.MTTRHours)
	}
	if result.ChangeFailureRateLevel != "elite" {
		t.Errorf("expected CFR level 'elite' for 0 PRs, got '%s'", result.ChangeFailureRateLevel)
	}
}

func TestCalculateNoIncidents(t *testing.T) {
	now := time.Now().UTC()
	prs := []*db.PullRequest{
		{
			RepoID:        1,
			PRNumber:      1,
			Title:         "Add feature",
			FirstCommitAt: sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true},
			CreatedAt:     now.Add(-23 * time.Hour),
			MergedAt:      now,
		},
	}

	result := Calculate(CalculateInput{
		PRs:           prs,
		RepoRulesMap:  map[int64]IncidentRules{1: DefaultIncidentRules()},
		LeadTimeStart: "first_commit_at",
		PeriodDays:    30,
	})

	if result.IncidentPRs != 0 {
		t.Errorf("expected 0 incidents, got %d", result.IncidentPRs)
	}
	if result.MTTRHours != nil {
		t.Error("expected nil MTTR for zero incidents")
	}
	if result.ChangeFailureRate != 0 {
		t.Errorf("expected 0%% CFR, got %.1f%%", result.ChangeFailureRate)
	}
}

func TestCalculateFallback(t *testing.T) {
	now := time.Now().UTC()
	// PR without first_commit_at, using first_commit_at selection
	prs := []*db.PullRequest{
		{
			RepoID:    1,
			PRNumber:  1,
			Title:     "Feature",
			CreatedAt: now.Add(-24 * time.Hour),
			MergedAt:  now,
		},
	}

	result := Calculate(CalculateInput{
		PRs:           prs,
		RepoRulesMap:  map[int64]IncidentRules{1: DefaultIncidentRules()},
		LeadTimeStart: "first_commit_at",
		PeriodDays:    30,
	})

	if result.FallbackCount != 1 {
		t.Errorf("expected 1 fallback, got %d", result.FallbackCount)
	}
}

func TestDORALevelClassification(t *testing.T) {
	// Lead time thresholds
	tests := []struct {
		hours float64
		want  string
	}{
		{0.5, "elite"},  // 30 min < 1 day
		{23, "elite"},   // 23h < 1 day
		{25, "high"},    // 25h (1-7 days)
		{150, "high"},   // ~6.25 days
		{200, "medium"}, // ~8.3 days (1 week - 1 month)
		{800, "low"},    // ~33 days
	}

	for _, tt := range tests {
		d := time.Duration(tt.hours * float64(time.Hour))
		got := classifyLeadTime(d)
		if got != tt.want {
			t.Errorf("classifyLeadTime(%.0fh) = %s, want %s", tt.hours, got, tt.want)
		}
	}
}

func TestOverallLevel(t *testing.T) {
	// If all elite, overall should be elite
	mttrLevel := "elite"
	r := FourKeysResult{
		LeadTimeLevel:          "elite",
		DeployFrequencyLevel:   "elite",
		ChangeFailureRateLevel: "elite",
		MTTRLevel:              &mttrLevel,
	}
	if overallLevel(r) != "elite" {
		t.Errorf("expected overall 'elite', got '%s'", overallLevel(r))
	}

	// One low drags everything down
	r.DeployFrequencyLevel = "low"
	if overallLevel(r) != "low" {
		t.Errorf("expected overall 'low', got '%s'", overallLevel(r))
	}
}

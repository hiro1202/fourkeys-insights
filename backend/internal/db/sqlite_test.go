package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestUpsertAndListRepos(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	id, err := store.UpsertRepo(ctx, &Repo{
		Owner: "owner", Name: "repo", FullName: "owner/repo", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	// Upsert same repo should return same id
	id2, err := store.UpsertRepo(ctx, &Repo{
		Owner: "owner", Name: "repo", FullName: "owner/repo", DefaultBranch: "develop",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatalf("expected same id %d, got %d", id, id2)
	}

	repos, err := store.ListRepos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].DefaultBranch != "develop" {
		t.Fatalf("expected branch 'develop', got '%s'", repos[0].DefaultBranch)
	}
}

func TestRepoSettings(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	id, _ := store.UpsertRepo(ctx, &Repo{
		Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main",
	})

	err := store.UpdateRepoSettings(ctx, id, &RepoSettings{
		IncidentRules: `{"title_keywords":["revert"]}`,
		LeadTimeStart: "pr_created_at",
		PeriodDays:    60,
	})
	if err != nil {
		t.Fatal(err)
	}

	repo, err := store.GetRepo(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if repo.LeadTimeStart != "pr_created_at" {
		t.Fatalf("expected lead_time_start 'pr_created_at', got '%s'", repo.LeadTimeStart)
	}
	if repo.PeriodDays != 60 {
		t.Fatalf("expected period_days 60, got %d", repo.PeriodDays)
	}
}

func TestGroupCRUD(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	r1, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "a", FullName: "o/a", DefaultBranch: "main"})
	r2, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "b", FullName: "o/b", DefaultBranch: "main"})

	groupID, err := store.CreateGroup(ctx, "backend", 30, []int64{r1, r2})
	if err != nil {
		t.Fatal(err)
	}

	groups, err := store.ListGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Repos) != 2 {
		t.Fatalf("expected 2 repos in group, got %d", len(groups[0].Repos))
	}

	// Update
	err = store.UpdateGroup(ctx, groupID, "backend-team", 90)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := store.GetGroup(ctx, groupID)
	if g.Name != "backend-team" {
		t.Fatalf("expected name 'backend-team', got '%s'", g.Name)
	}

	// Delete (cascade should remove members)
	err = store.DeleteGroup(ctx, groupID)
	if err != nil {
		t.Fatal(err)
	}
	groups, _ = store.ListGroups(ctx)
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups after delete, got %d", len(groups))
	}
}

func TestPullRequestUpsertAndList(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	repoID, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	groupID, _ := store.CreateGroup(ctx, "team", 30, []int64{repoID})

	now := time.Now().UTC().Truncate(time.Second)
	pr := &PullRequest{
		RepoID:    repoID,
		PRNumber:  42,
		Title:     "Fix login bug",
		BranchName: sql.NullString{String: "hotfix/login", Valid: true},
		Additions: 10,
		Deletions: 5,
		CreatedAt: now.Add(-24 * time.Hour),
		MergedAt:  now,
	}
	if err := store.UpsertPullRequest(ctx, pr); err != nil {
		t.Fatal(err)
	}

	// Upsert again with updated title
	pr.Title = "Fix login bug (updated)"
	if err := store.UpsertPullRequest(ctx, pr); err != nil {
		t.Fatal(err)
	}

	prs, total, err := store.ListPullRequestsByGroup(ctx, groupID, PullRequestListOpts{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("expected 1 PR, got %d", total)
	}
	if prs[0].Title != "Fix login bug (updated)" {
		t.Fatalf("expected updated title, got '%s'", prs[0].Title)
	}
	if prs[0].RepoFullName != "o/r" {
		t.Fatalf("expected repo_full_name 'o/r', got '%s'", prs[0].RepoFullName)
	}
}

func TestGetMergedPRsByGroup(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	repoID, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	groupID, _ := store.CreateGroup(ctx, "team", 30, []int64{repoID})

	now := time.Now().UTC().Truncate(time.Second)

	// Old PR (outside period)
	store.UpsertPullRequest(ctx, &PullRequest{
		RepoID: repoID, PRNumber: 1, Title: "Old PR",
		CreatedAt: now.Add(-60 * 24 * time.Hour), MergedAt: now.Add(-31 * 24 * time.Hour),
	})
	// Recent PR
	store.UpsertPullRequest(ctx, &PullRequest{
		RepoID: repoID, PRNumber: 2, Title: "Recent PR",
		CreatedAt: now.Add(-2 * 24 * time.Hour), MergedAt: now.Add(-1 * 24 * time.Hour),
	})

	since := now.Add(-30 * 24 * time.Hour)
	prs, err := store.GetMergedPRsByGroup(ctx, groupID, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 recent PR, got %d", len(prs))
	}
	if prs[0].PRNumber != 2 {
		t.Fatalf("expected PR #2, got #%d", prs[0].PRNumber)
	}
}

func TestJobLifecycle(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	repoID, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	groupID, _ := store.CreateGroup(ctx, "team", 30, []int64{repoID})

	jobID, err := store.CreateJob(ctx, groupID)
	if err != nil {
		t.Fatal(err)
	}

	job, _ := store.GetJob(ctx, jobID)
	if job.Status != "fetching" {
		t.Fatalf("expected status 'fetching', got '%s'", job.Status)
	}

	// Update progress
	err = store.UpdateJobStatus(ctx, jobID, "fetching", &JobProgress{Fetched: 10, Total: 50, CurrentRepo: "o/r"}, "")
	if err != nil {
		t.Fatal(err)
	}

	// Complete
	err = store.UpdateJobStatus(ctx, jobID, "complete", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	job, _ = store.GetJob(ctx, jobID)
	if job.Status != "complete" {
		t.Fatalf("expected status 'complete', got '%s'", job.Status)
	}
	if !job.CompletedAt.Valid {
		t.Fatal("expected completed_at to be set")
	}
}

func TestRecoverInterruptedJobs(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	repoID, _ := store.UpsertRepo(ctx, &Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	groupID, _ := store.CreateGroup(ctx, "team", 30, []int64{repoID})

	// Create two jobs in "fetching" state
	j1, _ := store.CreateJob(ctx, groupID)
	j2, _ := store.CreateJob(ctx, groupID)

	// Simulate recovery
	err := store.RecoverInterruptedJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}

	job1, _ := store.GetJob(ctx, j1)
	job2, _ := store.GetJob(ctx, j2)
	if job1.Status != "failed" || job2.Status != "failed" {
		t.Fatalf("expected both jobs to be 'failed', got '%s' and '%s'", job1.Status, job2.Status)
	}
	if !job1.Error.Valid || job1.Error.String != "Process restarted" {
		t.Fatal("expected error message 'Process restarted'")
	}
}

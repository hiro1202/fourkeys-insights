package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Repos ---

func (s *SQLiteStore) UpsertRepo(ctx context.Context, repo *Repo) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO repos (owner, name, full_name, default_branch)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(full_name) DO UPDATE SET
			default_branch = excluded.default_branch,
			updated_at = CURRENT_TIMESTAMP
	`, repo.Owner, repo.Name, repo.FullName, repo.DefaultBranch)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// ON CONFLICT doesn't update LastInsertId, so fetch it
	if id == 0 {
		err = s.db.QueryRowContext(ctx, "SELECT id FROM repos WHERE full_name = ?", repo.FullName).Scan(&id)
	}
	return id, err
}

func (s *SQLiteStore) ListRepos(ctx context.Context) ([]*Repo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, owner, name, full_name, default_branch, etag, incident_rules, lead_time_start, mttr_start, created_at, updated_at FROM repos ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*Repo
	for rows.Next() {
		r := &Repo{}
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.DefaultBranch, &r.ETag, &r.IncidentRules, &r.LeadTimeStart, &r.MTTRStart, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *SQLiteStore) GetRepo(ctx context.Context, id int64) (*Repo, error) {
	r := &Repo{}
	err := s.db.QueryRowContext(ctx, `SELECT id, owner, name, full_name, default_branch, etag, incident_rules, lead_time_start, mttr_start, created_at, updated_at FROM repos WHERE id = ?`, id).
		Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.DefaultBranch, &r.ETag, &r.IncidentRules, &r.LeadTimeStart, &r.MTTRStart, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *SQLiteStore) UpdateRepoSettings(ctx context.Context, id int64, settings *RepoSettings) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repos SET incident_rules = ?, lead_time_start = ?, mttr_start = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, settings.IncidentRules, settings.LeadTimeStart, settings.MTTRStart, id)
	return err
}

func (s *SQLiteStore) UpdateRepoETag(ctx context.Context, id int64, etag string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE repos SET etag = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, etag, id)
	return err
}

// --- Groups ---

func (s *SQLiteStore) ListGroups(ctx context.Context) ([]*Group, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, aggregation_unit, lead_time_start, mttr_start, incident_rules, created_at FROM repo_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		g := &Group{}
		var incidentRules sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &g.AggregationUnit, &g.LeadTimeStart, &g.MTTRStart, &incidentRules, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.IncidentRules = incidentRules.String
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, g := range groups {
		g.Repos, err = s.getGroupRepos(ctx, g.ID)
		if err != nil {
			return nil, err
		}
	}
	return groups, nil
}

func (s *SQLiteStore) GetGroup(ctx context.Context, id int64) (*Group, error) {
	g := &Group{}
	var incidentRules sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id, name, aggregation_unit, lead_time_start, mttr_start, incident_rules, created_at FROM repo_groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.AggregationUnit, &g.LeadTimeStart, &g.MTTRStart, &incidentRules, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	g.IncidentRules = incidentRules.String
	g.Repos, err = s.getGroupRepos(ctx, id)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (s *SQLiteStore) getGroupRepos(ctx context.Context, groupID int64) ([]*Repo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.owner, r.name, r.full_name, r.default_branch, r.etag, r.incident_rules, r.lead_time_start, r.mttr_start, r.created_at, r.updated_at
		FROM repos r
		JOIN repo_group_members m ON m.repo_id = r.id
		WHERE m.group_id = ?
		ORDER BY r.full_name
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*Repo
	for rows.Next() {
		r := &Repo{}
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.DefaultBranch, &r.ETag, &r.IncidentRules, &r.LeadTimeStart, &r.MTTRStart, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *SQLiteStore) CreateGroup(ctx context.Context, name string, aggregationUnit string, repoIDs []int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if aggregationUnit == "" {
		aggregationUnit = "weekly"
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO repo_groups (name, aggregation_unit) VALUES (?, ?)`, name, aggregationUnit)
	if err != nil {
		return 0, err
	}
	groupID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, repoID := range repoIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO repo_group_members (group_id, repo_id) VALUES (?, ?)`, groupID, repoID); err != nil {
			return 0, err
		}
	}

	return groupID, tx.Commit()
}

func (s *SQLiteStore) UpdateGroup(ctx context.Context, id int64, name string, aggregationUnit string, leadTimeStart string, mttrStart string, incidentRules string) error {
	var irVal interface{}
	if incidentRules != "" {
		irVal = incidentRules
	}
	_, err := s.db.ExecContext(ctx, `UPDATE repo_groups SET name = ?, aggregation_unit = ?, lead_time_start = ?, mttr_start = ?, incident_rules = ? WHERE id = ?`,
		name, aggregationUnit, leadTimeStart, mttrStart, irVal, id)
	return err
}

func (s *SQLiteStore) DeleteGroup(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM jobs WHERE group_id = ?`, id); err != nil {
		return fmt.Errorf("delete jobs: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM repo_groups WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return tx.Commit()
}

// --- Pull Requests ---

func (s *SQLiteStore) UpsertPullRequest(ctx context.Context, pr *PullRequest) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pull_requests (repo_id, pr_number, title, branch_name, labels, body, additions, deletions, first_commit_at, created_at, merged_at, linked_issue_number, linked_issue_created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, pr_number) DO UPDATE SET
			title = excluded.title,
			branch_name = excluded.branch_name,
			labels = excluded.labels,
			body = excluded.body,
			additions = excluded.additions,
			deletions = excluded.deletions,
			first_commit_at = excluded.first_commit_at,
			linked_issue_number = excluded.linked_issue_number,
			linked_issue_created_at = excluded.linked_issue_created_at
	`, pr.RepoID, pr.PRNumber, pr.Title, pr.BranchName, pr.Labels, pr.Body,
		pr.Additions, pr.Deletions, pr.FirstCommitAt, pr.CreatedAt, pr.MergedAt,
		pr.LinkedIssueNumber, pr.LinkedIssueCreatedAt)
	return err
}

func (s *SQLiteStore) ListPullRequestsByGroup(ctx context.Context, groupID int64, opts PullRequestListOpts) ([]*PullRequest, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = 50
	}

	// Count total
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pull_requests p
		JOIN repo_group_members m ON m.repo_id = p.repo_id
		WHERE m.group_id = ?
	`, groupID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Validate sort
	sortCol := "p.merged_at"
	allowedSorts := map[string]string{
		"merged_at":  "p.merged_at",
		"created_at": "p.created_at",
		"pr_number":  "p.pr_number",
		"title":      "p.title",
	}
	if col, ok := allowedSorts[opts.SortBy]; ok {
		sortCol = col
	}
	order := "DESC"
	if strings.EqualFold(opts.Order, "asc") {
		order = "ASC"
	}

	offset := (opts.Page - 1) * opts.PerPage
	query := fmt.Sprintf(`
		SELECT p.id, p.repo_id, p.pr_number, p.title, p.branch_name, p.labels, p.body,
			p.additions, p.deletions, p.first_commit_at, p.created_at, p.merged_at,
			p.linked_issue_number, p.linked_issue_created_at, r.full_name
		FROM pull_requests p
		JOIN repo_group_members m ON m.repo_id = p.repo_id
		JOIN repos r ON r.id = p.repo_id
		WHERE m.group_id = ?
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, sortCol, order)

	rows, err := s.db.QueryContext(ctx, query, groupID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var prs []*PullRequest
	for rows.Next() {
		pr := &PullRequest{}
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.PRNumber, &pr.Title, &pr.BranchName, &pr.Labels, &pr.Body,
			&pr.Additions, &pr.Deletions, &pr.FirstCommitAt, &pr.CreatedAt, &pr.MergedAt,
			&pr.LinkedIssueNumber, &pr.LinkedIssueCreatedAt, &pr.RepoFullName); err != nil {
			return nil, 0, err
		}
		prs = append(prs, pr)
	}
	return prs, total, rows.Err()
}

func (s *SQLiteStore) GetMergedPRsByGroup(ctx context.Context, groupID int64, since time.Time) ([]*PullRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.repo_id, p.pr_number, p.title, p.branch_name, p.labels, p.body,
			p.additions, p.deletions, p.first_commit_at, p.created_at, p.merged_at,
			p.linked_issue_number, p.linked_issue_created_at, r.full_name
		FROM pull_requests p
		JOIN repo_group_members m ON m.repo_id = p.repo_id
		JOIN repos r ON r.id = p.repo_id
		WHERE m.group_id = ? AND p.merged_at >= ?
		ORDER BY p.merged_at DESC
	`, groupID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*PullRequest
	for rows.Next() {
		pr := &PullRequest{}
		if err := rows.Scan(&pr.ID, &pr.RepoID, &pr.PRNumber, &pr.Title, &pr.BranchName, &pr.Labels, &pr.Body,
			&pr.Additions, &pr.Deletions, &pr.FirstCommitAt, &pr.CreatedAt, &pr.MergedAt,
			&pr.LinkedIssueNumber, &pr.LinkedIssueCreatedAt, &pr.RepoFullName); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, rows.Err()
}

func (s *SQLiteStore) DeletePullRequestsByRepo(ctx context.Context, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pull_requests WHERE repo_id = ?`, repoID)
	return err
}

// --- Jobs ---

func (s *SQLiteStore) CreateJob(ctx context.Context, groupID int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (group_id, status, started_at) VALUES (?, 'fetching', CURRENT_TIMESTAMP)
	`, groupID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetJob(ctx context.Context, id int64) (*Job, error) {
	j := &Job{}
	err := s.db.QueryRowContext(ctx, `SELECT id, group_id, status, progress, error, started_at, completed_at FROM jobs WHERE id = ?`, id).
		Scan(&j.ID, &j.GroupID, &j.Status, &j.Progress, &j.Error, &j.StartedAt, &j.CompletedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (s *SQLiteStore) GetLatestJobByGroup(ctx context.Context, groupID int64) (*Job, error) {
	j := &Job{}
	err := s.db.QueryRowContext(ctx, `SELECT id, group_id, status, progress, error, started_at, completed_at FROM jobs WHERE group_id = ? ORDER BY id DESC LIMIT 1`, groupID).
		Scan(&j.ID, &j.GroupID, &j.Status, &j.Progress, &j.Error, &j.StartedAt, &j.CompletedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (s *SQLiteStore) UpdateJobStatus(ctx context.Context, id int64, status string, progress *JobProgress, errMsg string) error {
	var progressJSON JSONNullString
	if progress != nil {
		b, err := json.Marshal(progress)
		if err != nil {
			return err
		}
		progressJSON = NewJSONNullString(string(b), true)
	}

	var errorVal JSONNullString
	if errMsg != "" {
		errorVal = NewJSONNullString(errMsg, true)
	}

	completedAt := JSONNullTime{}
	if status == "complete" || status == "failed" || status == "cancelled" {
		completedAt = NewJSONNullTime(time.Now().UTC(), true)
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, progress = ?, error = ?, completed_at = ? WHERE id = ?
	`, status, progressJSON, errorVal, completedAt, id)
	return err
}

func (s *SQLiteStore) RecoverInterruptedJobs(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE jobs SET status = 'failed', error = 'Process restarted', completed_at = CURRENT_TIMESTAMP
		WHERE status IN ('fetching', 'computing')
	`)
	return err
}

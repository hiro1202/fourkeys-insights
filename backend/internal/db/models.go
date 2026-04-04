package db

import (
	"database/sql"
	"time"
)

type Repo struct {
	ID            int64          `json:"id"`
	Owner         string         `json:"owner"`
	Name          string         `json:"name"`
	FullName      string         `json:"full_name"`
	DefaultBranch string         `json:"default_branch"`
	ETag          sql.NullString `json:"-"`
	IncidentRules sql.NullString `json:"incident_rules"`
	LeadTimeStart string         `json:"lead_time_start"`
	PeriodDays    int            `json:"period_days"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type RepoSettings struct {
	IncidentRules string `json:"incident_rules"`
	LeadTimeStart string `json:"lead_time_start"`
	PeriodDays    int    `json:"period_days"`
}

type Group struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	PeriodDays int       `json:"period_days"`
	CreatedAt  time.Time `json:"created_at"`
	Repos      []*Repo   `json:"repos,omitempty"`
}

type PullRequest struct {
	ID                   int64          `json:"id"`
	RepoID               int64          `json:"repo_id"`
	PRNumber             int            `json:"pr_number"`
	Title                string         `json:"title"`
	BranchName           sql.NullString `json:"branch_name"`
	Labels               sql.NullString `json:"labels"`
	Body                 sql.NullString `json:"body"`
	Additions            int            `json:"additions"`
	Deletions            int            `json:"deletions"`
	FirstCommitAt        sql.NullTime   `json:"first_commit_at"`
	CreatedAt            time.Time      `json:"created_at"`
	MergedAt             time.Time      `json:"merged_at"`
	LinkedIssueNumber    sql.NullInt64  `json:"linked_issue_number"`
	LinkedIssueCreatedAt sql.NullTime   `json:"linked_issue_created_at"`

	// Joined fields (not in DB)
	RepoFullName string `json:"repo_full_name,omitempty"`
}

type Job struct {
	ID          int64          `json:"id"`
	GroupID     sql.NullInt64  `json:"group_id"`
	Status      string         `json:"status"`
	Progress    sql.NullString `json:"progress"`
	Error       sql.NullString `json:"error"`
	StartedAt   sql.NullTime   `json:"started_at"`
	CompletedAt sql.NullTime   `json:"completed_at"`
}

type JobProgress struct {
	Fetched     int    `json:"fetched"`
	Total       int    `json:"total"`
	CurrentRepo string `json:"current_repo"`
}

type PullRequestListOpts struct {
	Page    int
	PerPage int
	SortBy  string
	Order   string
}

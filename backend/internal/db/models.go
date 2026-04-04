package db

import (
	"time"
)

type Repo struct {
	ID            int64          `json:"id"`
	Owner         string         `json:"owner"`
	Name          string         `json:"name"`
	FullName      string         `json:"full_name"`
	DefaultBranch string         `json:"default_branch"`
	ETag          JSONNullString `json:"-"`
	IncidentRules JSONNullString `json:"incident_rules"`
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
	BranchName           JSONNullString `json:"branch_name"`
	Labels               JSONNullString `json:"labels"`
	Body                 JSONNullString `json:"body"`
	Additions            int            `json:"additions"`
	Deletions            int            `json:"deletions"`
	FirstCommitAt        JSONNullTime   `json:"first_commit_at"`
	CreatedAt            time.Time      `json:"created_at"`
	MergedAt             time.Time      `json:"merged_at"`
	LinkedIssueNumber    JSONNullInt64  `json:"linked_issue_number"`
	LinkedIssueCreatedAt JSONNullTime   `json:"linked_issue_created_at"`

	// Joined fields (not in DB)
	RepoFullName string `json:"repo_full_name,omitempty"`
}

type Job struct {
	ID          int64          `json:"id"`
	GroupID     JSONNullInt64  `json:"group_id"`
	Status      string         `json:"status"`
	Progress    JSONNullString `json:"progress"`
	Error       JSONNullString `json:"error"`
	StartedAt   JSONNullTime   `json:"started_at"`
	CompletedAt JSONNullTime   `json:"completed_at"`
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

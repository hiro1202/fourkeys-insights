package db

import (
	"context"
	"time"
)

// Store defines the data access interface.
// SQLite is the v1 implementation. PostgreSQL planned for Phase 2.
type Store interface {
	// Repos
	UpsertRepo(ctx context.Context, repo *Repo) (int64, error)
	ListRepos(ctx context.Context) ([]*Repo, error)
	GetRepo(ctx context.Context, id int64) (*Repo, error)
	UpdateRepoSettings(ctx context.Context, id int64, settings *RepoSettings) error
	UpdateRepoETag(ctx context.Context, id int64, etag string) error

	// Groups
	ListGroups(ctx context.Context) ([]*Group, error)
	GetGroup(ctx context.Context, id int64) (*Group, error)
	CreateGroup(ctx context.Context, name string, periodDays int, repoIDs []int64) (int64, error)
	UpdateGroup(ctx context.Context, id int64, name string, periodDays int) error
	DeleteGroup(ctx context.Context, id int64) error

	// Pull Requests
	UpsertPullRequest(ctx context.Context, pr *PullRequest) error
	ListPullRequestsByGroup(ctx context.Context, groupID int64, opts PullRequestListOpts) ([]*PullRequest, int, error)
	GetMergedPRsByGroup(ctx context.Context, groupID int64, since time.Time) ([]*PullRequest, error)
	DeletePullRequestsByRepo(ctx context.Context, repoID int64) error

	// Jobs
	CreateJob(ctx context.Context, groupID int64) (int64, error)
	GetJob(ctx context.Context, id int64) (*Job, error)
	UpdateJobStatus(ctx context.Context, id int64, status string, progress *JobProgress, errMsg string) error
	RecoverInterruptedJobs(ctx context.Context) error

	// Lifecycle
	Close() error
}

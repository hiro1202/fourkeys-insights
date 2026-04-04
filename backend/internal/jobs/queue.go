package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hiro1202/fourkeys-insights/internal/db"
	gh "github.com/hiro1202/fourkeys-insights/internal/github"
	"go.uber.org/zap"
)

// Queue manages sync jobs with goroutine-based execution.
type Queue struct {
	store  db.Store
	gh     *gh.Client
	logger *zap.Logger

	mu      sync.Mutex
	running map[int64]context.CancelFunc // jobID -> cancel
}

// NewQueue creates a new job queue.
func NewQueue(store db.Store, ghClient *gh.Client, logger *zap.Logger) *Queue {
	return &Queue{
		store:   store,
		gh:      ghClient,
		logger:  logger,
		running: make(map[int64]context.CancelFunc),
	}
}

// StartSync creates a job and starts syncing PRs for a group in the background.
func (q *Queue) StartSync(groupID int64) (int64, error) {
	jobID, err := q.store.CreateJob(context.Background(), groupID)
	if err != nil {
		return 0, fmt.Errorf("creating job: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	q.mu.Lock()
	q.running[jobID] = cancel
	q.mu.Unlock()

	go q.runSync(ctx, jobID, groupID)

	return jobID, nil
}

// CancelJob cancels a running sync job.
func (q *Queue) CancelJob(jobID int64) error {
	q.mu.Lock()
	cancel, ok := q.running[jobID]
	q.mu.Unlock()

	if !ok {
		return fmt.Errorf("job %d is not running", jobID)
	}

	cancel()
	return nil
}

func (q *Queue) runSync(ctx context.Context, jobID, groupID int64) {
	defer func() {
		q.mu.Lock()
		delete(q.running, jobID)
		q.mu.Unlock()
	}()

	logger := q.logger.With(zap.Int64("job_id", jobID), zap.Int64("group_id", groupID))
	logger.Info("sync started")

	group, err := q.store.GetGroup(ctx, groupID)
	if err != nil {
		q.failJob(jobID, "failed to get group: "+err.Error())
		return
	}

	if len(group.Repos) == 0 {
		q.failJob(jobID, "group has no repos")
		return
	}

	totalRepos := len(group.Repos)
	fetchedRepos := 0

	for _, repo := range group.Repos {
		if ctx.Err() != nil {
			q.store.UpdateJobStatus(context.Background(), jobID, "cancelled", nil, "")
			logger.Info("sync cancelled")
			return
		}

		logger.Info("syncing repo", zap.String("repo", repo.FullName))

		// Update progress
		q.store.UpdateJobStatus(ctx, jobID, "fetching", &db.JobProgress{
			Fetched:     fetchedRepos,
			Total:       totalRepos,
			CurrentRepo: repo.FullName,
		}, "")

		err := q.syncRepo(ctx, repo)
		if err != nil {
			if ctx.Err() != nil {
				q.store.UpdateJobStatus(context.Background(), jobID, "cancelled", nil, "")
				logger.Info("sync cancelled during repo fetch")
				return
			}
			q.failJob(jobID, fmt.Sprintf("syncing %s: %v", repo.FullName, err))
			return
		}

		fetchedRepos++
	}

	// Mark computing
	q.store.UpdateJobStatus(ctx, jobID, "computing", nil, "")

	// Mark complete
	q.store.UpdateJobStatus(ctx, jobID, "complete", nil, "")
	logger.Info("sync complete", zap.Int("repos", totalRepos))
}

func (q *Queue) syncRepo(ctx context.Context, repo *db.Repo) error {
	// Fetch merged PRs with ETag support
	etag := ""
	if repo.ETag.Valid {
		etag = repo.ETag.String
	}

	result, err := q.gh.ListMergedPRs(ctx, repo.Owner, repo.Name, repo.DefaultBranch, etag)
	if err != nil {
		if err == gh.ErrNotModified {
			q.logger.Debug("repo not modified, skipping", zap.String("repo", repo.FullName))
			return nil
		}
		return err
	}

	// Update ETag
	if result.NewETag != "" {
		q.store.UpdateRepoETag(ctx, repo.ID, result.NewETag)
	}

	// Process each PR
	for _, prInfo := range result.PRs {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Get first commit timestamp
		firstCommitAt, err := q.gh.GetFirstCommitAt(ctx, repo.Owner, repo.Name, prInfo.Number)
		if err != nil {
			q.logger.Warn("failed to get first commit", zap.String("repo", repo.FullName), zap.Int("pr", prInfo.Number), zap.Error(err))
		}

		// Parse issue reference from PR body
		var linkedIssueNumber db.JSONNullInt64
		var linkedIssueCreatedAt db.JSONNullTime

		if issueNum := gh.ParseIssueReference(prInfo.Body); issueNum > 0 {
			linkedIssueNumber = db.NewJSONNullInt64(int64(issueNum), true)
			if issueCreatedAt, err := q.gh.GetIssueCreatedAt(ctx, repo.Owner, repo.Name, issueNum); err == nil {
				linkedIssueCreatedAt = db.NewJSONNullTime(issueCreatedAt, true)
			} else {
				q.logger.Warn("failed to get issue created_at", zap.String("repo", repo.FullName), zap.Int("issue", issueNum), zap.Error(err))
			}
		}

		// Labels to JSON
		var labelsJSON db.JSONNullString
		if len(prInfo.Labels) > 0 {
			b, _ := json.Marshal(prInfo.Labels)
			labelsJSON = db.NewJSONNullString(string(b), true)
		}

		pr := &db.PullRequest{
			RepoID:               repo.ID,
			PRNumber:             prInfo.Number,
			Title:                prInfo.Title,
			BranchName:           db.NewJSONNullString(prInfo.Branch, prInfo.Branch != ""),
			Labels:               labelsJSON,
			Body:                 db.NewJSONNullString(prInfo.Body, prInfo.Body != ""),
			Additions:            prInfo.Additions,
			Deletions:            prInfo.Deletions,
			CreatedAt:            prInfo.CreatedAt,
			MergedAt:             prInfo.MergedAt,
			LinkedIssueNumber:    linkedIssueNumber,
			LinkedIssueCreatedAt: linkedIssueCreatedAt,
		}

		if !firstCommitAt.IsZero() {
			pr.FirstCommitAt = db.NewJSONNullTime(firstCommitAt, true)
		}

		if err := q.store.UpsertPullRequest(ctx, pr); err != nil {
			return fmt.Errorf("upserting PR #%d: %w", prInfo.Number, err)
		}
	}

	return nil
}

func (q *Queue) failJob(jobID int64, errMsg string) {
	q.logger.Error("sync failed", zap.Int64("job_id", jobID), zap.String("error", errMsg))
	q.store.UpdateJobStatus(context.Background(), jobID, "failed", nil, errMsg)
}

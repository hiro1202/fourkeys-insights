package github

import (
	"context"
	"time"

	gh "github.com/google/go-github/v62/github"
	"go.uber.org/zap"
)

// GetFirstCommitAt returns the timestamp of the earliest commit in a PR.
// Used for lead time calculation when start point is "first_commit_at".
func (c *Client) GetFirstCommitAt(ctx context.Context, owner, repo string, prNumber int) (time.Time, error) {
	var earliest time.Time
	opts := &gh.ListOptions{PerPage: 100}

	for {
		commits, resp, err := c.client.PullRequests.ListCommits(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return time.Time{}, err
		}

		for _, commit := range commits {
			ts := commit.GetCommit().GetAuthor().GetDate().Time
			if earliest.IsZero() || ts.Before(earliest) {
				earliest = ts
			}
		}

		c.logger.Debug("fetched PR commits",
			zap.String("repo", owner+"/"+repo),
			zap.Int("pr", prNumber),
			zap.Int("count", len(commits)),
		)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return earliest, nil
}

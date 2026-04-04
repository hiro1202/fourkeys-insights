package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	gh "github.com/google/go-github/v62/github"
	"go.uber.org/zap"
)

// PRInfo contains the fields we need from a GitHub pull request.
type PRInfo struct {
	Number    int
	Title     string
	Branch    string
	Labels    []string
	Body      string
	Additions int
	Deletions int
	CreatedAt time.Time
	MergedAt  time.Time
}

// ErrNotModified is returned when the API returns 304 (ETag match).
var ErrNotModified = &notModifiedError{}

type notModifiedError struct{}

func (e *notModifiedError) Error() string { return "not modified (304)" }

// ListMergedPRsResult holds the result of a merged PR list fetch.
type ListMergedPRsResult struct {
	PRs     []PRInfo
	NewETag string
}

// ListMergedPRs fetches all merged PRs for a repo's base branch.
// Supports ETag conditional requests: pass current etag, get ErrNotModified on 304.
func (c *Client) ListMergedPRs(ctx context.Context, owner, repo, baseBranch, etag string) (*ListMergedPRsResult, error) {
	var all []PRInfo
	opts := &gh.PullRequestListOptions{
		State:     "closed",
		Base:      baseBranch,
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	var newETag string
	firstPage := true

	for {
		// Build the request manually for ETag support on first page
		req, err := c.client.NewRequest("GET", "repos/"+owner+"/"+repo+"/pulls", nil)
		if err != nil {
			return nil, err
		}

		// Add query params
		q := req.URL.Query()
		q.Set("state", opts.State)
		q.Set("base", opts.Base)
		q.Set("sort", opts.Sort)
		q.Set("direction", opts.Direction)
		q.Set("per_page", "100")
		if opts.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		req.URL.RawQuery = q.Encode()

		if firstPage && etag != "" {
			req.Header.Set("If-None-Match", etag)
		}

		var prs []*gh.PullRequest
		resp, err := c.client.Do(ctx, req, &prs)

		if err != nil {
			// Check for 304
			if resp != nil && resp.StatusCode == http.StatusNotModified {
				return nil, ErrNotModified
			}
			return nil, err
		}

		if firstPage {
			newETag = resp.Header.Get("ETag")
			firstPage = false
		}

		for _, pr := range prs {
			if pr.GetMergedAt().IsZero() {
				continue // skip non-merged PRs
			}

			var labels []string
			for _, l := range pr.Labels {
				labels = append(labels, l.GetName())
			}

			all = append(all, PRInfo{
				Number:    pr.GetNumber(),
				Title:     pr.GetTitle(),
				Branch:    pr.GetHead().GetRef(),
				Labels:    labels,
				Body:      pr.GetBody(),
				Additions: pr.GetAdditions(),
				Deletions: pr.GetDeletions(),
				CreatedAt: pr.GetCreatedAt().Time,
				MergedAt:  pr.GetMergedAt().Time,
			})
		}

		c.logger.Debug("fetched PRs page",
			zap.String("repo", owner+"/"+repo),
			zap.Int("page_count", len(prs)),
			zap.Int("merged_total", len(all)),
		)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &ListMergedPRsResult{PRs: all, NewETag: newETag}, nil
}

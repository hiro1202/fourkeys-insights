package github

import (
	"context"

	gh "github.com/google/go-github/v62/github"
	"go.uber.org/zap"
)

// RepoInfo contains the fields we need from a GitHub repository.
type RepoInfo struct {
	Owner         string
	Name          string
	FullName      string
	DefaultBranch string
}

// ListAccessibleRepos returns all repos the authenticated user can access.
// Uses pagination with per_page=100 as specified in DESIGN.md.
func (c *Client) ListAccessibleRepos(ctx context.Context) ([]RepoInfo, error) {
	var all []RepoInfo
	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Type: "all",
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := c.client.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, r := range repos {
			all = append(all, RepoInfo{
				Owner:         r.GetOwner().GetLogin(),
				Name:          r.GetName(),
				FullName:      r.GetFullName(),
				DefaultBranch: r.GetDefaultBranch(),
			})
		}

		c.logger.Debug("fetched repos page",
			zap.Int("count", len(repos)),
			zap.Int("total", len(all)),
		)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return all, nil
}

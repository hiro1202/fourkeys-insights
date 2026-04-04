package github

import (
	"context"
	"regexp"
	"strconv"
	"time"
)

// issuePattern matches GitHub closing keywords in PR body.
// Pattern: (closes|close|closed|fix|fixes|fixed|resolve|resolves|resolved) #N
// Case-insensitive, first match wins.
var issuePattern = regexp.MustCompile(
	`(?i)(?:closes|close|closed|fix|fixes|fixed|resolve|resolves|resolved)\s+#(\d+)`,
)

// ParseIssueReference extracts the first linked issue number from a PR body.
// Returns 0 if no issue reference is found.
func ParseIssueReference(body string) int {
	matches := issuePattern.FindStringSubmatch(body)
	if len(matches) < 2 {
		return 0
	}
	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return num
}

// GetIssueCreatedAt fetches the created_at timestamp for a given issue number.
func (c *Client) GetIssueCreatedAt(ctx context.Context, owner, repo string, issueNumber int) (time.Time, error) {
	issue, _, err := c.client.Issues.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return time.Time{}, err
	}
	return issue.GetCreatedAt().Time, nil
}

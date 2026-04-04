package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gh "github.com/google/go-github/v62/github"
	"go.uber.org/zap"
)

// Client wraps the GitHub API with rate limit handling and logging.
type Client struct {
	client *gh.Client
	logger *zap.Logger
}

// NewClient creates a GitHub API client with PAT authentication.
func NewClient(token, baseURL string, logger *zap.Logger) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("github token is required")
	}

	httpClient := &http.Client{
		Transport: &rateLimitTransport{
			base:   &authTransport{token: token},
			logger: logger,
		},
		Timeout: 30 * time.Second,
	}

	client := gh.NewClient(httpClient)
	if baseURL != "" && baseURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, fmt.Errorf("setting enterprise URL: %w", err)
		}
	}

	return &Client{client: client, logger: logger}, nil
}

// ValidateToken checks the PAT and returns the authenticated user's login.
func (c *Client) ValidateToken(ctx context.Context) (string, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("validating token: %w", err)
	}
	return user.GetLogin(), nil
}

// authTransport adds the Authorization header.
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// rateLimitTransport handles primary and secondary rate limits.
type rateLimitTransport struct {
	base   http.RoundTripper
	logger *zap.Logger
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	const maxRetries = 3
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// Check primary rate limit (X-RateLimit-Remaining)
		if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
			if rem, _ := strconv.Atoi(remaining); rem < 10 {
				resetStr := resp.Header.Get("X-RateLimit-Reset")
				if resetUnix, _ := strconv.ParseInt(resetStr, 10, 64); resetUnix > 0 {
					resetTime := time.Unix(resetUnix, 0)
					wait := time.Until(resetTime)
					if wait > 0 && rem == 0 {
						t.logger.Warn("primary rate limit reached, waiting",
							zap.Duration("wait", wait),
							zap.Int("remaining", rem),
						)
						select {
						case <-time.After(wait):
						case <-req.Context().Done():
							return resp, req.Context().Err()
						}
						continue
					}
				}
			}
		}

		// Handle secondary rate limit (403 or 429)
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			wait := time.Duration(1<<uint(attempt)) * time.Second // exponential backoff
			if retryAfter != "" {
				if secs, _ := strconv.Atoi(retryAfter); secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}

			t.logger.Warn("secondary rate limit, retrying",
				zap.Int("status", resp.StatusCode),
				zap.Duration("wait", wait),
				zap.Int("attempt", attempt+1),
			)

			resp.Body.Close()
			select {
			case <-time.After(wait):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
			continue
		}

		return resp, nil
	}

	return resp, err
}

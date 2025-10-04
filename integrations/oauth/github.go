package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GitHubClient handles GitHub OAuth and API interactions
type GitHubClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
	baseURL      string
}

// NewGitHubClient creates a new GitHub OAuth client
func NewGitHubClient(clientID, clientSecret, redirectURL string) *GitHubClient {
	return &GitHubClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		baseURL:      "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetAuthURL returns the OAuth authorization URL
func (c *GitHubClient) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=read:user,repo",
		c.clientID,
		url.QueryEscape(c.redirectURL),
		state,
	)
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (c *GitHubClient) ExchangeCodeForToken(ctx context.Context, code string) (*OAuthToken, error) {
	resp, err := c.httpClient.PostForm(
		"https://github.com/login/oauth/access_token",
		url.Values{
			"client_id":     {c.clientID},
			"client_secret": {c.clientSecret},
			"code":          {code},
			"redirect_uri":  {c.redirectURL},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response (GitHub returns form-encoded response by default)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if errMsg := values.Get("error"); errMsg != "" {
		return nil, fmt.Errorf("github oauth error: %s - %s", errMsg, values.Get("error_description"))
	}

	token := &OAuthToken{
		AccessToken: values.Get("access_token"),
		TokenType:   values.Get("token_type"),
		Scope:       values.Get("scope"),
		CreatedAt:   time.Now(),
	}

	return token, nil
}

// GetUserStats fetches comprehensive GitHub statistics for a user
func (c *GitHubClient) GetUserStats(ctx context.Context, username string) (*GitHubUserStats, error) {
	// Note: This is a simplified implementation
	// Production version would use GraphQL API for efficiency and make multiple calls

	// Fetch user profile
	user, err := c.fetchJSON(ctx, fmt.Sprintf("%s/users/%s", c.baseURL, username))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	stats := &GitHubUserStats{
		Username:    username,
		Followers:   getInt(user, "followers"),
		Following:   getInt(user, "following"),
		PublicRepos: getInt(user, "public_repos"),
		PublicGists: getInt(user, "public_gists"),
		FetchedAt:   time.Now(),
	}

	// Calculate account age
	if createdAt := getString(user, "created_at"); createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			stats.AccountAge = int(time.Since(t).Hours() / 24)
			stats.ContribYears = int(time.Since(t).Hours() / 24 / 365)
		}
	}

	// For production: fetch additional stats via GraphQL
	// - Total commits across all repos
	// - Total stars received
	// - Total forks
	// - Organization memberships
	// - Top programming languages

	return stats, nil
}

// fetchJSON makes an authenticated GET request and returns parsed JSON
func (c *GitHubClient) fetchJSON(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "OCX-Reputation/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API error: status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper functions
func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(float64); ok {
			return int(i)
		}
	}
	return 0
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHubAdapter handles GitHub profile fetching and scoring
type GitHubAdapter struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewGitHubAdapter creates a new GitHub adapter
func NewGitHubAdapter(apiKey string) *GitHubAdapter {
	return &GitHubAdapter{
		apiKey:  apiKey,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Profile represents a GitHub user profile
type Profile struct {
	Platform    string
	Username    string
	Followers   int
	Following   int
	PublicRepos int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metrics     map[string]interface{}
}

// FetchProfile retrieves a GitHub user profile
func (a *GitHubAdapter) FetchProfile(ctx context.Context, username string) (*Profile, error) {
	// Fetch user profile
	userURL := fmt.Sprintf("%s/users/%s", a.baseURL, username)
	user, err := a.fetchJSON(ctx, userURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	// Fetch recent events for activity score
	eventsURL := fmt.Sprintf("%s/users/%s/events/public?per_page=100", a.baseURL, username)
	events, err := a.fetchJSONArray(ctx, eventsURL)
	if err != nil {
		// Don't fail if events can't be fetched, just log and continue
		events = []interface{}{}
	}

	// Fetch repositories for contribution score
	reposURL := fmt.Sprintf("%s/users/%s/repos?sort=updated&per_page=100", a.baseURL, username)
	repos, err := a.fetchJSONArray(ctx, reposURL)
	if err != nil {
		repos = []interface{}{}
	}

	return &Profile{
		Platform:    "github",
		Username:    username,
		Followers:   getInt(user, "followers"),
		Following:   getInt(user, "following"),
		PublicRepos: getInt(user, "public_repos"),
		CreatedAt:   parseGitHubDate(getString(user, "created_at")),
		UpdatedAt:   time.Now(),
		Metrics: map[string]interface{}{
			"total_stars":         calculateTotalStars(repos),
			"total_forks":         calculateTotalForks(repos),
			"recent_activity":     len(events),
			"contribution_streak": calculateStreak(events),
			"top_languages":       extractLanguages(repos),
		},
	}, nil
}

// CalculateScore computes a reputation score from a GitHub profile
func (a *GitHubAdapter) CalculateScore(profile *Profile) (float64, error) {
	var score float64

	// Account age (max 20 points)
	// Reward accounts older than 1 year progressively
	accountAge := time.Since(profile.CreatedAt).Hours() / 24 / 365
	score += min(accountAge*4, 20)

	// Follower quality (max 15 points)
	// Prefer accounts with good follower/following ratio
	if profile.Following > 0 {
		ratio := float64(profile.Followers) / float64(profile.Following)
		score += min(ratio*3, 15)
	} else if profile.Followers > 0 {
		score += 15 // Max points if following nobody but has followers
	}

	// Repository count (max 15 points)
	// Reward active contributors
	score += min(float64(profile.PublicRepos)/2, 15)

	// Stars received (max 20 points)
	// Indicates valuable contributions
	stars := getFloat(profile.Metrics, "total_stars")
	score += min(stars/50, 20)

	// Recent activity (max 15 points)
	// Reward active developers
	activity := getFloat(profile.Metrics, "recent_activity")
	score += min(activity/2, 15)

	// Contribution streak (max 15 points)
	// Reward consistent contributors
	streak := getFloat(profile.Metrics, "contribution_streak")
	score += min(streak/4, 15)

	return min(score, 100), nil
}

// ValidateConnection verifies a user owns a GitHub account
func (a *GitHubAdapter) ValidateConnection(userID, username string) error {
	// In production, this would use OAuth or GitHub's verification API
	// For now, we just verify the account exists
	ctx := context.Background()
	_, err := a.FetchProfile(ctx, username)
	return err
}

// Private helper methods

func (a *GitHubAdapter) fetchJSON(ctx context.Context, url string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "OCX-TrustScore/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("user not found")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("rate limited or forbidden")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *GitHubAdapter) fetchJSONArray(ctx context.Context, url string) ([]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "OCX-TrustScore/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper functions for extracting data

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(float64); ok {
			return int(i)
		}
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

func parseGitHubDate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}

func calculateTotalStars(repos []interface{}) float64 {
	total := 0.0
	for _, repo := range repos {
		if r, ok := repo.(map[string]interface{}); ok {
			total += getFloat(r, "stargazers_count")
		}
	}
	return total
}

func calculateTotalForks(repos []interface{}) float64 {
	total := 0.0
	for _, repo := range repos {
		if r, ok := repo.(map[string]interface{}); ok {
			total += getFloat(r, "forks_count")
		}
	}
	return total
}

func calculateStreak(events []interface{}) float64 {
	if len(events) == 0 {
		return 0
	}

	// Calculate days with activity in the last 30 days
	now := time.Now()
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	activeDays := make(map[string]bool)

	for _, event := range events {
		if e, ok := event.(map[string]interface{}); ok {
			createdAt := getString(e, "created_at")
			eventTime := parseGitHubDate(createdAt)
			if eventTime.After(thirtyDaysAgo) {
				day := eventTime.Format("2006-01-02")
				activeDays[day] = true
			}
		}
	}

	return float64(len(activeDays))
}

func extractLanguages(repos []interface{}) []string {
	languageMap := make(map[string]bool)
	for _, repo := range repos {
		if r, ok := repo.(map[string]interface{}); ok {
			lang := getString(r, "language")
			if lang != "" {
				languageMap[lang] = true
			}
		}
	}

	languages := make([]string, 0, len(languageMap))
	for lang := range languageMap {
		languages = append(languages, lang)
	}
	return languages
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

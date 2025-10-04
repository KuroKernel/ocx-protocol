package oauth

import "time"

// OAuthToken represents an OAuth 2.0 access token
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// IsExpired checks if the token has expired
func (t *OAuthToken) IsExpired() bool {
	if t.ExpiresIn == 0 {
		return false // No expiration
	}
	expiryTime := t.CreatedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
	return time.Now().After(expiryTime)
}

// GitHubUserStats represents GitHub user statistics for reputation scoring
type GitHubUserStats struct {
	Username      string    `json:"username"`
	TotalCommits  int       `json:"total_commits"`
	TotalStars    int       `json:"total_stars"`
	TotalForks    int       `json:"total_forks"`
	Followers     int       `json:"followers"`
	Following     int       `json:"following"`
	PublicRepos   int       `json:"public_repos"`
	PublicGists   int       `json:"public_gists"`
	ContribYears  int       `json:"contrib_years"`   // Years of contributions
	AccountAge    int       `json:"account_age"`     // Age in days
	TopLanguages  []string  `json:"top_languages"`
	Organizations int       `json:"organizations"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// LinkedInUserStats represents LinkedIn user statistics for reputation scoring
type LinkedInUserStats struct {
	UserID       string    `json:"user_id"`
	Connections  int       `json:"connections"`
	Endorsements int       `json:"endorsements"`
	Posts        int       `json:"posts"`
	Followers    int       `json:"followers"`
	ProfileViews int       `json:"profile_views"`
	SearchAppear int       `json:"search_appearances"`
	Skills       int       `json:"skills"`
	YearsExp     int       `json:"years_experience"`
	Education    int       `json:"education_count"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// UberUserStats represents Uber driver/rider statistics for reputation scoring
type UberUserStats struct {
	UserID      string    `json:"user_id"`
	UserType    string    `json:"user_type"` // "driver" or "rider"
	Rating      float64   `json:"rating"`    // 1.0-5.0 scale
	TotalTrips  int       `json:"total_trips"`
	TotalYears  int       `json:"total_years"`
	Cancellations int     `json:"cancellations"`
	Compliments   int     `json:"compliments"`
	FetchedAt   time.Time `json:"fetched_at"`
}

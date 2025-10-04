package adapters

import "context"

// PlatformAdapter defines the interface for platform integrations
type PlatformAdapter interface {
	// FetchProfile retrieves a user's profile from the platform
	FetchProfile(ctx context.Context, username string) (*Profile, error)

	// CalculateScore computes a reputation score from the profile
	CalculateScore(profile *Profile) (float64, error)

	// ValidateConnection verifies a user owns the platform account
	ValidateConnection(userID, username string) error
}

// AdapterFactory creates platform adapters
type AdapterFactory struct {
	githubAPIKey    string
	linkedinAPIKey  string
	twitterAPIKey   string
}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory(githubKey, linkedinKey, twitterKey string) *AdapterFactory {
	return &AdapterFactory{
		githubAPIKey:   githubKey,
		linkedinAPIKey: linkedinKey,
		twitterAPIKey:  twitterKey,
	}
}

// GetAdapter returns the appropriate adapter for a platform type
func (f *AdapterFactory) GetAdapter(platformType string) (PlatformAdapter, error) {
	switch platformType {
	case "github":
		return NewGitHubAdapter(f.githubAPIKey), nil
	// LinkedIn and Twitter adapters would be added here
	// case "linkedin":
	//     return NewLinkedInAdapter(f.linkedinAPIKey), nil
	// case "twitter":
	//     return NewTwitterAdapter(f.twitterAPIKey), nil
	default:
		return nil, nil // Return nil for unsupported platforms (graceful degradation)
	}
}

package integrations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ocx.local/integrations/cache"
	"ocx.local/integrations/normalizers"
	"ocx.local/integrations/oauth"
	"ocx.local/pkg/deterministicvm"
)

// PlatformScore represents a normalized score from a single platform
type PlatformScore struct {
	Platform  string    `json:"platform"`
	Score     float64   `json:"score"`      // Normalized to 0-100 range
	Timestamp time.Time `json:"timestamp"`  // When data was fetched
	Cached    bool      `json:"cached"`     // True if served from cache
	RawData   any       `json:"raw_data,omitempty"`
}

// ReputationData aggregates scores from all platforms
type ReputationData struct {
	UserID    string          `json:"user_id"`
	GitHub    *PlatformScore  `json:"github,omitempty"`
	LinkedIn  *PlatformScore  `json:"linkedin,omitempty"`
	Uber      *PlatformScore  `json:"uber,omitempty"`
	Timestamp int64           `json:"timestamp"` // Deterministic timestamp from OCX
}

// ReputationFetcher orchestrates platform data fetching with OAuth, caching, and rate limiting
type ReputationFetcher struct {
	// OCX engine for deterministic timestamps
	deterministicTime func() int64

	// Platform API clients
	githubClient   *oauth.GitHubClient
	linkedinClient *oauth.LinkedInClient
	uberClient     *oauth.UberClient

	// Normalizers convert raw API responses to 0-100 scores
	githubNormalizer   *normalizers.GitHubNormalizer
	linkedinNormalizer *normalizers.LinkedInNormalizer
	uberNormalizer     *normalizers.UberNormalizer

	// Caching layer (deterministic cache keying)
	cache cache.Cache

	// Rate limiter (per-platform rate limiting)
	rateLimiter *RateLimiter

	// Concurrency control
	mu sync.RWMutex
}

// NewReputationFetcher creates a new reputation fetcher with all dependencies
func NewReputationFetcher(
	deterministicTimeFn func() int64,
	githubClient *oauth.GitHubClient,
	linkedinClient *oauth.LinkedInClient,
	uberClient *oauth.UberClient,
	cacheImpl cache.Cache,
) *ReputationFetcher {
	return &ReputationFetcher{
		deterministicTime:  deterministicTimeFn,
		githubClient:       githubClient,
		linkedinClient:     linkedinClient,
		uberClient:         uberClient,
		githubNormalizer:   normalizers.NewGitHubNormalizer(),
		linkedinNormalizer: normalizers.NewLinkedInNormalizer(),
		uberNormalizer:     normalizers.NewUberNormalizer(),
		cache:              cacheImpl,
		rateLimiter:        NewRateLimiter(),
	}
}

// FetchAll fetches scores from all enabled platforms concurrently
// Returns ReputationData with platform scores populated
func (f *ReputationFetcher) FetchAll(ctx context.Context, userID string, platformFlags uint32) (*ReputationData, error) {
	data := &ReputationData{
		UserID:    userID,
		Timestamp: f.deterministicTime(),
	}

	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	// Fetch GitHub (platform flag bit 0)
	if platformFlags&0x01 != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score, err := f.FetchGitHub(ctx, userID)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("github: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			data.GitHub = score
			mu.Unlock()
		}()
	}

	// Fetch LinkedIn (platform flag bit 1)
	if platformFlags&0x02 != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score, err := f.FetchLinkedIn(ctx, userID)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("linkedin: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			data.LinkedIn = score
			mu.Unlock()
		}()
	}

	// Fetch Uber (platform flag bit 2)
	if platformFlags&0x04 != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score, err := f.FetchUber(ctx, userID)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("uber: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			data.Uber = score
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Return partial results even if some platforms failed
	if len(errors) > 0 && data.GitHub == nil && data.LinkedIn == nil && data.Uber == nil {
		// All platforms failed
		return nil, fmt.Errorf("all platforms failed: %v", errors)
	}

	return data, nil
}

// FetchGitHub fetches and normalizes GitHub reputation data
func (f *ReputationFetcher) FetchGitHub(ctx context.Context, userID string) (*PlatformScore, error) {
	// Use deterministic timestamp for cache keying (1-hour buckets)
	deterministicTime := f.deterministicTime()
	cacheKey := fmt.Sprintf("github:%s:%d", userID, deterministicTime/3600)

	// Check cache first (deterministic lookup)
	if cached, ok := f.cache.Get(cacheKey); ok {
		if score, ok := cached.(*PlatformScore); ok {
			score.Cached = true
			return score, nil
		}
	}

	// Rate limit check
	if err := f.rateLimiter.Wait(ctx, "github"); err != nil {
		// If rate limited, try to serve stale cache
		if stale := f.getStaleCache("github", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// OAuth-authenticated API call
	stats, err := f.githubClient.GetUserStats(ctx, userID)
	if err != nil {
		// On error, try to serve stale cache
		if stale := f.getStaleCache("github", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("github API error: %w", err)
	}

	// Normalize to 0-100 score (deterministic math)
	score := f.githubNormalizer.Normalize(stats)

	platformScore := &PlatformScore{
		Platform:  "github",
		Score:     score,
		Timestamp: time.Unix(deterministicTime, 0),
		Cached:    false,
		RawData:   stats,
	}

	// Cache with deterministic timestamp
	f.cache.Set(cacheKey, platformScore, 1*time.Hour)

	return platformScore, nil
}

// FetchLinkedIn fetches and normalizes LinkedIn reputation data
func (f *ReputationFetcher) FetchLinkedIn(ctx context.Context, userID string) (*PlatformScore, error) {
	// Use deterministic timestamp for cache keying (1-hour buckets)
	deterministicTime := f.deterministicTime()
	cacheKey := fmt.Sprintf("linkedin:%s:%d", userID, deterministicTime/3600)

	// Check cache first
	if cached, ok := f.cache.Get(cacheKey); ok {
		if score, ok := cached.(*PlatformScore); ok {
			score.Cached = true
			return score, nil
		}
	}

	// Rate limit check
	if err := f.rateLimiter.Wait(ctx, "linkedin"); err != nil {
		if stale := f.getStaleCache("linkedin", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// OAuth-authenticated API call
	stats, err := f.linkedinClient.GetUserStats(ctx, userID)
	if err != nil {
		if stale := f.getStaleCache("linkedin", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("linkedin API error: %w", err)
	}

	// Normalize to 0-100 score
	score := f.linkedinNormalizer.Normalize(stats)

	platformScore := &PlatformScore{
		Platform:  "linkedin",
		Score:     score,
		Timestamp: time.Unix(deterministicTime, 0),
		Cached:    false,
		RawData:   stats,
	}

	// Cache with deterministic timestamp
	f.cache.Set(cacheKey, platformScore, 1*time.Hour)

	return platformScore, nil
}

// FetchUber fetches and normalizes Uber driver/rider rating data
func (f *ReputationFetcher) FetchUber(ctx context.Context, userID string) (*PlatformScore, error) {
	// Use deterministic timestamp for cache keying (1-hour buckets)
	deterministicTime := f.deterministicTime()
	cacheKey := fmt.Sprintf("uber:%s:%d", userID, deterministicTime/3600)

	// Check cache first
	if cached, ok := f.cache.Get(cacheKey); ok {
		if score, ok := cached.(*PlatformScore); ok {
			score.Cached = true
			return score, nil
		}
	}

	// Rate limit check
	if err := f.rateLimiter.Wait(ctx, "uber"); err != nil {
		if stale := f.getStaleCache("uber", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// OAuth-authenticated API call
	stats, err := f.uberClient.GetUserStats(ctx, userID)
	if err != nil {
		if stale := f.getStaleCache("uber", userID); stale != nil {
			stale.Cached = true
			return stale, nil
		}
		return nil, fmt.Errorf("uber API error: %w", err)
	}

	// Normalize to 0-100 score
	score := f.uberNormalizer.Normalize(stats)

	platformScore := &PlatformScore{
		Platform:  "uber",
		Score:     score,
		Timestamp: time.Unix(deterministicTime, 0),
		Cached:    false,
		RawData:   stats,
	}

	// Cache with deterministic timestamp
	f.cache.Set(cacheKey, platformScore, 1*time.Hour)

	return platformScore, nil
}

// getStaleCache attempts to retrieve any cached value for a platform/user
// Used as fallback when rate limited or API errors occur
func (f *ReputationFetcher) getStaleCache(platform, userID string) *PlatformScore {
	// Try to find any recent cache entry (check last 24 hours)
	now := f.deterministicTime()
	for hourOffset := int64(0); hourOffset < 24; hourOffset++ {
		cacheKey := fmt.Sprintf("%s:%s:%d", platform, userID, (now-hourOffset*3600)/3600)
		if cached, ok := f.cache.Get(cacheKey); ok {
			if score, ok := cached.(*PlatformScore); ok {
				return score
			}
		}
	}
	return nil
}

// GetDeterministicTimestampFromEngine is a helper to extract deterministic time from D-MVM
func GetDeterministicTimestampFromEngine(engine *deterministicvm.Engine) int64 {
	// This would call into the D-MVM's ChaCha20/12 PRNG-based timestamp
	// For now, return current time (will be replaced with actual D-MVM integration)
	return time.Now().Unix()
}

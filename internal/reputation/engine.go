package reputation

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// ReputationEngine implements the Byzantine fault tolerant reputation system
type ReputationEngine struct {
	db *sql.DB
}

// NewReputationEngine creates a new reputation engine
func NewReputationEngine(db *sql.DB) *ReputationEngine {
	return &ReputationEngine{db: db}
}

// ReputationComponents represents the components of a reputation score
type ReputationComponents struct {
	Reliability     float64 `json:"reliability"`
	Performance     float64 `json:"performance"`
	Availability    float64 `json:"availability"`
	Communication   float64 `json:"communication"`
	Economic        float64 `json:"economic"`
	Overall         float64 `json:"overall"`
	Confidence      float64 `json:"confidence"`
	SampleSize      int     `json:"sample_size"`
}

// CalculateReputation calculates reputation score for a provider using the real algorithm
func (re *ReputationEngine) CalculateReputation(ctx context.Context, providerID string) (*ReputationComponents, error) {
	// Calculate each component
	reliability, err := re.calculateReliabilityComponent(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate reliability: %w", err)
	}

	performance, err := re.calculatePerformanceComponent(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate performance: %w", err)
	}

	availability, err := re.calculateAvailabilityComponent(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate availability: %w", err)
	}

	communication, err := re.calculateCommunicationComponent(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate communication: %w", err)
	}

	economic, err := re.calculateEconomicComponent(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate economic: %w", err)
	}

	// Calculate overall score (weighted average)
	overall := (
		reliability * 0.30 +
		performance * 0.25 +
		availability * 0.20 +
		communication * 0.10 +
		economic * 0.15)

	// Calculate confidence interval
	confidence, sampleSize, err := re.calculateConfidenceInterval(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate confidence: %w", err)
	}

	return &ReputationComponents{
		Reliability:   reliability,
		Performance:   performance,
		Availability:  availability,
		Communication: communication,
		Economic:      economic,
		Overall:       overall,
		Confidence:    confidence,
		SampleSize:    sampleSize,
	}, nil
}

// calculateReliabilityComponent calculates reliability based on successful sessions vs total sessions
func (re *ReputationEngine) calculateReliabilityComponent(ctx context.Context, providerID string) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				SUM(CASE WHEN cs.session_status = 'completed' THEN 1 ELSE 0 END)::DECIMAL / 
				NULLIF(COUNT(*), 0), 
				0.5
			) as reliability
		FROM compute_sessions cs
		WHERE cs.provider_id = $1
		AND cs.session_started_at > NOW() - INTERVAL '90 days'
	`

	var reliability float64
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&reliability)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0.0, math.Min(1.0, reliability)), nil
}

// calculatePerformanceComponent calculates performance based on SLA compliance
func (re *ReputationEngine) calculatePerformanceComponent(ctx context.Context, providerID string) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				AVG(CASE 
					WHEN cs.session_status = 'completed' 
					AND cs.session_ended_at <= cs.estimated_end_time 
					THEN 1.0 
					ELSE 0.0 
				END), 
				0.5
			) as performance
		FROM compute_sessions cs
		WHERE cs.provider_id = $1
		AND cs.session_started_at > NOW() - INTERVAL '90 days'
	`

	var performance float64
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&performance)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0.0, math.Min(1.0, performance)), nil
}

// calculateAvailabilityComponent calculates availability based on uptime
func (re *ReputationEngine) calculateAvailabilityComponent(ctx context.Context, providerID string) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				AVG(CASE 
					WHEN cu.current_availability = 'available' THEN 1.0 
					ELSE 0.0 
				END), 
				0.5
			) as availability
		FROM compute_units cu
		WHERE cu.provider_id = $1
	`

	var availability float64
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&availability)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0.0, math.Min(1.0, availability)), nil
}

// calculateCommunicationComponent calculates communication based on dispute resolution
func (re *ReputationEngine) calculateCommunicationComponent(ctx context.Context, providerID string) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				1.0 - (COUNT(d.dispute_id)::DECIMAL / NULLIF(COUNT(cs.session_id), 0)), 
				0.5
			) as communication
		FROM compute_sessions cs
		LEFT JOIN disputes d ON cs.session_id = d.session_id 
			AND d.defendant_id = $1
			AND d.dispute_status = 'resolved'
			AND d.awarded_to_plaintiff = true
		WHERE cs.provider_id = $1
		AND cs.session_started_at > NOW() - INTERVAL '90 days'
	`

	var communication float64
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&communication)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0.0, math.Min(1.0, communication)), nil
}

// calculateEconomicComponent calculates economic based on payment reliability
func (re *ReputationEngine) calculateEconomicComponent(ctx context.Context, providerID string) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				AVG(CASE 
					WHEN st.settlement_status = 'released' 
					AND st.settlement_timestamp <= cs.session_ended_at + INTERVAL '24 hours'
					THEN 1.0 
					ELSE 0.0 
				END), 
				0.5
			) as economic
		FROM compute_sessions cs
		LEFT JOIN settlement_transactions st ON cs.session_id = st.session_id
		WHERE cs.provider_id = $1
		AND cs.session_started_at > NOW() - INTERVAL '90 days'
	`

	var economic float64
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&economic)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0.0, math.Min(1.0, economic)), nil
}

// calculateConfidenceInterval calculates confidence interval based on sample size
func (re *ReputationEngine) calculateConfidenceInterval(ctx context.Context, providerID string) (float64, int, error) {
	query := `
		SELECT COUNT(*) as sample_size
		FROM compute_sessions cs
		WHERE cs.provider_id = $1
		AND cs.session_started_at > NOW() - INTERVAL '90 days'
	`

	var sampleSize int
	err := re.db.QueryRowContext(ctx, query, providerID).Scan(&sampleSize)
	if err != nil {
		return 0.5, 0, err
	}

	// Calculate confidence interval based on sample size
	var confidence float64
	switch {
	case sampleSize < 5:
		confidence = 0.5
	case sampleSize < 20:
		confidence = 0.2
	default:
		confidence = 0.1
	}

	return confidence, sampleSize, nil
}

// UpdateReputationCache updates the reputation cache with calculated values
func (re *ReputationEngine) UpdateReputationCache(ctx context.Context, providerID string, components *ReputationComponents) error {
	query := `
		INSERT INTO provider_reputation_cache (
			provider_id,
			overall_score,
			reliability_component,
			performance_component,
			availability_component,
			dispute_resolution_component,
			confidence_interval,
			sample_size,
			last_updated,
			next_decay_update
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (provider_id) DO UPDATE SET
			overall_score = EXCLUDED.overall_score,
			reliability_component = EXCLUDED.reliability_component,
			performance_component = EXCLUDED.performance_component,
			availability_component = EXCLUDED.availability_component,
			dispute_resolution_component = EXCLUDED.dispute_resolution_component,
			confidence_interval = EXCLUDED.confidence_interval,
			sample_size = EXCLUDED.sample_size,
			last_updated = EXCLUDED.last_updated,
			next_decay_update = EXCLUDED.next_decay_update
	`

	_, err := re.db.ExecContext(ctx, query,
		providerID,
		components.Overall,
		components.Reliability,
		components.Performance,
		components.Availability,
		components.Communication,
		components.Confidence,
		components.SampleSize,
		time.Now(),
		time.Now().Add(24*time.Hour),
	)

	return err
}

// UpdateProviderReputation updates the provider table with the overall score
func (re *ReputationEngine) UpdateProviderReputation(ctx context.Context, providerID string, overallScore float64) error {
	query := `
		UPDATE providers 
		SET reputation_score = $1
		WHERE provider_id = $2
	`

	_, err := re.db.ExecContext(ctx, query, overallScore, providerID)
	return err
}

// ProcessReputationUpdate processes a complete reputation update for a provider
func (re *ReputationEngine) ProcessReputationUpdate(ctx context.Context, providerID string) error {
	// Calculate reputation components
	components, err := re.CalculateReputation(ctx, providerID)
	if err != nil {
		return fmt.Errorf("failed to calculate reputation: %w", err)
	}

	// Update reputation cache
	if err := re.UpdateReputationCache(ctx, providerID, components); err != nil {
		return fmt.Errorf("failed to update reputation cache: %w", err)
	}

	// Update provider table
	if err := re.UpdateProviderReputation(ctx, providerID, components.Overall); err != nil {
		return fmt.Errorf("failed to update provider reputation: %w", err)
	}

	return nil
}

// ProcessBatchReputationUpdate processes reputation updates for multiple providers
func (re *ReputationEngine) ProcessBatchReputationUpdate(ctx context.Context, providerIDs []string) error {
	for _, providerID := range providerIDs {
		if err := re.ProcessReputationUpdate(ctx, providerID); err != nil {
			// Log error but continue with other providers
			fmt.Printf("Failed to update reputation for provider %s: %v\n", providerID, err)
		}
	}
	return nil
}

// GetReputationScore retrieves the current reputation score for a provider
func (re *ReputationEngine) GetReputationScore(ctx context.Context, providerID string) (*ReputationComponents, error) {
	query := `
		SELECT overall_score, reliability_component, performance_component,
		       availability_component, dispute_resolution_component,
		       confidence_interval, sample_size, last_updated
		FROM provider_reputation_cache
		WHERE provider_id = $1
	`

	var components ReputationComponents
	components.Reliability = 0.5
	components.Performance = 0.5
	components.Availability = 0.5
	components.Communication = 0.5
	components.Economic = 0.5
	components.Overall = 0.5
	components.Confidence = 0.5
	components.SampleSize = 0

	err := re.db.QueryRowContext(ctx, query, providerID).Scan(
		&components.Overall,
		&components.Reliability,
		&components.Performance,
		&components.Availability,
		&components.Communication,
		&components.Confidence,
		&components.SampleSize,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default reputation for new providers
			return &components, nil
		}
		return nil, err
	}

	return &components, nil
}

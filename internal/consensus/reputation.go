package consensus

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// ReputationManager handles reputation calculations and updates
type ReputationManager struct {
	db *sql.DB
}

// NewReputationManager creates a new reputation manager
func NewReputationManager(db *sql.DB) *ReputationManager {
	return &ReputationManager{db: db}
}

// ReputationScore represents a provider's reputation score
type ReputationScore struct {
	ProviderID       string    `json:"provider_id"`
	OverallScore     float64   `json:"overall_score"`
	ReliabilityScore float64   `json:"reliability_score"`
	PerformanceScore float64   `json:"performance_score"`
	AvailabilityScore float64  `json:"availability_score"`
	CommunicationScore float64 `json:"communication_score"`
	EconomicScore    float64   `json:"economic_score"`
	ConfidenceInterval float64 `json:"confidence_interval"`
	SampleSize       int       `json:"sample_size"`
	LastUpdated      time.Time `json:"last_updated"`
}

// CheckRequesterReputation checks if a requester meets minimum reputation requirements
func (rm *ReputationManager) CheckRequesterReputation(ctx context.Context, requesterID string) error {
	// Get requester's reputation score
	score, err := rm.GetReputationScore(ctx, requesterID)
	if err != nil {
		return fmt.Errorf("failed to get requester reputation: %w", err)
	}

	// Check if reputation meets minimum threshold
	minReputation := 0.5 // Minimum reputation score required
	if score.OverallScore < minReputation {
		return fmt.Errorf("requester reputation %.2f below minimum %.2f", score.OverallScore, minReputation)
	}

	// Check if there's sufficient sample size
	minSampleSize := 5
	if score.SampleSize < minSampleSize {
		return fmt.Errorf("insufficient reputation sample size: %d < %d", score.SampleSize, minSampleSize)
	}

	return nil
}

// GetReputationScore retrieves a provider's current reputation score
func (rm *ReputationManager) GetReputationScore(ctx context.Context, providerID string) (*ReputationScore, error) {
	query := `
		SELECT overall_score, reliability_component, performance_component, 
		       availability_component, dispute_resolution_component, 
		       confidence_interval, sample_size, last_updated
		FROM provider_reputation_cache 
		WHERE provider_id = $1
	`

	var score ReputationScore
	score.ProviderID = providerID

	err := rm.db.QueryRowContext(ctx, query, providerID).Scan(
		&score.OverallScore,
		&score.ReliabilityScore,
		&score.PerformanceScore,
		&score.AvailabilityScore,
		&score.CommunicationScore,
		&score.ConfidenceInterval,
		&score.SampleSize,
		&score.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default reputation for new providers
			return &ReputationScore{
				ProviderID:       providerID,
				OverallScore:     0.5,
				ReliabilityScore: 0.5,
				PerformanceScore: 0.5,
				AvailabilityScore: 0.5,
				CommunicationScore: 0.5,
				EconomicScore:    0.5,
				ConfidenceInterval: 0.5,
				SampleSize:       0,
				LastUpdated:      time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get reputation score: %w", err)
	}

	return &score, nil
}

// UpdateReputationScore updates a provider's reputation based on session results
func (rm *ReputationManager) UpdateReputationScore(ctx context.Context, sessionID string, providerID string, success bool, performanceMetrics map[string]float64) error {
	// Calculate reputation impact based on session results
	var impact float64
	if success {
		impact = 0.1 // Positive impact for successful sessions
	} else {
		impact = -0.2 // Negative impact for failed sessions
	}

	// Adjust impact based on performance metrics
	if performanceMetrics != nil {
		if gpuUtilization, exists := performanceMetrics["gpu_utilization"]; exists {
			if gpuUtilization > 90 {
				impact += 0.05 // Bonus for high utilization
			} else if gpuUtilization < 50 {
				impact -= 0.05 // Penalty for low utilization
			}
		}
	}

	// Record reputation event
	eventQuery := `
		INSERT INTO reputation_events (provider_id, session_id, event_type, impact_score, description, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	eventType := "session_completed_successfully"
	description := "Session completed successfully"
	if !success {
		eventType = "session_terminated_early"
		description = "Session terminated early"
	}

	_, err := rm.db.ExecContext(ctx, eventQuery, providerID, sessionID, eventType, impact, description, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record reputation event: %w", err)
	}

	// Recalculate reputation score
	return rm.RecalculateReputationScore(ctx, providerID)
}

// RecalculateReputationScore recalculates a provider's reputation score
func (rm *ReputationManager) RecalculateReputationScore(ctx context.Context, providerID string) error {
	// Get recent events (last 90 days)
	eventsQuery := `
		SELECT event_type, impact_score, timestamp
		FROM reputation_events 
		WHERE provider_id = $1 
		AND timestamp > NOW() - INTERVAL '90 days'
		ORDER BY timestamp DESC
	`

	rows, err := rm.db.QueryContext(ctx, eventsQuery, providerID)
	if err != nil {
		return fmt.Errorf("failed to get reputation events: %w", err)
	}
	defer rows.Close()

	var events []ReputationEvent
	for rows.Next() {
		var event ReputationEvent
		err := rows.Scan(&event.EventType, &event.ImpactScore, &event.Timestamp)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	// Calculate reputation components
	reliabilityScore := rm.calculateReliabilityScore(events)
	performanceScore := rm.calculatePerformanceScore(events)
	availabilityScore := rm.calculateAvailabilityScore(events)
	communicationScore := rm.calculateCommunicationScore(events)
	economicScore := rm.calculateEconomicScore(events)

	// Calculate overall score (weighted average)
	overallScore := (
		reliabilityScore*0.30 +
		performanceScore*0.25 +
		availabilityScore*0.20 +
		communicationScore*0.10 +
		economicScore*0.15)

	// Calculate confidence interval
	confidenceInterval := rm.calculateConfidenceInterval(len(events))

	// Update reputation cache
	updateQuery := `
		INSERT INTO provider_reputation_cache (
			provider_id, overall_score, reliability_component, 
			performance_component, availability_component, 
			dispute_resolution_component, confidence_interval, 
			sample_size, last_updated, next_decay_update
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

	_, err = rm.db.ExecContext(ctx, updateQuery, providerID, overallScore, reliabilityScore,
		performanceScore, availabilityScore, communicationScore, confidenceInterval,
		len(events), time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("failed to update reputation cache: %w", err)
	}

	// Update provider table
	providerUpdateQuery := `
		UPDATE providers 
		SET reputation_score = $1 
		WHERE provider_id = $2
	`

	_, err = rm.db.ExecContext(ctx, providerUpdateQuery, overallScore, providerID)
	if err != nil {
		return fmt.Errorf("failed to update provider reputation: %w", err)
	}

	return nil
}

// Helper methods for reputation calculation

func (rm *ReputationManager) calculateReliabilityScore(events []ReputationEvent) float64 {
	successCount := 0
	totalCount := 0

	for _, event := range events {
		if event.EventType == "session_completed_successfully" {
			successCount++
		}
		if event.EventType == "session_completed_successfully" || event.EventType == "session_terminated_early" {
			totalCount++
		}
	}

	if totalCount == 0 {
		return 0.5 // Default score for new providers
	}

	return math.Max(0.0, math.Min(1.0, float64(successCount)/float64(totalCount)))
}

func (rm *ReputationManager) calculatePerformanceScore(events []ReputationEvent) float64 {
	// Simplified performance calculation
	// In production, this would analyze actual performance metrics
	performanceEvents := 0
	positiveEvents := 0

	for _, event := range events {
		if event.EventType == "performance_exceeded_sla" {
			positiveEvents++
			performanceEvents++
		} else if event.EventType == "performance_below_sla" {
			performanceEvents++
		}
	}

	if performanceEvents == 0 {
		return 0.5 // Default score
	}

	return math.Max(0.0, math.Min(1.0, float64(positiveEvents)/float64(performanceEvents)))
}

func (rm *ReputationManager) calculateAvailabilityScore(events []ReputationEvent) float64 {
	// Simplified availability calculation
	// In production, this would analyze actual uptime data
	uptimeEvents := 0
	positiveEvents := 0

	for _, event := range events {
		if event.EventType == "uptime_penalty" {
			uptimeEvents++
		} else if event.EventType == "session_completed_successfully" {
			positiveEvents++
			uptimeEvents++
		}
	}

	if uptimeEvents == 0 {
		return 0.5 // Default score
	}

	return math.Max(0.0, math.Min(1.0, float64(positiveEvents)/float64(uptimeEvents)))
}

func (rm *ReputationManager) calculateCommunicationScore(events []ReputationEvent) float64 {
	// Simplified communication calculation
	// In production, this would analyze dispute resolution and support metrics
	disputeEvents := 0
	positiveEvents := 0

	for _, event := range events {
		if event.EventType == "dispute_resolved_in_favor" {
			positiveEvents++
			disputeEvents++
		} else if event.EventType == "dispute_resolved_against" {
			disputeEvents++
		}
	}

	if disputeEvents == 0 {
		return 0.5 // Default score
	}

	return math.Max(0.0, math.Min(1.0, float64(positiveEvents)/float64(disputeEvents)))
}

func (rm *ReputationManager) calculateEconomicScore(events []ReputationEvent) float64 {
	// Simplified economic calculation
	// In production, this would analyze payment reliability and financial stability
	economicEvents := 0
	positiveEvents := 0

	for _, event := range events {
		if event.EventType == "session_completed_successfully" {
			positiveEvents++
			economicEvents++
		} else if event.EventType == "session_terminated_early" {
			economicEvents++
		}
	}

	if economicEvents == 0 {
		return 0.5 // Default score
	}

	return math.Max(0.0, math.Min(1.0, float64(positiveEvents)/float64(economicEvents)))
}

func (rm *ReputationManager) calculateConfidenceInterval(sampleSize int) float64 {
	// Calculate confidence interval based on sample size
	if sampleSize < 5 {
		return 0.5
	} else if sampleSize < 20 {
		return 0.2
	} else {
		return 0.1
	}
}

// ReputationEvent represents a reputation-affecting event
type ReputationEvent struct {
	EventType   string    `json:"event_type"`
	ImpactScore float64   `json:"impact_score"`
	Timestamp   time.Time `json:"timestamp"`
}

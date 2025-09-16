package reputation

import (
	"database/sql"
	"fmt"
	"time"
)

// DatabaseReputationEngine implements reputation scoring with database persistence
type DatabaseReputationEngine struct {
	db *sql.DB
}

// NewDatabaseReputationEngine creates a new database-backed reputation engine
func NewDatabaseReputationEngine(db *sql.DB) *DatabaseReputationEngine {
	return &DatabaseReputationEngine{db: db}
}

// ReputationScore represents a provider's reputation score
type ReputationScore struct {
	ProviderID       string    `json:"provider_id"`
	OverallScore     float64   `json:"overall_score"`
	ReliabilityScore float64   `json:"reliability_score"`
	PerformanceScore float64   `json:"performance_score"`
	ComplianceScore  float64   `json:"compliance_score"`
	LastUpdated      time.Time `json:"last_updated"`
	EventCount       int       `json:"event_count"`
}

// ReputationEvent represents a reputation-affecting event
type ReputationEvent struct {
	EventID     string    `json:"event_id"`
	ProviderID  string    `json:"provider_id"`
	EventType   string    `json:"event_type"`
	Weight      float64   `json:"weight"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Metadata    string    `json:"metadata"` // JSON string
}

// CalculateReputation calculates reputation score for a provider
func (e *DatabaseReputationEngine) CalculateReputation(providerID string) (*ReputationScore, error) {
	// Get recent events (last 30 days)
	events, err := e.getRecentEvents(providerID, 30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	// Calculate scores
	reliabilityScore := e.calculateReliabilityScore(events)
	performanceScore := e.calculatePerformanceScore(events)
	complianceScore := e.calculateComplianceScore(events)
	
	// Overall score is weighted average
	overallScore := (reliabilityScore*0.4 + performanceScore*0.4 + complianceScore*0.2)

	score := &ReputationScore{
		ProviderID:       providerID,
		OverallScore:     overallScore,
		ReliabilityScore: reliabilityScore,
		PerformanceScore: performanceScore,
		ComplianceScore:  complianceScore,
		LastUpdated:      time.Now(),
		EventCount:       len(events),
	}

	// Update database
	if err := e.updateReputationScore(score); err != nil {
		return nil, fmt.Errorf("failed to update reputation score: %w", err)
	}

	return score, nil
}

// RecordEvent records a reputation-affecting event
func (e *DatabaseReputationEngine) RecordEvent(event *ReputationEvent) error {
	query := `
		INSERT INTO reputation_events (event_id, provider_id, event_type, weight, description, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := e.db.Exec(query, event.EventID, event.ProviderID, event.EventType, 
		event.Weight, event.Description, event.Timestamp, event.Metadata)
	
	return err
}

// GetReputationScore retrieves current reputation score for a provider
func (e *DatabaseReputationEngine) GetReputationScore(providerID string) (*ReputationScore, error) {
	query := `
		SELECT provider_id, overall_score, reliability_score, performance_score, 
		       compliance_score, last_updated, event_count
		FROM provider_reputation_cache 
		WHERE provider_id = $1
	`
	
	var score ReputationScore
	err := e.db.QueryRow(query, providerID).Scan(
		&score.ProviderID, &score.OverallScore, &score.ReliabilityScore,
		&score.PerformanceScore, &score.ComplianceScore, &score.LastUpdated, &score.EventCount,
	)
	
	if err == sql.ErrNoRows {
		// Calculate new score if not cached
		return e.CalculateReputation(providerID)
	}
	
	return &score, err
}

// GetTopProviders returns providers ranked by reputation
func (e *DatabaseReputationEngine) GetTopProviders(limit int) ([]ReputationScore, error) {
	query := `
		SELECT provider_id, overall_score, reliability_score, performance_score, 
		       compliance_score, last_updated, event_count
		FROM provider_reputation_cache 
		ORDER BY overall_score DESC
		LIMIT $1
	`
	
	rows, err := e.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var scores []ReputationScore
	for rows.Next() {
		var score ReputationScore
		err := rows.Scan(&score.ProviderID, &score.OverallScore, &score.ReliabilityScore,
			&score.PerformanceScore, &score.ComplianceScore, &score.LastUpdated, &score.EventCount)
		if err != nil {
			continue
		}
		scores = append(scores, score)
	}
	
	return scores, nil
}

// Helper methods
func (e *DatabaseReputationEngine) getRecentEvents(providerID string, duration time.Duration) ([]ReputationEvent, error) {
	query := `
		SELECT event_id, provider_id, event_type, weight, description, timestamp, metadata
		FROM reputation_events 
		WHERE provider_id = $1 AND timestamp > $2
		ORDER BY timestamp DESC
	`
	
	rows, err := e.db.Query(query, providerID, time.Now().Add(-duration))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []ReputationEvent
	for rows.Next() {
		var event ReputationEvent
		err := rows.Scan(&event.EventID, &event.ProviderID, &event.EventType,
			&event.Weight, &event.Description, &event.Timestamp, &event.Metadata)
		if err != nil {
			continue
		}
		events = append(events, event)
	}
	
	return events, nil
}

func (e *DatabaseReputationEngine) calculateReliabilityScore(events []ReputationEvent) float64 {
	// Base score
	score := 0.5
	
	// Positive events increase score
	for _, event := range events {
		switch event.EventType {
		case "session_completed", "order_fulfilled", "uptime_good":
			score += event.Weight * 0.1
		case "session_failed", "order_cancelled", "downtime":
			score -= event.Weight * 0.2
		}
	}
	
	// Clamp between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *DatabaseReputationEngine) calculatePerformanceScore(events []ReputationEvent) float64 {
	// Base score
	score := 0.5
	
	// Performance events affect score
	for _, event := range events {
		switch event.EventType {
		case "high_performance", "fast_provisioning", "efficient_compute":
			score += event.Weight * 0.15
		case "low_performance", "slow_provisioning", "inefficient_compute":
			score -= event.Weight * 0.1
		}
	}
	
	// Clamp between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *DatabaseReputationEngine) calculateComplianceScore(events []ReputationEvent) float64 {
	// Base score
	score := 0.5
	
	// Compliance events affect score
	for _, event := range events {
		switch event.EventType {
		case "compliance_verified", "security_audit_passed", "certification_renewed":
			score += event.Weight * 0.2
		case "compliance_violation", "security_breach", "certification_expired":
			score -= event.Weight * 0.3
		}
	}
	
	// Clamp between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *DatabaseReputationEngine) updateReputationScore(score *ReputationScore) error {
	query := `
		INSERT INTO provider_reputation_cache 
		(provider_id, overall_score, reliability_score, performance_score, compliance_score, last_updated, event_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (provider_id) 
		DO UPDATE SET 
			overall_score = EXCLUDED.overall_score,
			reliability_score = EXCLUDED.reliability_score,
			performance_score = EXCLUDED.performance_score,
			compliance_score = EXCLUDED.compliance_score,
			last_updated = EXCLUDED.last_updated,
			event_count = EXCLUDED.event_count
	`
	
	_, err := e.db.Exec(query, score.ProviderID, score.OverallScore, score.ReliabilityScore,
		score.PerformanceScore, score.ComplianceScore, score.LastUpdated, score.EventCount)
	
	return err
}

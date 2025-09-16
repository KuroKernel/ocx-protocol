package reputation

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

// ReputationEngine handles Byzantine fault tolerant reputation scoring
type ReputationEngine struct {
	db              *sql.DB
	weights         *ComponentWeights
	decayParameters *DecayParameters
	antiGamingRules *AntiGamingRules
}

// ComponentWeights defines the relative importance of different trust factors
type ComponentWeights struct {
	Reliability   float64 `json:"reliability"`   // 0.30 - Did they deliver what was promised?
	Performance   float64 `json:"performance"`   // 0.25 - Did hardware perform as specified?
	Availability  float64 `json:"availability"`  // 0.20 - Uptime and responsiveness
	Communication float64 `json:"communication"` // 0.10 - Dispute resolution, responsiveness
	Economic      float64 `json:"economic"`      // 0.15 - Payment reliability, staking history
}

// DecayParameters control temporal decay of reputation events
type DecayParameters struct {
	Lambda      float64       `json:"lambda"`       // Decay rate (higher = faster decay)
	MinWeight   float64       `json:"min_weight"`   // Minimum weight for very old events
	PriorWeight float64       `json:"prior_weight"` // Weight for Bayesian smoothing
}

// AntiGamingRules detect and prevent reputation manipulation
type AntiGamingRules struct {
	MinScoreVariance           float64       `json:"min_score_variance"`
	GamingPenaltyFactor        float64       `json:"gaming_penalty_factor"`
	RapidFireWindow            int           `json:"rapid_fire_window"`
	MinTimeBetweenEvents       time.Duration `json:"min_time_between_events"`
	MaxDiscrepancyThreshold    float64       `json:"max_discrepancy_threshold"`
}

// ReputationScore represents a provider's trustworthiness
type ReputationScore struct {
	Overall         float64   `json:"overall"`         // 0.000 - 1.000
	Reliability     float64   `json:"reliability"`     // Component scores
	Performance     float64   `json:"performance"`
	Availability    float64   `json:"availability"`
	Communication   float64   `json:"communication"`
	Economic        float64   `json:"economic"`
	ConfidenceInterval float64 `json:"confidence_interval"` // Standard error
	SampleSize      int       `json:"sample_size"`         // Number of interactions
	LastUpdated     time.Time `json:"last_updated"`
}

// ReputationEvent represents a single trust event
type ReputationEvent struct {
	EventID           string    `json:"event_id"`
	ProviderID        string    `json:"provider_id"`
	SessionID         string    `json:"session_id"`
	EventType         string    `json:"event_type"`
	ImpactScore       float64   `json:"impact_score"`
	EventDescription  string    `json:"event_description"`
	EvidenceHash      string    `json:"evidence_hash"`
	VerifiedBy        string    `json:"verified_by"`
	VerificationTime  time.Time `json:"verification_timestamp"`
	CreatedAt         time.Time `json:"created_at"`
}

// WeightedEvent includes temporal decay weighting
type WeightedEvent struct {
	Event       ReputationEvent `json:"event"`
	TimeWeight  float64         `json:"time_weight"`
	TotalWeight float64         `json:"total_weight"`
}

// ComponentScores holds individual trust component scores
type ComponentScores struct {
	Reliability   float64 `json:"reliability"`
	Performance   float64 `json:"performance"`
	Availability  float64 `json:"availability"`
	Communication float64 `json:"communication"`
	Economic      float64 `json:"economic"`
}

// ValidationResult from cross-provider reputation verification
type ValidationResult struct {
	InternalScore  float64            `json:"internal_score"`
	ExternalScores map[string]float64 `json:"external_scores"`
	Discrepancy    float64            `json:"discrepancy"`
	Flagged        bool               `json:"flagged"`
	ValidatedAt    time.Time          `json:"validated_at"`
}

// ReputationRewards for honest reputation reporting
type ReputationRewards struct {
	EvaluatorID        string    `json:"evaluator_id"`
	TotalReward        float64   `json:"total_reward"`
	AccuracyScore      float64   `json:"accuracy_score"`
	EventsSubmitted    int       `json:"events_submitted"`
	RewardCalculatedAt time.Time `json:"reward_calculated_at"`
}

// NewReputationEngine creates a new reputation engine with default parameters
func NewReputationEngine(db *sql.DB) *ReputationEngine {
	return &ReputationEngine{
		db: db,
		weights: &ComponentWeights{
			Reliability:   0.30,
			Performance:   0.25,
			Availability:  0.20,
			Communication: 0.10,
			Economic:      0.15,
		},
		decayParameters: &DecayParameters{
			Lambda:      0.05, // 5% decay per day
			MinWeight:   0.01, // Minimum 1% weight for very old events
			PriorWeight: 10.0, // Bayesian prior weight
		},
		antiGamingRules: &AntiGamingRules{
			MinScoreVariance:        0.3,              // Require 30% variance in scores
			GamingPenaltyFactor:     0.1,              // 90% penalty for detected gaming
			RapidFireWindow:         10,               // Check last 10 events
			MinTimeBetweenEvents:    5 * time.Minute,  // Minimum 5 minutes between events
			MaxDiscrepancyThreshold: 0.4,              // 40% max discrepancy with external sources
		},
	}
}

// CalculateReputation computes Byzantine fault tolerant reputation score
func (re *ReputationEngine) CalculateReputation(ctx context.Context, providerID string) (*ReputationScore, error) {
	// 1. Gather all reputation events
	events, err := re.gatherReputationEvents(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// 2. Apply temporal decay to older events
	weightedEvents := re.applyTemporalDecay(events)

	// 3. Detect and filter gaming attempts
	filteredEvents := re.filterGamingAttempts(weightedEvents)

	// 4. Calculate component scores with cross-validation
	components := re.calculateComponentScores(filteredEvents)

	// 5. Compute overall score with weighted average
	overall := re.computeWeightedScore(components)

	// 6. Apply confidence interval based on sample size
	confidence := re.calculateConfidenceInterval(filteredEvents, overall)

	return &ReputationScore{
		Overall:            overall,
		Reliability:        components.Reliability,
		Performance:        components.Performance,
		Availability:       components.Availability,
		Communication:      components.Communication,
		Economic:          components.Economic,
		ConfidenceInterval: confidence,
		SampleSize:         len(filteredEvents),
		LastUpdated:        time.Now(),
	}, nil
}

// applyTemporalDecay applies exponential decay to older events
func (re *ReputationEngine) applyTemporalDecay(events []ReputationEvent) []WeightedEvent {
	now := time.Now()
	weightedEvents := make([]WeightedEvent, len(events))

	for i, event := range events {
		daysSinceEvent := now.Sub(event.CreatedAt).Hours() / 24

		// Exponential decay: weight = e^(-λt) where λ controls decay rate
		decayFactor := math.Exp(-re.decayParameters.Lambda * daysSinceEvent)

		// More recent events have higher weight
		timeWeight := math.Max(decayFactor, re.decayParameters.MinWeight)

		weightedEvents[i] = WeightedEvent{
			Event:       event,
			TimeWeight:  timeWeight,
			TotalWeight: timeWeight * event.ImpactScore,
		}
	}

	return weightedEvents
}

// filterGamingAttempts detects and filters reputation manipulation
func (re *ReputationEngine) filterGamingAttempts(events []WeightedEvent) []WeightedEvent {
	// 1. Detect collusion patterns
	collusionFiltered := re.detectCollusion(events)

	// 2. Filter rapid-fire fake reviews
	rapidFireFiltered := re.filterRapidFire(collusionFiltered)

	// 3. Remove obvious sybil attacks
	sybilFiltered := re.filterSybilAttacks(rapidFireFiltered)

	// 4. Check for wash trading patterns
	washTradingFiltered := re.filterWashTrading(sybilFiltered)

	return washTradingFiltered
}

// detectCollusion identifies collusion patterns in reputation events
func (re *ReputationEngine) detectCollusion(events []WeightedEvent) []WeightedEvent {
	// Group events by evaluator
	evaluatorGroups := make(map[string][]WeightedEvent)
	for _, event := range events {
		evaluatorGroups[event.Event.VerifiedBy] = append(
			evaluatorGroups[event.Event.VerifiedBy], event)
	}

	filtered := make([]WeightedEvent, 0, len(events))

	for evaluatorID, evaluatorEvents := range evaluatorGroups {
		// Check if this evaluator only gives extreme scores (all 1.0 or all 0.0)
		scores := extractScores(evaluatorEvents)
		variance := calculateVariance(scores)

		if variance < re.antiGamingRules.MinScoreVariance {
			// Likely gaming - reduce weight of all events from this evaluator
			for _, event := range evaluatorEvents {
				event.TotalWeight *= re.antiGamingRules.GamingPenaltyFactor
				filtered = append(filtered, event)
			}
		} else {
			filtered = append(filtered, evaluatorEvents...)
		}
	}

	return filtered
}

// filterRapidFire removes events that are too close in time
func (re *ReputationEngine) filterRapidFire(events []WeightedEvent) []WeightedEvent {
	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Event.CreatedAt.Before(events[j].Event.CreatedAt)
	})

	filtered := make([]WeightedEvent, 0, len(events))

	for i, event := range events {
		// Check if this event is too close in time to previous events from same evaluator
		tooRapid := false

		for j := i - 1; j >= 0 && j >= i-re.antiGamingRules.RapidFireWindow; j-- {
			if events[j].Event.VerifiedBy == event.Event.VerifiedBy {
				timeDiff := event.Event.CreatedAt.Sub(events[j].Event.CreatedAt)
				if timeDiff < re.antiGamingRules.MinTimeBetweenEvents {
					tooRapid = true
					break
				}
			}
		}

		if !tooRapid {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// filterSybilAttacks removes events from likely sybil accounts
func (re *ReputationEngine) filterSybilAttacks(events []WeightedEvent) []WeightedEvent {
	// Simple sybil detection: if evaluator has very few total events
	// and all events are for the same provider, likely sybil
	evaluatorStats := make(map[string]map[string]int)
	
	for _, event := range events {
		if evaluatorStats[event.Event.VerifiedBy] == nil {
			evaluatorStats[event.Event.VerifiedBy] = make(map[string]int)
		}
		evaluatorStats[event.Event.VerifiedBy][event.Event.ProviderID]++
	}

	filtered := make([]WeightedEvent, 0, len(events))

	for _, event := range events {
		stats := evaluatorStats[event.Event.VerifiedBy]
		totalEvents := 0
		for _, count := range stats {
			totalEvents += count
		}

		// If evaluator has < 5 total events and > 80% are for same provider, likely sybil
		if totalEvents < 5 {
			maxForOneProvider := 0
			for _, count := range stats {
				if count > maxForOneProvider {
					maxForOneProvider = count
				}
			}
			if float64(maxForOneProvider)/float64(totalEvents) > 0.8 {
				// Likely sybil - reduce weight
				event.TotalWeight *= re.antiGamingRules.GamingPenaltyFactor
			}
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// filterWashTrading removes events that appear to be wash trading
func (re *ReputationEngine) filterWashTrading(events []WeightedEvent) []WeightedEvent {
	// Simple wash trading detection: circular reputation boosting
	// This is a placeholder - real implementation would be more sophisticated
	return events
}

// calculateComponentScores computes individual trust component scores
func (re *ReputationEngine) calculateComponentScores(events []WeightedEvent) ComponentScores {
	reliabilityEvents := filterEventsByType(events, []string{
		"session_completed_successfully", "session_terminated_early",
		"performance_exceeded_sla", "performance_below_sla"})

	performanceEvents := filterEventsByType(events, []string{
		"performance_exceeded_sla", "performance_below_sla",
		"benchmark_score_updated"})

	availabilityEvents := filterEventsByType(events, []string{
		"uptime_penalty", "maintenance_window_exceeded",
		"heartbeat_missed"})

	communicationEvents := filterEventsByType(events, []string{
		"dispute_resolved_in_favor", "dispute_resolved_against",
		"response_time_exceeded", "exceptional_service"})

	economicEvents := filterEventsByType(events, []string{
		"payment_completed_on_time", "payment_delayed",
		"staking_bonus", "slashing_penalty"})

	return ComponentScores{
		Reliability:   re.calculateComponentScore(reliabilityEvents, 0.8),   // Default: 80%
		Performance:   re.calculateComponentScore(performanceEvents, 0.75),  // Default: 75%
		Availability:  re.calculateComponentScore(availabilityEvents, 0.9),  // Default: 90%
		Communication: re.calculateComponentScore(communicationEvents, 0.8), // Default: 80%
		Economic:      re.calculateComponentScore(economicEvents, 0.85),     // Default: 85%
	}
}

// calculateComponentScore computes a single component score with Bayesian smoothing
func (re *ReputationEngine) calculateComponentScore(events []WeightedEvent, defaultScore float64) float64 {
	if len(events) == 0 {
		return defaultScore // No data yet, use reasonable default
	}

	// Calculate weighted average of event impacts
	totalWeight := 0.0
	weightedSum := 0.0

	for _, event := range events {
		totalWeight += event.TotalWeight

		// Convert event impact to 0-1 score
		eventScore := re.eventImpactToScore(event.Event.ImpactScore)
		weightedSum += eventScore * event.TotalWeight
	}

	if totalWeight == 0 {
		return defaultScore
	}

	score := weightedSum / totalWeight

	// Apply smoothing for low sample sizes (Bayesian approach)
	sampleSize := float64(len(events))
	priorWeight := re.decayParameters.PriorWeight

	smoothedScore := (score*sampleSize + defaultScore*priorWeight) / (sampleSize + priorWeight)

	return math.Max(0.0, math.Min(1.0, smoothedScore))
}

// eventImpactToScore converts impact score to 0-1 range
func (re *ReputationEngine) eventImpactToScore(impactScore float64) float64 {
	// Sigmoid function to map impact to 0-1 range
	return 1.0 / (1.0 + math.Exp(-impactScore))
}

// computeWeightedScore calculates overall reputation score
func (re *ReputationEngine) computeWeightedScore(components ComponentScores) float64 {
	return components.Reliability*re.weights.Reliability +
		components.Performance*re.weights.Performance +
		components.Availability*re.weights.Availability +
		components.Communication*re.weights.Communication +
		components.Economic*re.weights.Economic
}

// calculateConfidenceInterval computes statistical confidence
func (re *ReputationEngine) calculateConfidenceInterval(events []WeightedEvent, score float64) float64 {
	if len(events) < 2 {
		return 1.0 // No confidence with < 2 samples
	}

	// Calculate standard error
	variance := 0.0
	for _, event := range events {
		eventScore := re.eventImpactToScore(event.Event.ImpactScore)
		diff := eventScore - score
		variance += diff * diff * event.TotalWeight
	}

	totalWeight := 0.0
	for _, event := range events {
		totalWeight += event.TotalWeight
	}

	if totalWeight == 0 {
		return 1.0
	}

	variance /= totalWeight
	stdError := math.Sqrt(variance / float64(len(events)))

	return stdError
}

// Helper functions

func extractScores(events []WeightedEvent) []float64 {
	scores := make([]float64, len(events))
	for i, event := range events {
		scores[i] = event.Event.ImpactScore
	}
	return scores
}

func calculateVariance(scores []float64) float64 {
	if len(scores) < 2 {
		return 0.0
	}

	mean := 0.0
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	variance := 0.0
	for _, score := range scores {
		diff := score - mean
		variance += diff * diff
	}
	variance /= float64(len(scores) - 1)

	return variance
}

func filterEventsByType(events []WeightedEvent, types []string) []WeightedEvent {
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	filtered := make([]WeightedEvent, 0)
	for _, event := range events {
		if typeMap[event.Event.EventType] {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Database interaction methods (simplified for this example)

func (re *ReputationEngine) gatherReputationEvents(ctx context.Context, providerID string) ([]ReputationEvent, error) {
	// This would query the database for reputation events
	// For now, return empty slice as placeholder
	return []ReputationEvent{}, nil
}

// internal/tokenomics/usd.go
package tokenomics

import (
	"fmt"
	"log"
	"time"
)

// USDTokenomicsEngine handles USD-based payments and verifier rewards
// NO TOKEN SALES - enterprises pay with USD, verifiers earn USD
type USDTokenomicsEngine struct {
	// Verifier network
	Verifiers map[string]*Verifier `json:"verifiers"`
	
	// USD balances (not tokens!)
	USDBalances map[string]float64 `json:"usd_balances"`
	
	// Staking system (USD deposits, not tokens)
	StakePositions map[string]*StakePosition `json:"stake_positions"`
	
	// Fee structure
	TransactionFeeBPS int     `json:"transaction_fee_bps"` // 25 = 0.25%
	VerifierFeeShare  float64 `json:"verifier_fee_share"`  // 0.6 = 60%
	ProtocolFeeShare  float64 `json:"protocol_fee_share"`  // 0.4 = 40%
	
	// Minimum stake requirements
	MinimumStakeUSD float64 `json:"minimum_stake_usd"`
	
	// Statistics
	TotalStakedUSD    float64 `json:"total_staked_usd"`
	TotalFeesEarned   float64 `json:"total_fees_earned"`
	TotalTransactions int     `json:"total_transactions"`
}

// Verifier represents a verifier in the network
type Verifier struct {
	VerifierID     string    `json:"verifier_id"`
	StakeAmountUSD float64   `json:"stake_amount_usd"`
	JoinedAt       time.Time `json:"joined_at"`
	RewardsEarned  float64   `json:"rewards_earned"`
	SlashCount     int       `json:"slash_count"`
	IsActive       bool      `json:"is_active"`
}

// StakePosition represents a USD stake position
type StakePosition struct {
	StakerID        string     `json:"staker_id"`
	AmountUSD       float64    `json:"amount_usd"`
	StakeTimestamp  time.Time  `json:"stake_timestamp"`
	UnlockTimestamp *time.Time `json:"unlock_timestamp,omitempty"`
	SlashCount      int        `json:"slash_count"`
	RewardsEarned   float64    `json:"rewards_earned"`
}

// USDPaymentResult represents the result of a USD payment
type USDPaymentResult struct {
	TransactionID     string  `json:"transaction_id"`
	CustomerID        string  `json:"customer_id"`
	ProviderID        string  `json:"provider_id"`
	AmountUSD         float64 `json:"amount_usd"`
	TransactionFeeUSD float64 `json:"transaction_fee_usd"`
	VerifierFeesUSD   float64 `json:"verifier_fees_usd"`
	ProtocolFeesUSD   float64 `json:"protocol_fees_usd"`
	Success           bool    `json:"success"`
	Timestamp         time.Time `json:"timestamp"`
}

// VerificationReward represents a reward for participating in consensus
type VerificationReward struct {
	VerifierID        string    `json:"verifier_id"`
	ProposalID        string    `json:"proposal_id"`
	BaseRewardUSD     float64   `json:"base_reward_usd"`
	PerformanceBonus  float64   `json:"performance_bonus"`
	StakeMultiplier   float64   `json:"stake_multiplier"`
	FinalRewardUSD    float64   `json:"final_reward_usd"`
	RewardTimestamp   time.Time `json:"reward_timestamp"`
}

// SlashingEvent represents a slashing event for Byzantine behavior
type SlashingEvent struct {
	VerifierID      string    `json:"verifier_id"`
	ProposalID      string    `json:"proposal_id"`
	SlashAmountUSD  float64   `json:"slash_amount_usd"`
	Reason          string    `json:"reason"`
	EvidenceHash    string    `json:"evidence_hash"`
	SlashTimestamp  time.Time `json:"slash_timestamp"`
}

// NewUSDTokenomicsEngine creates a new USD tokenomics engine
func NewUSDTokenomicsEngine() *USDTokenomicsEngine {
	return &USDTokenomicsEngine{
		Verifiers:         make(map[string]*Verifier),
		USDBalances:       make(map[string]float64),
		StakePositions:    make(map[string]*StakePosition),
		TransactionFeeBPS: 25,  // 0.25%
		VerifierFeeShare:  0.6, // 60%
		ProtocolFeeShare:  0.4, // 40%
		MinimumStakeUSD:   10000.0, // $10,000 minimum stake
	}
}

// StakeUSD stakes USD to become a verifier (NO TOKENS!)
func (ute *USDTokenomicsEngine) StakeUSD(stakerID string, amountUSD float64) (*StakePosition, error) {
	if amountUSD < ute.MinimumStakeUSD {
		return nil, fmt.Errorf("minimum stake is $%.2f USD", ute.MinimumStakeUSD)
	}
	
	// Check if staker has sufficient USD balance
	if ute.USDBalances[stakerID] < amountUSD {
		return nil, fmt.Errorf("insufficient USD balance: $%.2f required, $%.2f available", 
			amountUSD, ute.USDBalances[stakerID])
	}
	
	// Deduct USD from balance
	ute.USDBalances[stakerID] -= amountUSD
	
	// Create stake position
	stakePosition := &StakePosition{
		StakerID:       stakerID,
		AmountUSD:      amountUSD,
		StakeTimestamp: time.Now(),
	}
	
	ute.StakePositions[stakerID] = stakePosition
	ute.TotalStakedUSD += amountUSD
	
	// Create verifier
	verifier := &Verifier{
		VerifierID:     stakerID,
		StakeAmountUSD: amountUSD,
		JoinedAt:       time.Now(),
		RewardsEarned:  0.0,
		SlashCount:     0,
		IsActive:       true,
	}
	
	ute.Verifiers[stakerID] = verifier
	
	log.Printf("Verifier %s staked $%.2f USD", stakerID, amountUSD)
	return stakePosition, nil
}

// UnstakeUSD unstakes USD after cooldown period
func (ute *USDTokenomicsEngine) UnstakeUSD(stakerID string) error {
	stakePosition, exists := ute.StakePositions[stakerID]
	if !exists {
		return fmt.Errorf("no stake position found for %s", stakerID)
	}
	
	// Check if in cooldown period
	if stakePosition.UnlockTimestamp != nil && time.Now().Before(*stakePosition.UnlockTimestamp) {
		remainingTime := stakePosition.UnlockTimestamp.Sub(time.Now())
		return fmt.Errorf("cooldown period: %.1f hours remaining", remainingTime.Hours())
	}
	
	// Return USD to balance
	ute.USDBalances[stakerID] += stakePosition.AmountUSD
	ute.TotalStakedUSD -= stakePosition.AmountUSD
	
	// Remove verifier
	delete(ute.Verifiers, stakerID)
	delete(ute.StakePositions, stakerID)
	
	log.Printf("Verifier %s unstaked $%.2f USD", stakerID, stakePosition.AmountUSD)
	return nil
}

// InitiateUnstaking starts the unstaking cooldown period
func (ute *USDTokenomicsEngine) InitiateUnstaking(stakerID string, cooldownHours int) error {
	stakePosition, exists := ute.StakePositions[stakerID]
	if !exists {
		return fmt.Errorf("no stake position found for %s", stakerID)
	}
	
	unlockTime := time.Now().Add(time.Duration(cooldownHours) * time.Hour)
	stakePosition.UnlockTimestamp = &unlockTime
	
	log.Printf("Unstaking initiated for %s, cooldown: %d hours", stakerID, cooldownHours)
	return nil
}

// ProcessUSDPayment processes a USD payment for compute services
func (ute *USDTokenomicsEngine) ProcessUSDPayment(customerID, providerID string, amountUSD float64) (*USDPaymentResult, error) {
	// Calculate transaction fee
	transactionFeeUSD := amountUSD * float64(ute.TransactionFeeBPS) / 10000.0
	
	// Fee distribution
	verifierFeesUSD := transactionFeeUSD * ute.VerifierFeeShare
	protocolFeesUSD := transactionFeeUSD * ute.ProtocolFeeShare
	
	// Check customer has sufficient USD balance
	if ute.USDBalances[customerID] < amountUSD {
		return nil, fmt.Errorf("insufficient USD balance: $%.2f required, $%.2f available", 
			amountUSD, ute.USDBalances[customerID])
	}
	
	// Process payment
	ute.USDBalances[customerID] -= amountUSD
	ute.USDBalances[providerID] += (amountUSD - transactionFeeUSD)
	
	// Distribute verifier fees among active verifiers
	activeVerifiers := ute.getActiveVerifiers()
	if len(activeVerifiers) > 0 {
		feePerVerifier := verifierFeesUSD / float64(len(activeVerifiers))
		for _, verifierID := range activeVerifiers {
			ute.USDBalances[verifierID] += feePerVerifier
			ute.Verifiers[verifierID].RewardsEarned += feePerVerifier
		}
	}
	
	// Update statistics
	ute.TotalFeesEarned += transactionFeeUSD
	ute.TotalTransactions++
	
	transactionID := fmt.Sprintf("usd_tx_%d_%s", time.Now().Unix(), customerID)
	
	result := &USDPaymentResult{
		TransactionID:     transactionID,
		CustomerID:        customerID,
		ProviderID:        providerID,
		AmountUSD:         amountUSD,
		TransactionFeeUSD: transactionFeeUSD,
		VerifierFeesUSD:   verifierFeesUSD,
		ProtocolFeesUSD:   protocolFeesUSD,
		Success:           true,
		Timestamp:         time.Now(),
	}
	
	log.Printf("Processed USD payment: $%.2f from %s to %s (fee: $%.2f)", 
		amountUSD, customerID, providerID, transactionFeeUSD)
	
	return result, nil
}

// CalculateVerificationReward calculates reward for participating in consensus
func (ute *USDTokenomicsEngine) CalculateVerificationReward(verifierID, proposalID string, 
	consensusParticipation map[string]interface{}) *VerificationReward {
	
	// Base reward from transaction fees (distributed among verifiers)
	baseRewardUSD := ute.TotalFeesEarned / float64(len(ute.getActiveVerifiers()))
	if baseRewardUSD == 0 {
		baseRewardUSD = 1.0 // Minimum reward
	}
	
	// Performance multiplier based on accuracy
	voteCorrect := getBool(consensusParticipation, "vote_correct", true)
	responseTimeMS := getFloat64(consensusParticipation, "response_time_ms", 1000.0)
	
	performanceBonus := 0.0
	if voteCorrect {
		performanceBonus = 0.2 // 20% bonus for correct vote
	} else {
		performanceBonus = -0.5 // 50% penalty for incorrect vote
	}
	
	// Response time bonus (faster response = higher reward)
	if responseTimeMS < 500 {
		performanceBonus += 0.1 // 10% bonus for fast response
	}
	
	// Stake multiplier (higher stake = higher reward share)
	verifier, exists := ute.Verifiers[verifierID]
	stakeMultiplier := 1.0
	if exists {
		stakeRatio := verifier.StakeAmountUSD / ute.TotalStakedUSD
		stakeMultiplier = 0.8 + (stakeRatio * 0.4) // 0.8x to 1.2x based on stake
	}
	
	// Final reward calculation
	finalRewardUSD := baseRewardUSD * (1.0 + performanceBonus) * stakeMultiplier
	
	reward := &VerificationReward{
		VerifierID:        verifierID,
		ProposalID:        proposalID,
		BaseRewardUSD:     baseRewardUSD,
		PerformanceBonus:  performanceBonus,
		StakeMultiplier:   stakeMultiplier,
		FinalRewardUSD:    finalRewardUSD,
		RewardTimestamp:   time.Now(),
	}
	
	// Distribute reward
	ute.USDBalances[verifierID] += finalRewardUSD
	ute.Verifiers[verifierID].RewardsEarned += finalRewardUSD
	
	log.Printf("Verifier %s earned $%.2f reward (base: $%.2f, bonus: %.1f%%, stake: %.2fx)", 
		verifierID, finalRewardUSD, baseRewardUSD, performanceBonus*100, stakeMultiplier)
	
	return reward
}

// SlashVerifier slashes verifier for Byzantine behavior
func (ute *USDTokenomicsEngine) SlashVerifier(verifierID, proposalID, reason, evidenceHash string) error {
	verifier, exists := ute.Verifiers[verifierID]
	if !exists {
		return fmt.Errorf("verifier %s not found", verifierID)
	}
	
	stakePosition, exists := ute.StakePositions[verifierID]
	if !exists {
		return fmt.Errorf("stake position not found for verifier %s", verifierID)
	}
	
	// Calculate slash amount (5% of stake)
	slashAmountUSD := stakePosition.AmountUSD * 0.05
	
	// Reduce staked amount
	stakePosition.AmountUSD -= slashAmountUSD
	stakePosition.SlashCount++
	verifier.SlashCount++
	verifier.StakeAmountUSD = stakePosition.AmountUSD
	
	// Update total staked amount
	ute.TotalStakedUSD -= slashAmountUSD
	
	// Record slashing event
// 	// slashingEvent := slashingEvent := &SlashingEvent{SlashingEvent{
// 		VerifierID:     verifierID,
// 		ProposalID:     proposalID,
// 		SlashAmountUSD: slashAmountUSD,
// 		Reason:         reason,
// 		EvidenceHash:   evidenceHash,
// 		SlashTimestamp: time.Now(),
// 	}
	
	// Remove verifier if stake falls below minimum
	if stakePosition.AmountUSD < ute.MinimumStakeUSD {
		delete(ute.Verifiers, verifierID)
		delete(ute.StakePositions, verifierID)
		verifier.IsActive = false
		
		log.Printf("Verifier %s slashed $%.2f and removed from network (stake below minimum)", 
			verifierID, slashAmountUSD)
	} else {
		log.Printf("Verifier %s slashed $%.2f for %s", verifierID, slashAmountUSD, reason)
	}
	
	return nil
}

// AddUSDBalance adds USD to an account balance
func (ute *USDTokenomicsEngine) AddUSDBalance(accountID string, amountUSD float64) {
	ute.USDBalances[accountID] += amountUSD
	log.Printf("Added $%.2f USD to account %s", amountUSD, accountID)
}

// GetUSDBalance returns the USD balance for an account
func (ute *USDTokenomicsEngine) GetUSDBalance(accountID string) float64 {
	return ute.USDBalances[accountID]
}

// getActiveVerifiers returns list of active verifier IDs
func (ute *USDTokenomicsEngine) getActiveVerifiers() []string {
	var activeVerifiers []string
	for verifierID, verifier := range ute.Verifiers {
		if verifier.IsActive {
			activeVerifiers = append(activeVerifiers, verifierID)
		}
	}
	return activeVerifiers
}

// GetTokenomicsStats returns comprehensive tokenomics statistics
func (ute *USDTokenomicsEngine) GetTokenomicsStats() *USDTokenomicsStats {
	activeVerifiers := ute.getActiveVerifiers()
	
	// Calculate average stake
	avgStakeUSD := 0.0
	if len(activeVerifiers) > 0 {
		totalStake := 0.0
		for _, verifierID := range activeVerifiers {
			totalStake += ute.Verifiers[verifierID].StakeAmountUSD
		}
		avgStakeUSD = totalStake / float64(len(activeVerifiers))
	}
	
	// Calculate total rewards distributed
	totalRewardsUSD := 0.0
	for _, verifier := range ute.Verifiers {
		totalRewardsUSD += verifier.RewardsEarned
	}
	
	return &USDTokenomicsStats{
		TotalVerifiers:     len(activeVerifiers),
		TotalStakedUSD:     ute.TotalStakedUSD,
		AverageStakeUSD:    avgStakeUSD,
		TotalFeesEarned:    ute.TotalFeesEarned,
		TotalRewardsUSD:    totalRewardsUSD,
		TotalTransactions:  ute.TotalTransactions,
		TransactionFeeBPS:  ute.TransactionFeeBPS,
		VerifierFeeShare:   ute.VerifierFeeShare,
		ProtocolFeeShare:   ute.ProtocolFeeShare,
		MinimumStakeUSD:    ute.MinimumStakeUSD,
	}
}

// USDTokenomicsStats represents tokenomics statistics
type USDTokenomicsStats struct {
	TotalVerifiers     int     `json:"total_verifiers"`
	TotalStakedUSD     float64 `json:"total_staked_usd"`
	AverageStakeUSD    float64 `json:"average_stake_usd"`
	TotalFeesEarned    float64 `json:"total_fees_earned"`
	TotalRewardsUSD    float64 `json:"total_rewards_usd"`
	TotalTransactions  int     `json:"total_transactions"`
	TransactionFeeBPS  int     `json:"transaction_fee_bps"`
	VerifierFeeShare   float64 `json:"verifier_fee_share"`
	ProtocolFeeShare   float64 `json:"protocol_fee_share"`
	MinimumStakeUSD    float64 `json:"minimum_stake_usd"`
}

// Helper functions
func getBool(data map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := data[key]; ok {
		if boolVal, ok := value.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

func getFloat64(data map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := data[key]; ok {
		if floatVal, ok := value.(float64); ok {
			return floatVal
		}
		if intVal, ok := value.(int); ok {
			return float64(intVal)
		}
	}
	return defaultValue
}

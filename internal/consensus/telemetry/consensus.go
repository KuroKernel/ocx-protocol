// internal/consensus/telemetry/consensus.go
package telemetry

import (
	"fmt"
	"sync"
	"time"
)

// ConsensusStatus represents the status of a consensus decision
type ConsensusStatus string

const (
	Proposed         ConsensusStatus = "proposed"
	Voting           ConsensusStatus = "voting"
	Confirmed        ConsensusStatus = "confirmed"
	Rejected         ConsensusStatus = "rejected"
	ByzantineDetected ConsensusStatus = "byzantine_detected"
)

// VerificationProposal represents a proposal for consensus on telemetry verification
type VerificationProposal struct {
	ProposalID      string                 `json:"proposal_id"`
	WorkloadID      string                 `json:"workload_id"`
	ProposerID      string                 `json:"proposer_id"`
	TelemetryEvents []*TelemetryEvent      `json:"telemetry_events"`
	SLAAssessment   map[string]interface{} `json:"sla_assessment"`
	Timestamp       time.Time              `json:"timestamp"`
	BlockHeight     int                    `json:"block_height"`
	PreviousHash    string                 `json:"previous_hash"`
}

// ComputeProposalHash computes hash for consensus voting
func (vp *VerificationProposal) ComputeProposalHash() string {
	eventsHash := vp.computeEventsHash()
	data := fmt.Sprintf("%s:%s:%s:%s:%d:%d:%s",
		vp.ProposalID,
		vp.WorkloadID,
		vp.ProposerID,
		eventsHash,
		vp.SLAAssessment["uptime_percentage"],
		vp.Timestamp.UnixMilli(),
		vp.BlockHeight,
		vp.PreviousHash,
	)
	return fmt.Sprintf("%x", data)
}

// computeEventsHash computes merkle-like hash of all telemetry events
func (vp *VerificationProposal) computeEventsHash() string {
	eventHashes := make([]string, len(vp.TelemetryEvents))
	for i, event := range vp.TelemetryEvents {
		eventHashes[i] = event.ComputeHash()
	}
	
	// Simple concatenation for demo (in production, use proper Merkle tree)
	combined := ""
	for _, hash := range eventHashes {
		combined += hash
	}
	return fmt.Sprintf("%x", combined)
}

// ByzantineConsensus represents the Byzantine fault tolerant consensus system
type ByzantineConsensus struct {
	VerifierNodes      map[string]*VerifierNode `json:"verifier_nodes"`
	ByzantineTolerance float64                  `json:"byzantine_tolerance"`
	Proposals          map[string]*VerificationProposal `json:"proposals"`
	Votes              map[string][]*Vote       `json:"votes"`
	ConsensusResults   map[string]ConsensusStatus `json:"consensus_results"`
	BlockHeight        int                      `json:"block_height"`
	LastBlockHash      string                   `json:"last_block_hash"`
	mutex              sync.RWMutex
}

// NewByzantineConsensus creates a new Byzantine consensus system
func NewByzantineConsensus(byzantineTolerance float64) *ByzantineConsensus {
	return &ByzantineConsensus{
		VerifierNodes:      make(map[string]*VerifierNode),
		ByzantineTolerance: byzantineTolerance,
		Proposals:          make(map[string]*VerificationProposal),
		Votes:              make(map[string][]*Vote),
		ConsensusResults:   make(map[string]ConsensusStatus),
		BlockHeight:        0,
		LastBlockHash:      "genesis",
	}
}

// AddVerifierNode adds a verifier node to the consensus network
func (bc *ByzantineConsensus) AddVerifierNode(node *VerifierNode) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	bc.VerifierNodes[node.NodeID] = node
	
	// Connect to other nodes
	for otherNodeID := range bc.VerifierNodes {
		if otherNodeID != node.NodeID {
			node.PeerNodes[otherNodeID] = true
			bc.VerifierNodes[otherNodeID].PeerNodes[node.NodeID] = true
		}
	}
}

// SimulateByzantineNodes simulates Byzantine behavior in a percentage of nodes
func (bc *ByzantineConsensus) SimulateByzantineNodes(percentage float64) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	nodeList := make([]*VerifierNode, 0, len(bc.VerifierNodes))
	for _, node := range bc.VerifierNodes {
		nodeList = append(nodeList, node)
	}
	
	byzantineCount := int(float64(len(nodeList)) * percentage)
	for i := 0; i < byzantineCount && i < len(nodeList); i++ {
		nodeList[i].IsByzantine = true
	}
}

// VerifyWorkloadConsensus runs Byzantine consensus to verify workload completion
func (bc *ByzantineConsensus) VerifyWorkloadConsensus(workload *ComputeWorkload, telemetryEvents []*TelemetryEvent) (*ConsensusResult, error) {
	if len(bc.VerifierNodes) < 4 {
		return nil, fmt.Errorf("need at least 4 verifier nodes for Byzantine consensus")
	}
	
	// Select a proposer (for simplicity, use first node)
	var proposerNode *VerifierNode
	for _, node := range bc.VerifierNodes {
		proposerNode = node
		break
	}
	
	// Create verification proposal
	proposal := bc.createVerificationProposal(proposerNode, workload, telemetryEvents)
	
	bc.mutex.Lock()
	bc.Proposals[proposal.ProposalID] = proposal
	bc.mutex.Unlock()
	
	// Collect votes from all nodes
	votes := make([]*Vote, 0, len(bc.VerifierNodes))
	for _, node := range bc.VerifierNodes {
		vote := node.VoteOnProposal(proposal, workload)
		votes = append(votes, vote)
	}
	
	// Store votes
	bc.mutex.Lock()
	bc.Votes[proposal.ProposalID] = votes
	bc.mutex.Unlock()
	
	// Determine consensus
	consensusResult := bc.determineConsensus(proposal.ProposalID)
	
	bc.mutex.Lock()
	bc.ConsensusResults[proposal.ProposalID] = consensusResult
	bc.mutex.Unlock()
	
	// Update blockchain state
	if consensusResult == Confirmed {
		bc.mutex.Lock()
		bc.BlockHeight++
		bc.LastBlockHash = proposal.ComputeProposalHash()
		bc.mutex.Unlock()
	}
	
	// Calculate vote breakdown
	voteBreakdown := bc.getVoteBreakdown(proposal.ProposalID)
	
	return &ConsensusResult{
		ProposalID:      proposal.ProposalID,
		ConsensusStatus: consensusResult,
		VotesCollected:  len(votes),
		VoteBreakdown:   voteBreakdown,
		SLAAssessment:   proposal.SLAAssessment,
		BlockHeight:     bc.BlockHeight,
		ByzantineDetected: consensusResult == ByzantineDetected,
	}, nil
}

// createVerificationProposal creates a verification proposal
func (bc *ByzantineConsensus) createVerificationProposal(proposer *VerifierNode, workload *ComputeWorkload, events []*TelemetryEvent) *VerificationProposal {
	// Assess SLA compliance
	slaAssessment := proposer.AssessSLACompliance(workload, events)
	
	proposalID := fmt.Sprintf("proposal_%s_%s_%d", workload.WorkloadID, proposer.NodeID, time.Now().Unix())
	
	return &VerificationProposal{
		ProposalID:      proposalID,
		WorkloadID:      workload.WorkloadID,
		ProposerID:      proposer.NodeID,
		TelemetryEvents: events,
		SLAAssessment:   slaAssessment,
		Timestamp:       time.Now(),
		BlockHeight:     bc.BlockHeight,
		PreviousHash:    bc.LastBlockHash,
	}
}

// determineConsensus determines consensus status based on collected votes
func (bc *ByzantineConsensus) determineConsensus(proposalID string) ConsensusStatus {
	bc.mutex.RLock()
	votes := bc.Votes[proposalID]
	bc.mutex.RUnlock()
	
	if len(votes) == 0 {
		return Rejected
	}
	
	totalStake := 0.0
	acceptStake := 0.0
	
	for _, vote := range votes {
		totalStake += vote.StakeWeight
		if vote.Vote {
			acceptStake += vote.StakeWeight
		}
	}
	
	// Calculate percentages
	acceptPercentage := acceptStake / totalStake
	rejectPercentage := 1.0 - acceptPercentage
	
	// Byzantine fault tolerance: need >66% agreement to confirm
	requiredThreshold := 1.0 - bc.ByzantineTolerance
	
	if acceptPercentage > requiredThreshold {
		return Confirmed
	} else if rejectPercentage > requiredThreshold {
		return Rejected
	} else {
		// Inconclusive - possible Byzantine attack
		return ByzantineDetected
	}
}

// getVoteBreakdown gets detailed breakdown of voting results
func (bc *ByzantineConsensus) getVoteBreakdown(proposalID string) *VoteBreakdown {
	bc.mutex.RLock()
	votes := bc.Votes[proposalID]
	bc.mutex.RUnlock()
	
	acceptVotes := 0
	rejectVotes := 0
	totalStake := 0.0
	acceptStake := 0.0
	rejectReasons := make(map[string]int)
	
	for _, vote := range votes {
		totalStake += vote.StakeWeight
		if vote.Vote {
			acceptVotes++
			acceptStake += vote.StakeWeight
		} else {
			rejectVotes++
			rejectReasons[vote.Reasoning]++
		}
	}
	
	acceptPercentage := 0.0
	if totalStake > 0 {
		acceptPercentage = (acceptStake / totalStake) * 100
	}
	
	return &VoteBreakdown{
		TotalNodes:      len(votes),
		AcceptVotes:     acceptVotes,
		RejectVotes:     rejectVotes,
		TotalStake:      totalStake,
		AcceptStake:     acceptStake,
		AcceptPercentage: acceptPercentage,
		RejectReasons:   rejectReasons,
	}
}

// GetNetworkStatus gets overall network health and consensus history
func (bc *ByzantineConsensus) GetNetworkStatus() *NetworkStatus {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	byzantineNodes := make([]string, 0)
	for _, node := range bc.VerifierNodes {
		if node.IsByzantine {
			byzantineNodes = append(byzantineNodes, node.NodeID)
		}
	}
	
	consensusStats := make(map[string]int)
	for _, status := range bc.ConsensusResults {
		consensusStats[string(status)]++
	}
	
	byzantinePercentage := 0.0
	if len(bc.VerifierNodes) > 0 {
		byzantinePercentage = float64(len(byzantineNodes)) / float64(len(bc.VerifierNodes)) * 100
	}
	
	return &NetworkStatus{
		TotalNodes:        len(bc.VerifierNodes),
		ByzantineNodes:    byzantineNodes,
		ByzantinePercentage: byzantinePercentage,
		BlockHeight:       bc.BlockHeight,
		TotalProposals:    len(bc.Proposals),
		ConsensusStats:    consensusStats,
		ByzantineTolerance: bc.ByzantineTolerance * 100,
	}
}

// ConsensusResult represents the result of a consensus decision
type ConsensusResult struct {
	ProposalID       string                 `json:"proposal_id"`
	ConsensusStatus  ConsensusStatus        `json:"consensus_status"`
	VotesCollected   int                    `json:"votes_collected"`
	VoteBreakdown    *VoteBreakdown         `json:"vote_breakdown"`
	SLAAssessment    map[string]interface{} `json:"sla_assessment"`
	BlockHeight      int                    `json:"block_height"`
	ByzantineDetected bool                  `json:"byzantine_detected"`
}

// VoteBreakdown represents detailed voting results
type VoteBreakdown struct {
	TotalNodes      int            `json:"total_nodes"`
	AcceptVotes     int            `json:"accept_votes"`
	RejectVotes     int            `json:"reject_votes"`
	TotalStake      float64        `json:"total_stake"`
	AcceptStake     float64        `json:"accept_stake"`
	AcceptPercentage float64       `json:"accept_percentage"`
	RejectReasons   map[string]int `json:"reject_reasons"`
}

// NetworkStatus represents the overall network health
type NetworkStatus struct {
	TotalNodes         int            `json:"total_nodes"`
	ByzantineNodes     []string       `json:"byzantine_nodes"`
	ByzantinePercentage float64       `json:"byzantine_percentage"`
	BlockHeight        int            `json:"block_height"`
	TotalProposals     int            `json:"total_proposals"`
	ConsensusStats     map[string]int `json:"consensus_stats"`
	ByzantineTolerance float64        `json:"byzantine_tolerance"`
}

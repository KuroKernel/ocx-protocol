package scaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ClusterConfig defines configuration for cluster management
type ClusterConfig struct {
	// Node identification
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"` // "primary", "secondary", "worker"

	// Discovery settings
	DiscoveryEnabled bool     `json:"discovery_enabled"`
	DiscoveryServers []string `json:"discovery_servers"`

	// Heartbeat settings
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	HeartbeatTimeout  time.Duration `json:"heartbeat_timeout"`

	// Cluster settings
	ClusterName        string  `json:"cluster_name"`
	MinNodes           int     `json:"min_nodes"`
	MaxNodes           int     `json:"max_nodes"`
	AutoScale          bool    `json:"auto_scale"`
	ScaleUpThreshold   float64 `json:"scale_up_threshold"`   // CPU/Memory threshold for scaling up
	ScaleDownThreshold float64 `json:"scale_down_threshold"` // CPU/Memory threshold for scaling down

	// Communication settings
	ListenAddress    string `json:"listen_address"`
	AdvertiseAddress string `json:"advertise_address"`

	// Data replication
	ReplicationEnabled bool `json:"replication_enabled"`
	ReplicationFactor  int  `json:"replication_factor"`
}

// ClusterNode represents a node in the cluster
type ClusterNode struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Address       string            `json:"address"`
	AdvertiseAddr string            `json:"advertise_addr"`
	Status        string            `json:"status"` // "active", "inactive", "joining", "leaving"
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Metadata      map[string]string `json:"metadata"`
	Load          NodeLoad          `json:"load"`
	Capabilities  []string          `json:"capabilities"`
}

// NodeLoad represents the current load of a node
type NodeLoad struct {
	CPUUsage          float64   `json:"cpu_usage"`
	MemoryUsage       float64   `json:"memory_usage"`
	DiskUsage         float64   `json:"disk_usage"`
	NetworkIO         float64   `json:"network_io"`
	ActiveConnections int       `json:"active_connections"`
	Timestamp         time.Time `json:"timestamp"`
}

// ClusterManager manages cluster operations
type ClusterManager struct {
	config   ClusterConfig
	nodes    map[string]*ClusterNode
	nodesMu  sync.RWMutex
	leader   string
	leaderMu sync.RWMutex

	// Communication
	httpClient *http.Client
	server     *http.Server

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Event handlers
	onNodeJoin     func(*ClusterNode)
	onNodeLeave    func(*ClusterNode)
	onLeaderChange func(string, string) // old leader, new leader
}

// ClusterEvent represents a cluster event
type ClusterEvent struct {
	Type      string      `json:"type"` // "node_join", "node_leave", "leader_change", "load_update"
	NodeID    string      `json:"node_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(config ClusterConfig) (*ClusterManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &ClusterManager{
		config:     config,
		nodes:      make(map[string]*ClusterNode),
		httpClient: &http.Client{Timeout: config.HeartbeatTimeout},
		ctx:        ctx,
		cancel:     cancel,
	}

	// Add self to cluster
	self := &ClusterNode{
		ID:            config.NodeID,
		Type:          config.NodeType,
		Address:       config.ListenAddress,
		AdvertiseAddr: config.AdvertiseAddress,
		Status:        "active",
		LastHeartbeat: time.Now(),
		Metadata:      make(map[string]string),
		Load:          NodeLoad{Timestamp: time.Now()},
		Capabilities:  []string{"http", "grpc"},
	}
	cm.nodes[config.NodeID] = self

	// Set as leader if primary node
	if config.NodeType == "primary" {
		cm.leader = config.NodeID
	}

	// Start HTTP server for cluster communication
	if err := cm.startHTTPServer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start HTTP server: %v", err)
	}

	// Start background tasks
	cm.startHeartbeat()
	cm.startNodeDiscovery()
	cm.startLoadMonitoring()
	cm.startAutoScaling()

	return cm, nil
}

// GetNodes returns all nodes in the cluster
func (cm *ClusterManager) GetNodes() map[string]*ClusterNode {
	cm.nodesMu.RLock()
	defer cm.nodesMu.RUnlock()

	nodes := make(map[string]*ClusterNode)
	for id, node := range cm.nodes {
		nodes[id] = node
	}
	return nodes
}

// GetNode returns a specific node
func (cm *ClusterManager) GetNode(nodeID string) (*ClusterNode, bool) {
	cm.nodesMu.RLock()
	defer cm.nodesMu.RUnlock()

	node, exists := cm.nodes[nodeID]
	return node, exists
}

// GetLeader returns the current leader node ID
func (cm *ClusterManager) GetLeader() string {
	cm.leaderMu.RLock()
	defer cm.leaderMu.RUnlock()
	return cm.leader
}

// IsLeader returns true if this node is the leader
func (cm *ClusterManager) IsLeader() bool {
	return cm.GetLeader() == cm.config.NodeID
}

// AddNode adds a node to the cluster
func (cm *ClusterManager) AddNode(node *ClusterNode) {
	cm.nodesMu.Lock()
	defer cm.nodesMu.Unlock()

	cm.nodes[node.ID] = node
	log.Printf("Node %s added to cluster", node.ID)

	// Trigger event handler
	if cm.onNodeJoin != nil {
		cm.onNodeJoin(node)
	}
}

// RemoveNode removes a node from the cluster
func (cm *ClusterManager) RemoveNode(nodeID string) {
	cm.nodesMu.Lock()
	defer cm.nodesMu.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		delete(cm.nodes, nodeID)
		log.Printf("Node %s removed from cluster", nodeID)

		// Trigger event handler
		if cm.onNodeLeave != nil {
			cm.onNodeLeave(node)
		}

		// Handle leader change if needed
		if nodeID == cm.leader {
			cm.electNewLeader()
		}
	}
}

// UpdateNodeLoad updates the load information for a node
func (cm *ClusterManager) UpdateNodeLoad(nodeID string, load NodeLoad) {
	cm.nodesMu.Lock()
	defer cm.nodesMu.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		node.Load = load
		node.Load.Timestamp = time.Now()
	}
}

// BroadcastMessage sends a message to all nodes in the cluster
func (cm *ClusterManager) BroadcastMessage(message interface{}) error {
	cm.nodesMu.RLock()
	nodes := make([]*ClusterNode, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		if node.ID != cm.config.NodeID && node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	cm.nodesMu.RUnlock()

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Send message to all nodes concurrently
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(n *ClusterNode) {
			defer wg.Done()
			cm.sendMessageToNode(n, messageData)
		}(node)
	}

	wg.Wait()
	return nil
}

// SetEventHandlers sets event handlers for cluster events
func (cm *ClusterManager) SetEventHandlers(
	onNodeJoin func(*ClusterNode),
	onNodeLeave func(*ClusterNode),
	onLeaderChange func(string, string),
) {
	cm.onNodeJoin = onNodeJoin
	cm.onNodeLeave = onNodeLeave
	cm.onLeaderChange = onLeaderChange
}

// Close shuts down the cluster manager
func (cm *ClusterManager) Close() error {
	cm.cancel()
	cm.wg.Wait()

	if cm.server != nil {
		return cm.server.Shutdown(context.Background())
	}
	return nil
}

// Helper methods

func (cm *ClusterManager) startHTTPServer() error {
	mux := http.NewServeMux()

	// Cluster API endpoints
	mux.HandleFunc("/cluster/nodes", cm.handleGetNodes)
	mux.HandleFunc("/cluster/heartbeat", cm.handleHeartbeat)
	mux.HandleFunc("/cluster/load", cm.handleUpdateLoad)
	mux.HandleFunc("/cluster/message", cm.handleMessage)

	cm.server = &http.Server{
		Addr:    cm.config.ListenAddress,
		Handler: mux,
	}

	go func() {
		if err := cm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

func (cm *ClusterManager) startHeartbeat() {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(cm.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cm.ctx.Done():
				return
			case <-ticker.C:
				cm.sendHeartbeat()
			}
		}
	}()
}

func (cm *ClusterManager) startNodeDiscovery() {
	if !cm.config.DiscoveryEnabled {
		return
	}

	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-cm.ctx.Done():
				return
			case <-ticker.C:
				cm.discoverNodes()
			}
		}
	}()
}

func (cm *ClusterManager) startLoadMonitoring() {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-cm.ctx.Done():
				return
			case <-ticker.C:
				cm.updateSelfLoad()
			}
		}
	}()
}

func (cm *ClusterManager) startAutoScaling() {
	if !cm.config.AutoScale {
		return
	}

	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-cm.ctx.Done():
				return
			case <-ticker.C:
				cm.checkAutoScaling()
			}
		}
	}()
}

func (cm *ClusterManager) sendHeartbeat() {
	cm.nodesMu.RLock()
	nodes := make([]*ClusterNode, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		if node.ID != cm.config.NodeID && node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	cm.nodesMu.RUnlock()

	heartbeat := map[string]interface{}{
		"node_id":   cm.config.NodeID,
		"timestamp": time.Now(),
		"status":    "active",
		"load":      cm.getSelfLoad(),
	}

	for _, node := range nodes {
		go func(n *ClusterNode) {
			cm.sendHeartbeatToNode(n, heartbeat)
		}(node)
	}
}

func (cm *ClusterManager) sendHeartbeatToNode(node *ClusterNode, heartbeat map[string]interface{}) {
	url := fmt.Sprintf("http://%s/cluster/heartbeat", node.AdvertiseAddr)

	heartbeatData, err := json.Marshal(heartbeat)
	if err != nil {
		log.Printf("Failed to marshal heartbeat: %v", err)
		return
	}

	resp, err := cm.httpClient.Post(url, "application/json",
		strings.NewReader(string(heartbeatData)))
	if err != nil {
		log.Printf("Failed to send heartbeat to %s: %v", node.ID, err)
		cm.markNodeUnhealthy(node.ID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heartbeat failed for node %s: status %d", node.ID, resp.StatusCode)
		cm.markNodeUnhealthy(node.ID)
	}
}

func (cm *ClusterManager) markNodeUnhealthy(nodeID string) {
	cm.nodesMu.Lock()
	defer cm.nodesMu.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		node.Status = "inactive"
		log.Printf("Marked node %s as inactive", nodeID)
	}
}

func (cm *ClusterManager) electNewLeader() {
	cm.leaderMu.Lock()
	defer cm.leaderMu.Unlock()

	oldLeader := cm.leader
	cm.leader = ""

	// Find the first primary node that is active
	for _, node := range cm.nodes {
		if node.Type == "primary" && node.Status == "active" {
			cm.leader = node.ID
			break
		}
	}

	// If no primary node found, elect any active node
	if cm.leader == "" {
		for _, node := range cm.nodes {
			if node.Status == "active" {
				cm.leader = node.ID
				break
			}
		}
	}

	if cm.leader != oldLeader && cm.onLeaderChange != nil {
		cm.onLeaderChange(oldLeader, cm.leader)
	}

	log.Printf("Leader changed from %s to %s", oldLeader, cm.leader)
}

func (cm *ClusterManager) getSelfLoad() NodeLoad {
	// Implementation: collect actual system metrics
	return NodeLoad{
		CPUUsage:          0.5, // Mock data
		MemoryUsage:       0.6, // Mock data
		DiskUsage:         0.3, // Mock data
		NetworkIO:         100, // Mock data
		ActiveConnections: 50,  // Mock data
		Timestamp:         time.Now(),
	}
}

func (cm *ClusterManager) updateSelfLoad() {
	load := cm.getSelfLoad()
	cm.UpdateNodeLoad(cm.config.NodeID, load)
}

func (cm *ClusterManager) checkAutoScaling() {
	if !cm.IsLeader() {
		return
	}

	// Check if we need to scale up or down
	// This is a simplified implementation
	cm.nodesMu.RLock()
	totalNodes := len(cm.nodes)
	activeNodes := 0
	totalLoad := 0.0

	for _, node := range cm.nodes {
		if node.Status == "active" {
			activeNodes++
			totalLoad += node.Load.CPUUsage
		}
	}
	cm.nodesMu.RUnlock()

	if activeNodes == 0 {
		return
	}

	averageLoad := totalLoad / float64(activeNodes)

	// Scale up if average load is too high
	if averageLoad > cm.config.ScaleUpThreshold && totalNodes < cm.config.MaxNodes {
		log.Printf("High load detected (%.2f), triggering scale up", averageLoad)
		// Implementation: trigger node creation
	}

	// Scale down if average load is too low
	if averageLoad < cm.config.ScaleDownThreshold && totalNodes > cm.config.MinNodes {
		log.Printf("Low load detected (%.2f), triggering scale down", averageLoad)
		// Implementation: trigger node termination
	}
}

// HTTP handlers

func (cm *ClusterManager) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	nodes := cm.GetNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (cm *ClusterManager) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var heartbeat map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		http.Error(w, "Invalid heartbeat data", http.StatusBadRequest)
		return
	}

	nodeID, ok := heartbeat["node_id"].(string)
	if !ok {
		http.Error(w, "Missing node_id", http.StatusBadRequest)
		return
	}

	cm.nodesMu.Lock()
	if node, exists := cm.nodes[nodeID]; exists {
		node.LastHeartbeat = time.Now()
		node.Status = "active"
	}
	cm.nodesMu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (cm *ClusterManager) handleUpdateLoad(w http.ResponseWriter, r *http.Request) {
	var loadUpdate struct {
		NodeID string   `json:"node_id"`
		Load   NodeLoad `json:"load"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loadUpdate); err != nil {
		http.Error(w, "Invalid load data", http.StatusBadRequest)
		return
	}

	cm.UpdateNodeLoad(loadUpdate.NodeID, loadUpdate.Load)
	w.WriteHeader(http.StatusOK)
}

func (cm *ClusterManager) handleMessage(w http.ResponseWriter, r *http.Request) {
	var message map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid message data", http.StatusBadRequest)
		return
	}

	// Process the message
	log.Printf("Received cluster message: %+v", message)
	w.WriteHeader(http.StatusOK)
}

func (cm *ClusterManager) sendMessageToNode(node *ClusterNode, messageData []byte) {
	url := fmt.Sprintf("http://%s/cluster/message", node.AdvertiseAddr)

	resp, err := cm.httpClient.Post(url, "application/json",
		strings.NewReader(string(messageData)))
	if err != nil {
		log.Printf("Failed to send message to %s: %v", node.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Message failed for node %s: status %d", node.ID, resp.StatusCode)
	}
}

func (cm *ClusterManager) discoverNodes() {
	// Implementation: use service discovery
	// we'll just log that discovery is running
	log.Printf("Running node discovery...")
}

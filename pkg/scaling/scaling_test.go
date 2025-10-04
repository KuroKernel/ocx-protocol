package scaling

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadBalancer(t *testing.T) {
	config := LoadBalancerConfig{
		Algorithm:           "round_robin",
		HealthCheckInterval: time.Second,
		HealthCheckTimeout:  time.Second,
		HealthCheckPath:     "/health",
		Backends: []BackendConfig{
			{ID: "backend1", URL: "http://localhost:8081", Weight: 1, Enabled: true, Health: "healthy"},
			{ID: "backend2", URL: "http://localhost:8082", Weight: 1, Enabled: true, Health: "healthy"},
		},
		CircuitBreakerEnabled: true,
		FailureThreshold:      3,
		RecoveryTimeout:       time.Second * 5,
		StickySessionEnabled:  false,
	}

	lb, err := NewLoadBalancer(config)
	if err != nil {
		t.Fatalf("Failed to create load balancer: %v", err)
	}
	defer lb.Close()

	// Test backend selection
	req := httptest.NewRequest("GET", "/test", nil)
	backend, err := lb.SelectBackend(req)
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}

	if backend == nil {
		t.Fatal("Expected backend, got nil")
	}

	// Test recording request
	lb.RecordRequest(backend.config.ID, true, time.Millisecond*100)

	// Test getting stats
	stats := lb.GetBackendStats()
	if len(stats) != 2 {
		t.Fatalf("Expected 2 backend stats, got %d", len(stats))
	}
}

func TestClusterManager(t *testing.T) {
	config := ClusterConfig{
		NodeID:            "node1",
		NodeType:          "primary",
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  time.Second,
		ClusterName:       "test-cluster",
		MinNodes:          1,
		MaxNodes:          10,
		ListenAddress:     ":8080",
		AdvertiseAddress:  "localhost:8080",
	}

	cm, err := NewClusterManager(config)
	if err != nil {
		t.Fatalf("Failed to create cluster manager: %v", err)
	}
	defer cm.Close()

	// Test getting nodes
	nodes := cm.GetNodes()
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(nodes))
	}

	// Test getting self node
	self, exists := cm.GetNode("node1")
	if !exists {
		t.Fatal("Expected to find self node")
	}

	if self.ID != "node1" {
		t.Fatalf("Expected node ID 'node1', got '%s'", self.ID)
	}

	// Test leader election
	leader := cm.GetLeader()
	if leader != "node1" {
		t.Fatalf("Expected leader 'node1', got '%s'", leader)
	}

	if !cm.IsLeader() {
		t.Fatal("Expected self to be leader")
	}
}

func TestDistributedCache(t *testing.T) {
	// Create a mock cluster manager
	clusterConfig := ClusterConfig{
		NodeID:            "node1",
		NodeType:          "primary",
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  time.Second,
		ListenAddress:     ":8082",
		AdvertiseAddress:  "localhost:8082",
	}
	cm, _ := NewClusterManager(clusterConfig)
	defer cm.Close()

	config := CacheConfig{
		DefaultTTL:         time.Minute,
		MaxSize:            1024 * 1024, // 1MB
		CleanupInterval:    time.Second,
		DistributedEnabled: false, // Disable for testing
		EvictionPolicy:     "lru",
	}

	cache, err := NewDistributedCache(config, cm)
	if err != nil {
		t.Fatalf("Failed to create distributed cache: %v", err)
	}
	defer cache.Close()

	// Test set and get
	err = cache.Set("key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache value: %v", err)
	}

	value, exists := cache.Get("key1")
	if !exists {
		t.Fatal("Expected to find cached value")
	}

	if value != "value1" {
		t.Fatalf("Expected value 'value1', got '%v'", value)
	}

	// Test delete
	deleted := cache.Delete("key1")
	if !deleted {
		t.Fatal("Expected delete to return true")
	}

	_, exists = cache.Get("key1")
	if exists {
		t.Fatal("Expected value to be deleted")
	}

	// Test stats
	stats := cache.GetStats()
	if stats.TotalEntries != 0 {
		t.Fatalf("Expected 0 total entries, got %d", stats.TotalEntries)
	}
}

func TestSessionManager(t *testing.T) {
	// Create a mock cluster manager
	clusterConfig := ClusterConfig{
		NodeID:            "node1",
		NodeType:          "primary",
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  time.Second,
		ListenAddress:     ":8083",
		AdvertiseAddress:  "localhost:8083",
	}
	cm, _ := NewClusterManager(clusterConfig)
	defer cm.Close()

	config := SessionConfig{
		DefaultTTL:         time.Minute,
		MaxSessions:        1000,
		CleanupInterval:    time.Second,
		DistributedEnabled: false, // Disable for testing
		CookieName:         "session",
		CookiePath:         "/",
		CookieHTTPOnly:     true,
		SecureCookies:      false,
	}

	sm, err := NewSessionManager(config, cm)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()

	// Test creating session
	session, err := sm.CreateSession("user1", "127.0.0.1", "test-agent", time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != "user1" {
		t.Fatalf("Expected user ID 'user1', got '%s'", session.UserID)
	}

	// Test getting session
	retrievedSession, exists := sm.GetSession(session.ID)
	if !exists {
		t.Fatal("Expected to find session")
	}

	if retrievedSession.ID != session.ID {
		t.Fatalf("Expected session ID '%s', got '%s'", session.ID, retrievedSession.ID)
	}

	// Test updating session
	err = sm.UpdateSession(session.ID, map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	updatedSession, exists := sm.GetSession(session.ID)
	if !exists {
		t.Fatal("Expected to find updated session")
	}

	if updatedSession.Data["key"] != "value" {
		t.Fatalf("Expected data value 'value', got '%v'", updatedSession.Data["key"])
	}

	// Test deleting session
	deleted := sm.DeleteSession(session.ID)
	if !deleted {
		t.Fatal("Expected delete to return true")
	}

	_, exists = sm.GetSession(session.ID)
	if exists {
		t.Fatal("Expected session to be deleted")
	}

	// Test stats
	stats := sm.GetStats()
	if stats.TotalSessions != 0 {
		t.Fatalf("Expected 0 total sessions, got %d", stats.TotalSessions)
	}
}

func TestLoadBalancingAlgorithms(t *testing.T) {
	// Create mock backends
	backends := map[string]*Backend{
		"backend1": {
			config: BackendConfig{ID: "backend1", Weight: 1},
			stats:  BackendStats{ActiveRequests: 0},
		},
		"backend2": {
			config: BackendConfig{ID: "backend2", Weight: 2},
			stats:  BackendStats{ActiveRequests: 0},
		},
	}

	// Test Round Robin
	rr := &RoundRobinAlgorithm{}
	backend, err := rr.SelectBackend(backends, "")
	if err != nil {
		t.Fatalf("Round robin failed: %v", err)
	}
	if backend == nil {
		t.Fatal("Round robin returned nil backend")
	}

	// Test Least Connections
	lc := &LeastConnectionsAlgorithm{}
	backend, err = lc.SelectBackend(backends, "")
	if err != nil {
		t.Fatalf("Least connections failed: %v", err)
	}
	if backend == nil {
		t.Fatal("Least connections returned nil backend")
	}

	// Test Weighted Round Robin
	wrr := &WeightedRoundRobinAlgorithm{}
	backend, err = wrr.SelectBackend(backends, "")
	if err != nil {
		t.Fatalf("Weighted round robin failed: %v", err)
	}
	if backend == nil {
		t.Fatal("Weighted round robin returned nil backend")
	}

	// Test IP Hash
	ih := &IPHashAlgorithm{}
	backend, err = ih.SelectBackend(backends, "192.168.1.1")
	if err != nil {
		t.Fatalf("IP hash failed: %v", err)
	}
	if backend == nil {
		t.Fatal("IP hash returned nil backend")
	}
}

func TestHealthChecker(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test health checker
	hc := NewHealthChecker(time.Second, time.Second, "/health")

	// Test healthy endpoint
	healthy := hc.CheckHealth(server.URL)
	if !healthy {
		t.Fatal("Expected health check to pass")
	}

	// Test detailed health check
	result := hc.CheckHealthDetailed(server.URL)
	if !result.Healthy {
		t.Fatal("Expected detailed health check to pass")
	}

	if result.Status != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", result.Status)
	}

	// Test unhealthy endpoint
	unhealthy := hc.CheckHealth("http://localhost:9999")
	if unhealthy {
		t.Fatal("Expected health check to fail")
	}
}

func TestShardedCache(t *testing.T) {
	// Create a mock cluster manager
	clusterConfig := ClusterConfig{
		NodeID:            "node1",
		NodeType:          "primary",
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  time.Second,
		ListenAddress:     ":8084",
		AdvertiseAddress:  "localhost:8084",
	}
	cm, _ := NewClusterManager(clusterConfig)
	defer cm.Close()

	config := CacheConfig{
		DefaultTTL:         time.Minute,
		MaxSize:            1024 * 1024,
		CleanupInterval:    time.Second,
		DistributedEnabled: false,
		EvictionPolicy:     "lru",
	}

	shardedCache, err := NewShardedCache(config, 3, cm)
	if err != nil {
		t.Fatalf("Failed to create sharded cache: %v", err)
	}
	defer shardedCache.Close()

	// Test set and get
	err = shardedCache.Set("key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set sharded cache value: %v", err)
	}

	value, exists := shardedCache.Get("key1")
	if !exists {
		t.Fatal("Expected to find cached value")
	}

	if value != "value1" {
		t.Fatalf("Expected value 'value1', got '%v'", value)
	}

	// Test stats
	stats := shardedCache.GetStats()
	if stats.TotalEntries != 1 {
		t.Fatalf("Expected 1 total entry, got %d", stats.TotalEntries)
	}
}

func TestSessionMiddleware(t *testing.T) {
	// Create a mock cluster manager
	clusterConfig := ClusterConfig{
		NodeID:            "node1",
		NodeType:          "primary",
		HeartbeatInterval: time.Second,
		HeartbeatTimeout:  time.Second,
		ListenAddress:     ":8085",
		AdvertiseAddress:  "localhost:8085",
	}
	cm, _ := NewClusterManager(clusterConfig)
	defer cm.Close()

	// Create session manager
	sessionConfig := SessionConfig{
		DefaultTTL:         time.Minute,
		MaxSessions:        1000,
		CleanupInterval:    time.Second,
		DistributedEnabled: false,
		CookieName:         "session",
		CookiePath:         "/",
		CookieHTTPOnly:     true,
		SecureCookies:      false,
	}

	sm, err := NewSessionManager(sessionConfig, cm)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()

	// Create session middleware
	middleware := NewSessionMiddleware(sm)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := r.Context().Value("session")
		if session != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("session found"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("no session"))
		}
	})

	// Test without session
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}

	// Test with session
	session, err := sm.CreateSession("user1", "127.0.0.1", "test-agent", time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	w = httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

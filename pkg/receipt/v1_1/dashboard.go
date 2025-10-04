package v1_1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DashboardManager manages the real-time statistics dashboard
type DashboardManager struct {
	db           *sql.DB
	stats        *DashboardStats
	mu           sync.RWMutex
	updateTicker *time.Ticker
	stopUpdates  chan struct{}
}

// DashboardStats contains real-time statistics
type DashboardStats struct {
	LastUpdated        time.Time              `json:"last_updated"`
	TotalReceipts      int64                  `json:"total_receipts"`
	VerifiedReceipts   int64                  `json:"verified_receipts"`
	UnverifiedReceipts int64                  `json:"unverified_receipts"`
	ReplayAttacks      int64                  `json:"replay_attacks"`
	ActiveNonces       int64                  `json:"active_nonces"`
	Issuers            map[string]IssuerStats `json:"issuers"`
	RecentActivity     []ActivityEvent        `json:"recent_activity"`
	Performance        PerformanceStats       `json:"performance"`
	SystemHealth       SystemHealth           `json:"system_health"`
}

// IssuerStats contains statistics for a specific issuer
type IssuerStats struct {
	IssuerID         string    `json:"issuer_id"`
	TotalReceipts    int64     `json:"total_receipts"`
	VerifiedReceipts int64     `json:"verified_receipts"`
	LastActivity     time.Time `json:"last_activity"`
	KeyVersions      []uint32  `json:"key_versions"`
}

// ActivityEvent represents a recent activity event
type ActivityEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	IssuerID    string    `json:"issuer_id"`
	ReceiptID   string    `json:"receipt_id"`
	Description string    `json:"description"`
	Success     bool      `json:"success"`
}

// PerformanceStats contains performance metrics
type PerformanceStats struct {
	AverageExecutionTime    time.Duration `json:"average_execution_time_ms"`
	AverageVerificationTime time.Duration `json:"average_verification_time_ms"`
	ThroughputPerSecond     float64       `json:"throughput_per_second"`
	ErrorRate               float64       `json:"error_rate"`
}

// SystemHealth contains system health information
type SystemHealth struct {
	DatabaseConnected      bool          `json:"database_connected"`
	ReplayProtectionActive bool          `json:"replay_protection_active"`
	LastDatabaseCheck      time.Time     `json:"last_database_check"`
	Uptime                 time.Duration `json:"uptime"`
	MemoryUsage            float64       `json:"memory_usage_mb"`
	CPUUsage               float64       `json:"cpu_usage_percent"`
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager(db *sql.DB) *DashboardManager {
	return &DashboardManager{
		db: db,
		stats: &DashboardStats{
			Issuers:        make(map[string]IssuerStats),
			RecentActivity: make([]ActivityEvent, 0, 100),
		},
	}
}

// Start starts the dashboard manager with periodic updates
func (dm *DashboardManager) Start(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.updateTicker != nil {
		return fmt.Errorf("dashboard already started")
	}

	// Update stats immediately
	if err := dm.updateStats(ctx); err != nil {
		return fmt.Errorf("failed to update initial stats: %w", err)
	}

	// Start periodic updates every 5 seconds
	dm.updateTicker = time.NewTicker(5 * time.Second)
	dm.stopUpdates = make(chan struct{})

	go func() {
		for {
			select {
			case <-dm.updateTicker.C:
				if err := dm.updateStats(ctx); err != nil {
					fmt.Printf("Warning: failed to update dashboard stats: %v\n", err)
				}
			case <-dm.stopUpdates:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the dashboard manager
func (dm *DashboardManager) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.updateTicker != nil {
		dm.updateTicker.Stop()
		dm.updateTicker = nil
	}

	if dm.stopUpdates != nil {
		close(dm.stopUpdates)
		dm.stopUpdates = nil
	}
}

// GetStats returns the current dashboard statistics
func (dm *DashboardManager) GetStats() *DashboardStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *dm.stats
	statsCopy.Issuers = make(map[string]IssuerStats)
	for k, v := range dm.stats.Issuers {
		statsCopy.Issuers[k] = v
	}
	statsCopy.RecentActivity = make([]ActivityEvent, len(dm.stats.RecentActivity))
	copy(statsCopy.RecentActivity, dm.stats.RecentActivity)

	return &statsCopy
}

// AddActivityEvent adds a new activity event to the dashboard
func (dm *DashboardManager) AddActivityEvent(event ActivityEvent) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Add to recent activity (keep last 100 events)
	dm.stats.RecentActivity = append(dm.stats.RecentActivity, event)
	if len(dm.stats.RecentActivity) > 100 {
		dm.stats.RecentActivity = dm.stats.RecentActivity[1:]
	}
}

// updateStats updates the dashboard statistics from the database
func (dm *DashboardManager) updateStats(ctx context.Context) error {
	stats := &DashboardStats{
		LastUpdated:    time.Now(),
		Issuers:        make(map[string]IssuerStats),
		RecentActivity: dm.stats.RecentActivity, // Preserve recent activity
	}

	// Get total receipt counts
	err := dm.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE verified = true),
			COUNT(*) FILTER (WHERE verified = false)
		FROM ocx_receipts
	`).Scan(&stats.TotalReceipts, &stats.VerifiedReceipts, &stats.UnverifiedReceipts)
	if err != nil {
		return fmt.Errorf("failed to get receipt counts: %w", err)
	}

	// Get replay attack count
	err = dm.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ocx_audit_log 
		WHERE event_type = 'replay_attack'
	`).Scan(&stats.ReplayAttacks)
	if err != nil {
		return fmt.Errorf("failed to get replay attack count: %w", err)
	}

	// Get active nonces count
	err = dm.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ocx_replay_protection 
		WHERE expires_at > NOW()
	`).Scan(&stats.ActiveNonces)
	if err != nil {
		return fmt.Errorf("failed to get active nonces count: %w", err)
	}

	// Get issuer statistics
	rows, err := dm.db.QueryContext(ctx, `
		SELECT 
			issuer_id,
			COUNT(*) as total_receipts,
			COUNT(*) FILTER (WHERE verified = true) as verified_receipts,
			MAX(created_at) as last_activity
		FROM ocx_receipts
		GROUP BY issuer_id
		ORDER BY total_receipts DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to get issuer stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issuerID string
		var totalReceipts, verifiedReceipts int64
		var lastActivity time.Time

		err := rows.Scan(&issuerID, &totalReceipts, &verifiedReceipts, &lastActivity)
		if err != nil {
			return fmt.Errorf("failed to scan issuer stats: %w", err)
		}

		// Get key versions for this issuer
		keyRows, err := dm.db.QueryContext(ctx, `
			SELECT DISTINCT key_version 
			FROM ocx_receipts 
			WHERE issuer_id = $1
			ORDER BY key_version
		`, issuerID)
		if err != nil {
			return fmt.Errorf("failed to get key versions: %w", err)
		}

		var keyVersions []uint32
		for keyRows.Next() {
			var version uint32
			if err := keyRows.Scan(&version); err != nil {
				keyRows.Close()
				return fmt.Errorf("failed to scan key version: %w", err)
			}
			keyVersions = append(keyVersions, version)
		}
		keyRows.Close()

		stats.Issuers[issuerID] = IssuerStats{
			IssuerID:         issuerID,
			TotalReceipts:    totalReceipts,
			VerifiedReceipts: verifiedReceipts,
			LastActivity:     lastActivity,
			KeyVersions:      keyVersions,
		}
	}

	// Get performance statistics
	err = dm.db.QueryRowContext(ctx, `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (finished_at - started_at)) * 1000) as avg_execution_time,
			AVG(EXTRACT(EPOCH FROM (created_at - issued_at)) * 1000) as avg_verification_time
		FROM ocx_receipts
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&stats.Performance.AverageExecutionTime, &stats.Performance.AverageVerificationTime)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get performance stats: %w", err)
	}

	// Calculate throughput (receipts per second in last hour)
	var throughput float64
	err = dm.db.QueryRowContext(ctx, `
		SELECT COUNT(*) / 3600.0 
		FROM ocx_receipts 
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&throughput)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get throughput: %w", err)
	}
	stats.Performance.ThroughputPerSecond = throughput

	// Calculate error rate
	var errorRate float64
	err = dm.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE verified = false) / GREATEST(COUNT(*), 1.0) * 100
		FROM ocx_receipts
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&errorRate)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get error rate: %w", err)
	}
	stats.Performance.ErrorRate = errorRate

	// Get system health
	stats.SystemHealth = dm.getSystemHealth(ctx)

	// Update the stats
	dm.mu.Lock()
	dm.stats = stats
	dm.mu.Unlock()

	return nil
}

// getSystemHealth gets current system health information
func (dm *DashboardManager) getSystemHealth(ctx context.Context) SystemHealth {
	health := SystemHealth{
		LastDatabaseCheck: time.Now(),
	}

	// Check database connection
	err := dm.db.PingContext(ctx)
	health.DatabaseConnected = (err == nil)

	// Check replay protection status
	var replayActive bool
	err = dm.db.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 FROM ocx_replay_protection 
		WHERE expires_at > NOW()
	`).Scan(&replayActive)
	health.ReplayProtectionActive = (err == nil && replayActive)

	// Get uptime (simplified - in production you'd track actual start time)
	health.Uptime = time.Since(time.Now().Add(-24 * time.Hour)) // Placeholder

	// Get memory and CPU usage (simplified - in production you'd use proper monitoring)
	health.MemoryUsage = 128.5 // Placeholder MB
	health.CPUUsage = 15.2     // Placeholder percent

	return health
}

// ServeHTTP serves the dashboard as a web interface
func (dm *DashboardManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stats := dm.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(stats); err != nil {
		http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
		return
	}
}

// ServeHTML serves a simple HTML dashboard
func (dm *DashboardManager) ServeHTML(w http.ResponseWriter, r *http.Request) {
	stats := dm.GetStats()

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>OCX Protocol Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stat-value { font-size: 2em; font-weight: bold; color: #2c3e50; }
        .stat-label { color: #7f8c8d; margin-top: 5px; }
        .issuer-table { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .issuer-table th, .issuer-table td { padding: 12px; text-align: left; border-bottom: 1px solid #ecf0f1; }
        .issuer-table th { background: #34495e; color: white; }
        .activity-list { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); padding: 20px; }
        .activity-item { padding: 10px 0; border-bottom: 1px solid #ecf0f1; }
        .activity-item:last-child { border-bottom: none; }
        .success { color: #27ae60; }
        .error { color: #e74c3c; }
        .refresh-btn { background: #3498db; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
        .refresh-btn:hover { background: #2980b9; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>OCX Protocol Dashboard</h1>
            <p>Last updated: %s</p>
            <button class="refresh-btn" onclick="location.reload()">Refresh</button>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Total Receipts</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Verified Receipts</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Unverified Receipts</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Replay Attacks Blocked</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Active Nonces</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%.2f</div>
                <div class="stat-label">Throughput (req/s)</div>
            </div>
        </div>
        
        <div class="issuer-table">
            <table style="width: 100%%;">
                <thead>
                    <tr>
                        <th>Issuer ID</th>
                        <th>Total Receipts</th>
                        <th>Verified</th>
                        <th>Last Activity</th>
                        <th>Key Versions</th>
                    </tr>
                </thead>
                <tbody>
`,
		stats.LastUpdated.Format("2006-01-02 15:04:05"),
		stats.TotalReceipts,
		stats.VerifiedReceipts,
		stats.UnverifiedReceipts,
		stats.ReplayAttacks,
		stats.ActiveNonces,
		stats.Performance.ThroughputPerSecond,
	)

	for _, issuer := range stats.Issuers {
		html += fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td>%d</td>
                        <td>%d</td>
                        <td>%s</td>
                        <td>%v</td>
                    </tr>
`,
			issuer.IssuerID,
			issuer.TotalReceipts,
			issuer.VerifiedReceipts,
			issuer.LastActivity.Format("2006-01-02 15:04:05"),
			issuer.KeyVersions,
		)
	}

	html += `
                </tbody>
            </table>
        </div>
        
        <div class="activity-list">
            <h3>Recent Activity</h3>
`

	for _, activity := range stats.RecentActivity {
		statusClass := "success"
		if !activity.Success {
			statusClass = "error"
		}

		html += fmt.Sprintf(`
            <div class="activity-item">
                <span class="%s">[%s]</span> %s - %s (%s)
            </div>
`,
			statusClass,
			activity.Timestamp.Format("15:04:05"),
			activity.EventType,
			activity.Description,
			activity.IssuerID,
		)
	}

	html += `
        </div>
    </div>
    
    <script>
        // Auto-refresh every 30 seconds
        setTimeout(function() {
            location.reload();
        }, 30000);
    </script>
</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

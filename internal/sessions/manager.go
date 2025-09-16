package sessions

import (
	"database/sql"
	"fmt"
	"time"
)

// SessionManager handles compute session lifecycle
type SessionManager struct {
	db *sql.DB
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{db: db}
}

// Session represents a compute session
type Session struct {
	SessionID           string    `json:"session_id"`
	OrderID             string    `json:"order_id"`
	UnitID              string    `json:"unit_id"`
	ProviderID          string    `json:"provider_id"`
	AgreedPrice         float64   `json:"agreed_price_per_hour"`
	EstimatedEndTime    time.Time `json:"estimated_end_time"`
	Status              string    `json:"session_status"`
	ConnectionDetails   string    `json:"connection_details_encrypted"`
	SessionToken        string    `json:"session_token"`
	AllocatedGPUDevices []int     `json:"allocated_gpu_devices"`
	AllocatedCPUCores   []int     `json:"allocated_cpu_cores"`
	AllocatedRAM        int       `json:"allocated_ram_gb"`
	AllocatedStorage    string    `json:"allocated_storage_path"`
	ProvisioningStarted time.Time `json:"provisioning_started_at"`
	SessionStarted      *time.Time `json:"session_started_at"`
	SessionEnded        *time.Time `json:"session_ended_at"`
	BaseCost            float64   `json:"base_cost_usdc"`
	UsagePremiums       float64   `json:"usage_premiums_usdc"`
	TotalCost           float64   `json:"total_cost_usdc"`
}

// CreateSession creates a new compute session from a match
func (m *SessionManager) CreateSession(matchID string) (*Session, error) {
	// Get match details
	match, err := m.getMatch(matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}
	
	// Create session
	session := &Session{
		SessionID:           fmt.Sprintf("session_%d", time.Now().UnixNano()),
		OrderID:             match.OrderID,
		UnitID:              match.UnitID,
		ProviderID:          match.ProviderID,
		AgreedPrice:         match.Price,
		EstimatedEndTime:    time.Now().Add(4 * time.Hour), // Default 4 hours
		Status:              "provisioning",
		SessionToken:        fmt.Sprintf("token_%d", time.Now().UnixNano()),
		AllocatedGPUDevices: []int{0}, // Default GPU 0
		AllocatedCPUCores:   []int{0, 1, 2, 3}, // Default 4 cores
		AllocatedRAM:        16, // Default 16GB
		AllocatedStorage:    "/tmp/ocx_session",
		ProvisioningStarted: time.Now(),
		BaseCost:            0,
		UsagePremiums:       0,
		TotalCost:           0,
	}
	
	// Save session to database
	if err := m.saveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	
	// Update match status
	if err := m.updateMatchStatus(matchID, "session_created"); err != nil {
		return nil, fmt.Errorf("failed to update match status: %w", err)
	}
	
	return session, nil
}

// StartSession starts a provisioned session
func (m *SessionManager) StartSession(sessionID string) error {
	now := time.Now()
	
	// Update session status
	query := `
		UPDATE compute_sessions 
		SET session_status = 'active', session_started_at = $1
		WHERE session_id = $2
	`
	
	_, err := m.db.Exec(query, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	
	// Update compute unit status
	unitQuery := `
		UPDATE compute_units 
		SET current_availability = 'in_use'
		WHERE unit_id = (SELECT unit_id FROM compute_sessions WHERE session_id = $1)
	`
	
	_, err = m.db.Exec(unitQuery, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update unit status: %w", err)
	}
	
	return nil
}

// EndSession ends a session
func (m *SessionManager) EndSession(sessionID string, reason string) error {
	now := time.Now()
	
	// Calculate costs
	session, err := m.getSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	
	// Calculate session duration and costs
	duration := now.Sub(*session.SessionStarted)
	hours := duration.Hours()
	totalCost := session.AgreedPrice * hours
	
	// Update session
	query := `
		UPDATE compute_sessions 
		SET session_status = 'completed', session_ended_at = $1, total_cost_usdc = $2
		WHERE session_id = $3
	`
	
	_, err = m.db.Exec(query, now, totalCost, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	
	// Update compute unit availability
	unitQuery := `
		UPDATE compute_units 
		SET current_availability = 'available'
		WHERE unit_id = (SELECT unit_id FROM compute_sessions WHERE session_id = $1)
	`
	
	_, err = m.db.Exec(unitQuery, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update unit availability: %w", err)
	}
	
	return nil
}

// GetSession returns session details
func (m *SessionManager) GetSession(sessionID string) (*Session, error) {
	return m.getSession(sessionID)
}

// GetActiveSessions returns all active sessions
func (m *SessionManager) GetActiveSessions() ([]Session, error) {
	query := `
		SELECT session_id, order_id, unit_id, provider_id, agreed_price_per_hour_usdc,
		       estimated_end_time, session_status, session_token, allocated_gpu_devices,
		       allocated_cpu_cores, allocated_ram_gb, allocated_storage_path,
		       provisioning_started_at, session_started_at, session_ended_at,
		       base_cost_usdc, usage_premiums_usdc, total_cost_usdc
		FROM compute_sessions 
		WHERE session_status IN ('provisioning', 'active')
		ORDER BY provisioning_started_at DESC
	`
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var sessions []Session
	for rows.Next() {
		var session Session
		var gpuDevices, cpuCores []int
		var sessionStarted, sessionEnded *time.Time
		
		err := rows.Scan(
			&session.SessionID, &session.OrderID, &session.UnitID, &session.ProviderID,
			&session.AgreedPrice, &session.EstimatedEndTime, &session.Status, &session.SessionToken,
			&gpuDevices, &cpuCores, &session.AllocatedRAM, &session.AllocatedStorage,
			&session.ProvisioningStarted, &sessionStarted, &sessionEnded,
			&session.BaseCost, &session.UsagePremiums, &session.TotalCost,
		)
		if err != nil {
			continue
		}
		
		session.AllocatedGPUDevices = gpuDevices
		session.AllocatedCPUCores = cpuCores
		session.SessionStarted = sessionStarted
		session.SessionEnded = sessionEnded
		
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

// RecordMetrics records session metrics
func (m *SessionManager) RecordMetrics(sessionID string, metrics map[string]interface{}) error {
	query := `
		INSERT INTO session_metrics (
			session_id, gpu_utilization_percent, gpu_memory_used_mb, gpu_temperature_celsius,
			cpu_utilization_percent, ram_used_gb, training_steps_per_second
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := m.db.Exec(query,
		sessionID,
		metrics["gpu_utilization_percent"],
		metrics["gpu_memory_used_mb"],
		metrics["gpu_temperature_celsius"],
		metrics["cpu_utilization_percent"],
		metrics["ram_used_gb"],
		metrics["training_steps_per_second"],
	)
	
	return err
}

// Helper methods
func (m *SessionManager) getMatch(matchID string) (*Match, error) {
	query := `
		SELECT match_id, order_id, unit_id, provider_id, price_per_hour, matched_at, status
		FROM order_matches 
		WHERE match_id = $1
	`
	
	var match Match
	err := m.db.QueryRow(query, matchID).Scan(
		&match.MatchID, &match.OrderID, &match.UnitID,
		&match.ProviderID, &match.Price, &match.MatchedAt, &match.Status,
	)
	
	return &match, err
}

func (m *SessionManager) getSession(sessionID string) (*Session, error) {
	query := `
		SELECT session_id, order_id, unit_id, provider_id, agreed_price_per_hour_usdc,
		       estimated_end_time, session_status, session_token, allocated_gpu_devices,
		       allocated_cpu_cores, allocated_ram_gb, allocated_storage_path,
		       provisioning_started_at, session_started_at, session_ended_at,
		       base_cost_usdc, usage_premiums_usdc, total_cost_usdc
		FROM compute_sessions 
		WHERE session_id = $1
	`
	
	var session Session
	var gpuDevices, cpuCores []int
	var sessionStarted, sessionEnded *time.Time
	
	err := m.db.QueryRow(query, sessionID).Scan(
		&session.SessionID, &session.OrderID, &session.UnitID, &session.ProviderID,
		&session.AgreedPrice, &session.EstimatedEndTime, &session.Status, &session.SessionToken,
		&gpuDevices, &cpuCores, &session.AllocatedRAM, &session.AllocatedStorage,
		&session.ProvisioningStarted, &sessionStarted, &sessionEnded,
		&session.BaseCost, &session.UsagePremiums, &session.TotalCost,
	)
	
	if err != nil {
		return nil, err
	}
	
	session.AllocatedGPUDevices = gpuDevices
	session.AllocatedCPUCores = cpuCores
	session.SessionStarted = sessionStarted
	session.SessionEnded = sessionEnded
	
	return &session, nil
}

func (m *SessionManager) saveSession(session *Session) error {
	query := `
		INSERT INTO compute_sessions (
			session_id, order_id, unit_id, provider_id, agreed_price_per_hour_usdc,
			estimated_end_time, session_status, session_token, allocated_gpu_devices,
			allocated_cpu_cores, allocated_ram_gb, allocated_storage_path,
			provisioning_started_at, base_cost_usdc, usage_premiums_usdc, total_cost_usdc
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	
	_, err := m.db.Exec(query,
		session.SessionID, session.OrderID, session.UnitID, session.ProviderID,
		session.AgreedPrice, session.EstimatedEndTime, session.Status, session.SessionToken,
		session.AllocatedGPUDevices, session.AllocatedCPUCores, session.AllocatedRAM,
		session.AllocatedStorage, session.ProvisioningStarted, session.BaseCost,
		session.UsagePremiums, session.TotalCost,
	)
	
	return err
}

func (m *SessionManager) updateMatchStatus(matchID, status string) error {
	query := `UPDATE order_matches SET status = $1 WHERE match_id = $2`
	_, err := m.db.Exec(query, status, matchID)
	return err
}

// Match represents a match from the database
type Match struct {
	MatchID    string    `json:"match_id"`
	OrderID    string    `json:"order_id"`
	UnitID     string    `json:"unit_id"`
	ProviderID string    `json:"provider_id"`
	Price      float64   `json:"price_per_hour"`
	MatchedAt  time.Time `json:"matched_at"`
	Status     string    `json:"status"`
}

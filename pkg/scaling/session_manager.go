package scaling

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SessionConfig defines configuration for session management
type SessionConfig struct {
	// Session settings
	DefaultTTL      time.Duration `json:"default_ttl"`
	MaxSessions     int64         `json:"max_sessions"`
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// Distributed session settings
	DistributedEnabled bool     `json:"distributed_enabled"`
	ClusterNodes       []string `json:"cluster_nodes"`
	ReplicationFactor  int      `json:"replication_factor"`

	// Session storage
	StorageType   string                 `json:"storage_type"` // "memory", "redis", "database"
	StorageConfig map[string]interface{} `json:"storage_config"`

	// Security
	EncryptionEnabled bool   `json:"encryption_enabled"`
	EncryptionKey     string `json:"encryption_key"`
	SecureCookies     bool   `json:"secure_cookies"`

	// Cookie settings
	CookieName     string `json:"cookie_name"`
	CookieDomain   string `json:"cookie_domain"`
	CookiePath     string `json:"cookie_path"`
	CookieHTTPOnly bool   `json:"cookie_http_only"`
	CookieSameSite string `json:"cookie_same_site"`
}

// Session represents a user session
type Session struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  time.Time              `json:"created_at"`
	LastAccess time.Time              `json:"last_access"`
	ExpiresAt  time.Time              `json:"expires_at"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	IsActive   bool                   `json:"is_active"`
	Version    int64                  `json:"version"`
}

// SessionStats represents session statistics
type SessionStats struct {
	TotalSessions          int64         `json:"total_sessions"`
	ActiveSessions         int64         `json:"active_sessions"`
	ExpiredSessions        int64         `json:"expired_sessions"`
	AverageSessionDuration time.Duration `json:"average_session_duration"`
	LastCleanup            time.Time     `json:"last_cleanup"`
}

// SessionManager manages distributed sessions across cluster nodes
type SessionManager struct {
	config     SessionConfig
	sessions   map[string]*Session
	sessionsMu sync.RWMutex
	stats      SessionStats
	statsMu    sync.RWMutex

	// Cluster communication
	clusterManager *ClusterManager

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Event handlers
	onSessionCreate func(*Session)
	onSessionUpdate func(*Session)
	onSessionDelete func(string)
	onSessionExpire func(*Session)
}

// SessionOperation represents a session operation
type SessionOperation struct {
	Type      string      `json:"type"` // "create", "update", "delete", "expire"
	SessionID string      `json:"session_id"`
	Session   *Session    `json:"session,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	NodeID    string      `json:"node_id"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(config SessionConfig, clusterManager *ClusterManager) (*SessionManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SessionManager{
		config:         config,
		sessions:       make(map[string]*Session),
		clusterManager: clusterManager,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start background tasks
	sm.startCleanup()
	sm.startReplication()

	return sm, nil
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(userID, ipAddress, userAgent string, ttl time.Duration) (*Session, error) {
	if ttl == 0 {
		ttl = sm.config.DefaultTTL
	}

	sessionID, err := sm.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	now := time.Now()
	session := &Session{
		ID:         sessionID,
		UserID:     userID,
		Data:       make(map[string]interface{}),
		CreatedAt:  now,
		LastAccess: now,
		ExpiresAt:  now.Add(ttl),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		IsActive:   true,
		Version:    1,
	}

	sm.sessionsMu.Lock()
	sm.sessions[sessionID] = session
	sm.sessionsMu.Unlock()

	// Update statistics
	sm.updateStats()

	// Replicate to other nodes if distributed
	if sm.config.DistributedEnabled {
		sm.replicateOperation(SessionOperation{
			Type:      "create",
			SessionID: sessionID,
			Session:   session,
			Timestamp: now,
			NodeID:    sm.clusterManager.config.NodeID,
		})
	}

	// Trigger event handler
	if sm.onSessionCreate != nil {
		sm.onSessionCreate(session)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.sessionsMu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.sessionsMu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		sm.DeleteSession(sessionID)
		return nil, false
	}

	// Update last access time
	sm.updateLastAccess(session)

	return session, true
}

// UpdateSession updates session data
func (sm *SessionManager) UpdateSession(sessionID string, data map[string]interface{}) error {
	sm.sessionsMu.Lock()
	session, exists := sm.sessions[sessionID]
	if exists {
		// Update session data
		for key, value := range data {
			session.Data[key] = value
		}
		session.LastAccess = time.Now()
		session.Version++
	}
	sm.sessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Replicate to other nodes if distributed
	if sm.config.DistributedEnabled {
		sm.replicateOperation(SessionOperation{
			Type:      "update",
			SessionID: sessionID,
			Session:   session,
			Data:      data,
			Timestamp: time.Now(),
			NodeID:    sm.clusterManager.config.NodeID,
		})
	}

	// Trigger event handler
	if sm.onSessionUpdate != nil {
		sm.onSessionUpdate(session)
	}

	return nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) bool {
	sm.sessionsMu.Lock()
	_, exists := sm.sessions[sessionID]
	if exists {
		delete(sm.sessions, sessionID)
	}
	sm.sessionsMu.Unlock()

	if exists {
		// Update statistics
		sm.updateStats()

		// Replicate to other nodes if distributed
		if sm.config.DistributedEnabled {
			sm.replicateOperation(SessionOperation{
				Type:      "delete",
				SessionID: sessionID,
				Timestamp: time.Now(),
				NodeID:    sm.clusterManager.config.NodeID,
			})
		}

		// Trigger event handler
		if sm.onSessionDelete != nil {
			sm.onSessionDelete(sessionID)
		}

		return true
	}

	return false
}

// ExtendSession extends the session expiration time
func (sm *SessionManager) ExtendSession(sessionID string, ttl time.Duration) error {
	sm.sessionsMu.Lock()
	session, exists := sm.sessions[sessionID]
	if exists {
		session.ExpiresAt = time.Now().Add(ttl)
		session.LastAccess = time.Now()
		session.Version++
	}
	sm.sessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Replicate to other nodes if distributed
	if sm.config.DistributedEnabled {
		sm.replicateOperation(SessionOperation{
			Type:      "update",
			SessionID: sessionID,
			Session:   session,
			Timestamp: time.Now(),
			NodeID:    sm.clusterManager.config.NodeID,
		})
	}

	return nil
}

// GetUserSessions returns all sessions for a user
func (sm *SessionManager) GetUserSessions(userID string) []*Session {
	sm.sessionsMu.RLock()
	defer sm.sessionsMu.RUnlock()

	var userSessions []*Session
	for _, session := range sm.sessions {
		if session.UserID == userID && session.IsActive && time.Now().Before(session.ExpiresAt) {
			userSessions = append(userSessions, session)
		}
	}

	return userSessions
}

// GetStats returns session statistics
func (sm *SessionManager) GetStats() SessionStats {
	sm.statsMu.RLock()
	defer sm.statsMu.RUnlock()
	return sm.stats
}

// GetActiveSessions returns all active sessions
func (sm *SessionManager) GetActiveSessions() []*Session {
	sm.sessionsMu.RLock()
	defer sm.sessionsMu.RUnlock()

	var activeSessions []*Session
	now := time.Now()

	for _, session := range sm.sessions {
		if session.IsActive && now.Before(session.ExpiresAt) {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// SetEventHandlers sets event handlers for session events
func (sm *SessionManager) SetEventHandlers(
	onSessionCreate func(*Session),
	onSessionUpdate func(*Session),
	onSessionDelete func(string),
	onSessionExpire func(*Session),
) {
	sm.onSessionCreate = onSessionCreate
	sm.onSessionUpdate = onSessionUpdate
	sm.onSessionDelete = onSessionDelete
	sm.onSessionExpire = onSessionExpire
}

// Close shuts down the session manager
func (sm *SessionManager) Close() {
	sm.cancel()
	sm.wg.Wait()
}

// Helper methods

func (sm *SessionManager) generateSessionID() (string, error) {
	// Generate a cryptographically secure random session ID
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (sm *SessionManager) updateLastAccess(session *Session) {
	sm.sessionsMu.Lock()
	session.LastAccess = time.Now()
	sm.sessionsMu.Unlock()
}

func (sm *SessionManager) updateStats() {
	sm.sessionsMu.RLock()
	totalSessions := int64(len(sm.sessions))
	activeSessions := int64(0)
	expiredSessions := int64(0)
	totalDuration := time.Duration(0)

	now := time.Now()
	for _, session := range sm.sessions {
		if session.IsActive && now.Before(session.ExpiresAt) {
			activeSessions++
			totalDuration += now.Sub(session.CreatedAt)
		} else {
			expiredSessions++
		}
	}
	sm.sessionsMu.RUnlock()

	sm.statsMu.Lock()
	sm.stats.TotalSessions = totalSessions
	sm.stats.ActiveSessions = activeSessions
	sm.stats.ExpiredSessions = expiredSessions
	if activeSessions > 0 {
		sm.stats.AverageSessionDuration = totalDuration / time.Duration(activeSessions)
	}
	sm.statsMu.Unlock()
}

func (sm *SessionManager) startCleanup() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		ticker := time.NewTicker(sm.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-ticker.C:
				sm.cleanup()
			}
		}
	}()
}

func (sm *SessionManager) cleanup() {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	now := time.Now()
	expired := 0

	// Remove expired sessions
	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, sessionID)
			expired++

			// Trigger event handler
			if sm.onSessionExpire != nil {
				sm.onSessionExpire(session)
			}
		}
	}

	// Check if we need to evict sessions based on max sessions
	if sm.config.MaxSessions > 0 && int64(len(sm.sessions)) > sm.config.MaxSessions {
		evicted := sm.evictOldestSessions(int64(len(sm.sessions)) - sm.config.MaxSessions)
		expired += evicted
	}

	// Update statistics
	sm.statsMu.Lock()
	sm.stats.LastCleanup = now
	sm.statsMu.Unlock()

	if expired > 0 {
		log.Printf("Session cleanup: removed %d expired sessions", expired)
	}
}

func (sm *SessionManager) evictOldestSessions(count int64) int {
	// Sort sessions by last access time
	type sessionAccess struct {
		sessionID  string
		lastAccess time.Time
	}

	var sessions []sessionAccess
	for sessionID, session := range sm.sessions {
		sessions = append(sessions, sessionAccess{
			sessionID:  sessionID,
			lastAccess: session.LastAccess,
		})
	}

	// Sort by last access time (oldest first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[i].lastAccess.After(sessions[j].lastAccess) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	// Remove oldest sessions
	evicted := 0
	for i := int64(0); i < count && i < int64(len(sessions)); i++ {
		sessionID := sessions[i].sessionID
		if session, exists := sm.sessions[sessionID]; exists {
			delete(sm.sessions, sessionID)
			evicted++

			// Trigger event handler
			if sm.onSessionExpire != nil {
				sm.onSessionExpire(session)
			}
		}
	}

	return evicted
}

func (sm *SessionManager) startReplication() {
	if !sm.config.DistributedEnabled {
		return
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		// Implementation: handle replication
		// we'll just log that replication is running
		log.Printf("Session replication started")
	}()
}

func (sm *SessionManager) replicateOperation(operation SessionOperation) {
	// Implementation: send the operation to other nodes
	// we'll just log the operation
	log.Printf("Replicating session operation: %+v", operation)
}

// SessionMiddleware provides HTTP middleware for session management
type SessionMiddleware struct {
	sessionManager *SessionManager
}

// NewSessionMiddleware creates a new session middleware
func NewSessionMiddleware(sessionManager *SessionManager) *SessionMiddleware {
	return &SessionMiddleware{
		sessionManager: sessionManager,
	}
}

// Middleware returns the HTTP middleware function
func (sm *SessionMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session ID from cookie
		sessionID := sm.getSessionIDFromRequest(r)

		if sessionID != "" {
			// Get session
			if session, exists := sm.sessionManager.GetSession(sessionID); exists {
				// Add session to request context
				ctx := context.WithValue(r.Context(), "session", session)
				r = r.WithContext(ctx)
			}
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

func (sm *SessionMiddleware) getSessionIDFromRequest(r *http.Request) string {
	// Try to get session ID from cookie
	if cookie, err := r.Cookie(sm.sessionManager.config.CookieName); err == nil {
		return cookie.Value
	}

	// Try to get session ID from header
	return r.Header.Get("X-Session-ID")
}

// SetSessionCookie sets the session cookie in the response
func (sm *SessionMiddleware) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name:     sm.sessionManager.config.CookieName,
		Value:    sessionID,
		Path:     sm.sessionManager.config.CookiePath,
		Domain:   sm.sessionManager.config.CookieDomain,
		HttpOnly: sm.sessionManager.config.CookieHTTPOnly,
		Secure:   sm.sessionManager.config.SecureCookies,
		SameSite: sm.getSameSiteMode(),
		Expires:  time.Now().Add(sm.sessionManager.config.DefaultTTL),
	}

	http.SetCookie(w, cookie)
}

func (sm *SessionMiddleware) getSameSiteMode() http.SameSite {
	switch sm.sessionManager.config.CookieSameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

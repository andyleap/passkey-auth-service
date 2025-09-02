package storage

import (
	"context"
	"sync"
	"time"

	"github.com/andyleap/passkey/internal/models"
)

type MemoryStorage struct {
	webauthnSessions map[string]*models.WebAuthnSession
	sessions         map[string]*models.Session
	mu               sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	storage := &MemoryStorage{
		webauthnSessions: make(map[string]*models.WebAuthnSession),
		sessions:         make(map[string]*models.Session),
	}

	// Start background cleanup routine
	go storage.cleanupRoutine()

	return storage
}

func (m *MemoryStorage) SaveWebAuthnSession(ctx context.Context, username string, session *models.WebAuthnSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.webauthnSessions[username] = session
	return nil
}

func (m *MemoryStorage) GetWebAuthnSession(ctx context.Context, username string) (*models.WebAuthnSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, exists := m.webauthnSessions[username]
	if !exists {
		return nil, nil
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session (note: we need to upgrade to write lock)
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.webauthnSessions, username)
		m.mu.Unlock()
		m.mu.RLock()
		return nil, nil
	}

	return session, nil
}

func (m *MemoryStorage) DeleteWebAuthnSession(ctx context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.webauthnSessions, username)
	return nil
}

func (m *MemoryStorage) SaveSession(ctx context.Context, session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sessions[session.ID] = session
	return nil
}

func (m *MemoryStorage) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, nil
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session (note: we need to upgrade to write lock)
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
		m.mu.RLock()
		return nil, nil
	}

	return session, nil
}

func (m *MemoryStorage) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.sessions, sessionID)
	return nil
}

// cleanupRoutine runs every 5 minutes to clean up expired sessions
func (m *MemoryStorage) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

func (m *MemoryStorage) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Clean up expired WebAuthn sessions
	for username, session := range m.webauthnSessions {
		if now.After(session.ExpiresAt) {
			delete(m.webauthnSessions, username)
		}
	}

	// Clean up expired user sessions
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, sessionID)
		}
	}
}
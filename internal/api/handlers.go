package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/andyleap/passkey/internal/auth"
	"github.com/andyleap/passkey/internal/storage"
)

type Server struct {
	webauthnService *auth.WebAuthnService
	sessionStorage  storage.SessionStorage
}

func NewServer(webauthnService *auth.WebAuthnService, sessionStorage storage.SessionStorage) *Server {
	return &Server{
		webauthnService: webauthnService,
		sessionStorage:  sessionStorage,
	}
}

func (s *Server) ValidateSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	session, err := s.sessionStorage.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "failed to get session", http.StatusInternalServerError)
		return
	}

	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":    true,
		"username": session.Username,
		"userId":   session.UserID,
		"expires":  session.ExpiresAt,
	})
}

func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("sessionId")
	}
	
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	if err := s.sessionStorage.DeleteSession(r.Context(), sessionID); err != nil {
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// getUserFromRequest extracts and validates user from session
func (s *Server) getUserFromRequest(r *http.Request) (string, error) {
	sessionID := ""
	
	// Try cookie first
	if cookie, err := r.Cookie("session_id"); err == nil {
		sessionID = cookie.Value
	}
	
	// Try Authorization header
	if sessionID == "" {
		if auth := r.Header.Get("Authorization"); auth != "" {
			if len(auth) > 7 && auth[:7] == "Bearer " {
				sessionID = auth[7:]
			}
		}
	}
	
	if sessionID == "" {
		return "", fmt.Errorf("no session found")
	}
	
	session, err := s.sessionStorage.GetSession(r.Context(), sessionID)
	if err != nil || session == nil {
		return "", fmt.Errorf("invalid session")
	}
	
	if session.ExpiresAt.Before(time.Now()) {
		return "", fmt.Errorf("session expired")
	}
	
	return session.Username, nil
}

// UserCredentialsHandler returns user's credentials
func (s *Server) UserCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	username, err := s.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	
	user, err := s.webauthnService.GetUser(r.Context(), username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Convert credentials to a safe format for JSON
	credentials := make([]map[string]interface{}, len(user.Credentials))
	for i, cred := range user.Credentials {
		credentials[i] = map[string]interface{}{
			"id":        base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(cred.ID),
			"createdAt": user.CreatedAt, // Approximate - we don't store individual cred dates
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":    user.Name,
		"credentials": credentials,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
	})
}

// UserSessionsHandler returns user's active sessions
func (s *Server) UserSessionsHandler(w http.ResponseWriter, r *http.Request) {
	username, err := s.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	
	sessions, err := s.sessionStorage.GetUserSessions(r.Context(), username)
	if err != nil {
		slog.Error("Failed to get user sessions", "error", err)
		http.Error(w, "Failed to get sessions", http.StatusInternalServerError)
		return
	}
	
	// Convert sessions to safe format
	safeSessions := make([]map[string]interface{}, len(sessions))
	for i, session := range sessions {
		safeSessions[i] = map[string]interface{}{
			"id":        session.ID,
			"createdAt": session.CreatedAt,
			"expiresAt": session.ExpiresAt,
			"current":   session.ID == r.Header.Get("X-Session-ID"), // Mark current session
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username": username,
		"sessions": safeSessions,
	})
}

// DeleteCredentialHandler deletes a specific credential
func (s *Server) DeleteCredentialHandler(w http.ResponseWriter, r *http.Request) {
	username, err := s.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	
	credentialID := r.PathValue("credentialId")
	if credentialID == "" {
		http.Error(w, "Credential ID required", http.StatusBadRequest)
		return
	}
	
	err = s.webauthnService.DeleteCredential(r.Context(), username, credentialID)
	if err != nil {
		slog.Error("Failed to delete credential", "error", err, "username", username, "credentialId", credentialID)
		http.Error(w, "Failed to delete credential", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// DeleteSessionHandler deletes a specific session
func (s *Server) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	username, err := s.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}
	
	// Verify session belongs to user
	session, err := s.sessionStorage.GetSession(r.Context(), sessionID)
	if err != nil || session == nil || session.Username != username {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	
	err = s.sessionStorage.DeleteSession(r.Context(), sessionID)
	if err != nil {
		slog.Error("Failed to delete session", "error", err, "sessionId", sessionID)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
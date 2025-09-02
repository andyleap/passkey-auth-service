package api

import (
	"encoding/json"
	"net/http"

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
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andyleap/passkey/internal/models"
	"github.com/andyleap/passkey/internal/storage"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnService struct {
	webauthn       *webauthn.WebAuthn
	userStorage    storage.UserStorage
	sessionStorage storage.SessionStorage
}

func NewWebAuthnService(webauthn *webauthn.WebAuthn, userStorage storage.UserStorage, sessionStorage storage.SessionStorage) *WebAuthnService {
	return &WebAuthnService{
		webauthn:       webauthn,
		userStorage:    userStorage,
		sessionStorage: sessionStorage,
	}
}

func (w *WebAuthnService) BeginRegistration(ctx *http.Request, username string) (*protocol.CredentialCreation, error) {
	user, err := w.userStorage.GetUser(ctx.Context(), username)
	if err != nil {
		// User doesn't exist, create new one
		user = &models.User{
			ID:          []byte(username),
			Name:        username,
			DisplayName: username,
			Credentials: []webauthn.Credential{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	options, sessionData, err := w.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	session := &models.WebAuthnSession{
		Username:  username,
		Data:      sessionData,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := w.sessionStorage.SaveWebAuthnSession(ctx.Context(), username, session); err != nil {
		return nil, fmt.Errorf("failed to save webauthn session: %w", err)
	}

	return options, nil
}

func (w *WebAuthnService) FinishRegistration(ctx *http.Request, username string) error {
	// First get the WebAuthn session to get the user that was created during BeginRegistration
	session, err := w.sessionStorage.GetWebAuthnSession(ctx.Context(), username)
	if err != nil {
		return fmt.Errorf("failed to get webauthn session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Try to get existing user, or create a new one if not found (for registration flow)
	user, err := w.userStorage.GetUser(ctx.Context(), username)
	if err != nil {
		// User doesn't exist yet (which is expected for registration), create a new one
		user = &models.User{
			ID:          []byte(username),
			Name:        username,
			DisplayName: username,
			Credentials: []webauthn.Credential{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	credential, err := w.webauthn.FinishRegistration(user, *session.Data, ctx)
	if err != nil {
		return fmt.Errorf("failed to finish registration: %w", err)
	}

	user.Credentials = append(user.Credentials, *credential)
	user.UpdatedAt = time.Now()

	if err := w.userStorage.SaveUser(ctx.Context(), user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if err := w.sessionStorage.DeleteWebAuthnSession(ctx.Context(), username); err != nil {
		return fmt.Errorf("failed to delete webauthn session: %w", err)
	}

	return nil
}

func (w *WebAuthnService) BeginLogin(ctx *http.Request, username string) (*protocol.CredentialAssertion, error) {
	user, err := w.userStorage.GetUser(ctx.Context(), username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	options, sessionData, err := w.webauthn.BeginLogin(user)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	session := &models.WebAuthnSession{
		Username:  username,
		Data:      sessionData,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := w.sessionStorage.SaveWebAuthnSession(ctx.Context(), username, session); err != nil {
		return nil, fmt.Errorf("failed to save webauthn session: %w", err)
	}

	return options, nil
}

func (w *WebAuthnService) FinishLogin(ctx *http.Request, username string) (*models.User, error) {
	user, err := w.userStorage.GetUser(ctx.Context(), username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	session, err := w.sessionStorage.GetWebAuthnSession(ctx.Context(), username)
	if err != nil {
		return nil, fmt.Errorf("failed to get webauthn session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	_, err = w.webauthn.FinishLogin(user, *session.Data, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %w", err)
	}

	if err := w.sessionStorage.DeleteWebAuthnSession(ctx.Context(), username); err != nil {
		return nil, fmt.Errorf("failed to delete webauthn session: %w", err)
	}

	return user, nil
}

func (ws *WebAuthnService) RegisterBeginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	options, err := ws.BeginRegistration(r, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("registration begin failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (ws *WebAuthnService) RegisterFinishHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	if err := ws.FinishRegistration(r, username); err != nil {
		http.Error(w, fmt.Sprintf("registration finish failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func (ws *WebAuthnService) LoginBeginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	options, err := ws.BeginLogin(r, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("login begin failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (ws *WebAuthnService) LoginFinishHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	user, err := ws.FinishLogin(r, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("login finish failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Create user session
	sessionID := generateSessionID()
	session := &models.Session{
		ID:        sessionID,
		Username:  user.Name,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := ws.sessionStorage.SaveSession(r.Context(), session); err != nil {
		http.Error(w, fmt.Sprintf("failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "authenticated",
		"sessionId": sessionID,
	})
}
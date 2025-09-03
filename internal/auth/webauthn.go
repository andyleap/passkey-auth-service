package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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
	} else {
		// User exists - check if they're authenticated or if it's their first credential
		if len(user.Credentials) > 0 {
			// User has existing credentials, check if they're authenticated
			isAuthenticated := w.isUserAuthenticated(ctx, username)
			if !isAuthenticated {
				return nil, fmt.Errorf("user already exists - please authenticate first to add additional passkeys")
			}
		}
	}

	options, sessionData, err := w.webauthn.BeginRegistration(
		user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey:      protocol.ResidentKeyRequired(),
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			UserVerification:        protocol.VerificationRequired,
		}),
	)
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

	// Try to get existing user or create new one
	user, err := w.userStorage.GetUser(ctx.Context(), username)
	if err != nil {
		// User doesn't exist yet (expected for new registration), create a new one
		user = &models.User{
			ID:          []byte(username),
			Name:        username,
			DisplayName: username,
			Credentials: []webauthn.Credential{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	} else {
		// User exists - same authentication check as in BeginRegistration
		if len(user.Credentials) > 0 {
			isAuthenticated := w.isUserAuthenticated(ctx, username)
			if !isAuthenticated {
				return fmt.Errorf("user already exists - please authenticate first to add additional passkeys")
			}
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


// BeginDiscoverableLogin starts a discoverable credential login flow (no username required)
func (w *WebAuthnService) BeginDiscoverableLogin(ctx *http.Request) (*protocol.CredentialAssertion, string, error) {
	// Generate a temporary session ID for this discoverable login attempt
	sessionID := generateSessionID()
	
	// Create assertion options for discoverable credentials
	log.Printf("DEBUG: Calling BeginDiscoverableLogin()")
	options, sessionData, err := w.webauthn.BeginDiscoverableLogin()
	if err != nil {
		log.Printf("DEBUG: BeginDiscoverableLogin failed: %v", err)
		return nil, "", fmt.Errorf("failed to begin discoverable login: %w", err)
	}
	log.Printf("DEBUG: BeginDiscoverableLogin succeeded, challenge: %x", sessionData.Challenge)

	session := &models.WebAuthnSession{
		Username:  sessionID, // Use session ID as temporary identifier
		Data:      sessionData,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := w.sessionStorage.SaveWebAuthnSession(ctx.Context(), sessionID, session); err != nil {
		return nil, "", fmt.Errorf("failed to save webauthn session: %w", err)
	}

	return options, sessionID, nil
}

// FinishDiscoverableLogin completes a discoverable credential login
func (w *WebAuthnService) FinishDiscoverableLogin(ctx *http.Request, sessionID string) (*models.User, error) {
	log.Printf("DEBUG: Starting discoverable login finish for session: %s", sessionID)
	log.Printf("DEBUG: Request Origin: %s, Host: %s", ctx.Header.Get("Origin"), ctx.Host)
	
	session, err := w.sessionStorage.GetWebAuthnSession(ctx.Context(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webauthn session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	log.Printf("DEBUG: Found session for sessionID: %s", sessionID)
	log.Printf("DEBUG: Session data challenge: %x", session.Data.Challenge)

	var foundUser *models.User
	credential, err := w.webauthn.FinishDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		log.Printf("DEBUG: FinishDiscoverableLogin callback - rawID: %x, userHandle: %x", rawID, userHandle)
		
		// Find user by user handle (which is the user ID)
		user, err := w.userStorage.GetUserByID(ctx.Context(), userHandle)
		if err != nil {
			log.Printf("DEBUG: Failed to find user by ID %x: %v", userHandle, err)
			return nil, err
		}
		
		log.Printf("DEBUG: Found user: %s with %d credentials", user.Name, len(user.Credentials))
		for i, cred := range user.Credentials {
			log.Printf("DEBUG: Credential %d - ID: %x", i, cred.ID)
		}
		
		foundUser = user // Store the user for later use
		return user, nil
	}, *session.Data, ctx)
	
	log.Printf("DEBUG: FinishDiscoverableLogin returned credential: %v", credential != nil)
	
	if err != nil {
		log.Printf("DEBUG: FinishDiscoverableLogin failed: %v", err)
		return nil, fmt.Errorf("failed to finish discoverable login: %w", err)
	}

	if foundUser == nil {
		return nil, fmt.Errorf("user not found during discoverable login")
	}

	log.Printf("DEBUG: Successfully authenticated user: %s", foundUser.Name)

	if err := w.sessionStorage.DeleteWebAuthnSession(ctx.Context(), sessionID); err != nil {
		return nil, fmt.Errorf("failed to delete webauthn session: %w", err)
	}

	return foundUser, nil
}

// isUserAuthenticated checks if the user has a valid session
func (w *WebAuthnService) isUserAuthenticated(ctx *http.Request, username string) bool {
	// Check for session cookie or header
	sessionID := ""
	
	// Try to get session ID from cookie
	if cookie, err := ctx.Cookie("session_id"); err == nil {
		sessionID = cookie.Value
	}
	
	// Try to get session ID from Authorization header
	if sessionID == "" {
		if auth := ctx.Header.Get("Authorization"); auth != "" {
			// Format: "Bearer <session_id>"
			if len(auth) > 7 && auth[:7] == "Bearer " {
				sessionID = auth[7:]
			}
		}
	}
	
	if sessionID == "" {
		return false
	}
	
	// Validate session
	session, err := w.sessionStorage.GetSession(ctx.Context(), sessionID)
	if err != nil || session == nil {
		return false
	}
	
	// Check if session belongs to the user and is not expired
	if session.Username != username || session.ExpiresAt.Before(time.Now()) {
		return false
	}
	
	return true
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
	// Discoverable credentials don't need a username
	options, sessionID, err := ws.BeginDiscoverableLogin(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("login begin failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"publicKey": options.Response,
		"sessionId": sessionID,
	})
}

func (ws *WebAuthnService) LoginFinishHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	user, err := ws.FinishDiscoverableLogin(r, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("login finish failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Create user session
	userSessionID := generateSessionID()
	session := &models.Session{
		ID:        userSessionID,
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
		"sessionId": userSessionID,
	})
}

// GetUser returns a user by username
func (w *WebAuthnService) GetUser(ctx context.Context, username string) (*models.User, error) {
	return w.userStorage.GetUser(ctx, username)
}

// DeleteCredential removes a credential from a user
func (w *WebAuthnService) DeleteCredential(ctx context.Context, username, credentialID string) error {
	user, err := w.userStorage.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	
	// Find and remove the credential
	// credentialID is base64url-encoded (URL-safe), so compare with base64url-encoded cred.ID
	newCredentials := make([]webauthn.Credential, 0, len(user.Credentials))
	found := false
	for _, cred := range user.Credentials {
		credIDBase64URL := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(cred.ID)
		if credIDBase64URL != credentialID {
			newCredentials = append(newCredentials, cred)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("credential not found")
	}
	
	// Don't allow deletion of the last credential
	if len(newCredentials) == 0 {
		return fmt.Errorf("cannot delete the last credential")
	}
	
	user.Credentials = newCredentials
	user.UpdatedAt = time.Now()
	
	return w.userStorage.SaveUser(ctx, user)
}
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/andyleap/passkey/internal/models"
	"github.com/andyleap/passkey/internal/storage"
)

type OAuthService struct {
	sessionStorage storage.SessionStorage
	// In a real implementation, you'd have client storage too
	// For now, we'll use a simple in-memory map
	clients map[string]*models.Client
}

func NewOAuthService(sessionStorage storage.SessionStorage) *OAuthService {
	// Create some default clients for demo
	clients := map[string]*models.Client{
		"demo-app": {
			ID:   "demo-app",
			Name: "Demo Application",
			RedirectURIs: []string{
				"http://localhost:3000/callback",
				"https://localhost:3000/callback",
				"http://localhost:8080/callback",
				"https://localhost:8080/callback",
			},
			CreatedAt: time.Now(),
		},
		"test-app": {
			ID:   "test-app", 
			Name: "Test Application",
			RedirectURIs: []string{
				"http://localhost:3001/callback",
				"https://localhost:3001/callback",
			},
			CreatedAt: time.Now(),
		},
	}

	return &OAuthService{
		sessionStorage: sessionStorage,
		clients:        clients,
	}
}

// ValidateAuthorizationRequest validates an OAuth authorization request
func (o *OAuthService) ValidateAuthorizationRequest(clientID, redirectURI string) (*models.Client, error) {
	client, exists := o.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("invalid client_id")
	}

	// Validate redirect URI
	validURI := false
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			validURI = true
			break
		}
	}
	if !validURI {
		return nil, fmt.Errorf("invalid redirect_uri")
	}

	return client, nil
}

// CreateAuthorizationRequest creates a new authorization request
func (o *OAuthService) CreateAuthorizationRequest(clientID, redirectURI, state string) (*models.AuthorizationRequest, error) {
	client, err := o.ValidateAuthorizationRequest(clientID, redirectURI)
	if err != nil {
		return nil, err
	}

	request := &models.AuthorizationRequest{
		ClientID:    client.ID,
		RedirectURI: redirectURI,
		State:       state,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 10 minute expiry
	}

	return request, nil
}

// CreateAuthorizationCode creates an authorization code after successful authentication
func (o *OAuthService) CreateAuthorizationCode(ctx context.Context, request *models.AuthorizationRequest, user *models.User) (*models.AuthorizationCode, error) {
	code := &models.AuthorizationCode{
		Code:        generateRandomCode(32),
		ClientID:    request.ClientID,
		RedirectURI: request.RedirectURI,
		State:       request.State,
		Username:    user.Name,
		UserID:      user.ID,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 10 minute expiry
	}

	// Store the authorization code in session storage with a special key
	codeSession := &models.Session{
		ID:        "auth_code:" + code.Code,
		Username:  user.Name,
		UserID:    user.ID,
		CreatedAt: code.CreatedAt,
		ExpiresAt: code.ExpiresAt,
	}

	if err := o.sessionStorage.SaveSession(ctx, codeSession); err != nil {
		return nil, fmt.Errorf("failed to save authorization code: %w", err)
	}

	return code, nil
}

// ExchangeAuthorizationCode exchanges an authorization code for user information
func (o *OAuthService) ExchangeAuthorizationCode(ctx context.Context, code, clientID, redirectURI string) (*models.AuthorizationCode, error) {
	// Validate client and redirect URI
	_, err := o.ValidateAuthorizationRequest(clientID, redirectURI)
	if err != nil {
		return nil, err
	}

	// Retrieve the authorization code
	session, err := o.sessionStorage.GetSession(ctx, "auth_code:"+code)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization code: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("invalid or expired authorization code")
	}

	// Delete the code (single use)
	o.sessionStorage.DeleteSession(ctx, "auth_code:"+code)

	authCode := &models.AuthorizationCode{
		Code:        code,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Username:    session.Username,
		UserID:      session.UserID,
		CreatedAt:   session.CreatedAt,
		ExpiresAt:   session.ExpiresAt,
	}

	return authCode, nil
}

// BuildRedirectURL builds the callback URL with code and state
func (o *OAuthService) BuildRedirectURL(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI // fallback
	}

	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// BuildErrorRedirectURL builds a callback URL with error information
func (o *OAuthService) BuildErrorRedirectURL(redirectURI, errorCode, errorDescription, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI // fallback
	}

	q := u.Query()
	q.Set("error", errorCode)
	if errorDescription != "" {
		q.Set("error_description", errorDescription)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// GetClient returns a client by ID
func (o *OAuthService) GetClient(clientID string) (*models.Client, bool) {
	client, exists := o.clients[clientID]
	return client, exists
}

func generateRandomCode(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/andyleap/passkey/internal/models"
	"github.com/andyleap/passkey/internal/oauth"
)

type OAuthAPIHandlers struct {
	oauthService *oauth.OAuthService
}

func NewOAuthAPIHandlers(oauthService *oauth.OAuthService) *OAuthAPIHandlers {
	return &OAuthAPIHandlers{
		oauthService: oauthService,
	}
}

// TokenHandler handles authorization code exchange
// POST /token
func (oh *OAuthAPIHandlers) TokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Code        string `json:"code"`
		ClientID    string `json:"client_id"`
		RedirectURI string `json:"redirect_uri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Code == "" || request.ClientID == "" || request.RedirectURI == "" {
		http.Error(w, "code, client_id, and redirect_uri are required", http.StatusBadRequest)
		return
	}

	// Exchange authorization code
	authCode, err := oh.oauthService.ExchangeAuthorizationCode(r.Context(), request.Code, request.ClientID, request.RedirectURI)
	if err != nil {
		slog.Error("Token exchange error", "error", err)
		http.Error(w, "Invalid authorization code", http.StatusBadRequest)
		return
	}

	// Return user information
	response := map[string]any{
		"username":   authCode.Username,
		"user_id":    authCode.UserID,
		"client_id":  authCode.ClientID,
		"expires_at": authCode.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CompleteHandler completes OAuth flow after successful authentication
// POST /oauth/complete
func (oh *OAuthAPIHandlers) CompleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username    string `json:"username"`
		ClientID    string `json:"client_id"`
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Username == "" || request.ClientID == "" || request.RedirectURI == "" {
		http.Error(w, "username, client_id, and redirect_uri are required", http.StatusBadRequest)
		return
	}

	// Create authorization request to validate the client
	authRequest, err := oh.oauthService.CreateAuthorizationRequest(request.ClientID, request.RedirectURI, request.State)
	if err != nil {
		slog.Error("Invalid authorization request", "error", err)
		http.Error(w, "Invalid authorization request", http.StatusBadRequest)
		return
	}

	// Create a minimal user object for the authorization code
	user := &models.User{
		ID:   []byte(request.Username),
		Name: request.Username,
	}

	// Create authorization code
	authCode, err := oh.oauthService.CreateAuthorizationCode(r.Context(), authRequest, user)
	if err != nil {
		slog.Error("Failed to create authorization code", "error", err)
		http.Error(w, "Failed to create authorization code", http.StatusInternalServerError)
		return
	}

	// Build redirect URL with authorization code
	redirectURL := oh.oauthService.BuildRedirectURL(request.RedirectURI, authCode.Code, request.State)

	response := map[string]string{
		"redirect_url": redirectURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
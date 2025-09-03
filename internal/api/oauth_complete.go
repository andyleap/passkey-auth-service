package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andyleap/passkey/internal/auth"
	"github.com/andyleap/passkey/internal/oauth"
	"github.com/andyleap/passkey/internal/storage"
)

type OAuthCompleteHandler struct {
	oauthService    *oauth.OAuthService
	userStorage     storage.UserStorage
	webauthnService *auth.WebAuthnService
}

func NewOAuthCompleteHandler(oauthService *oauth.OAuthService, userStorage storage.UserStorage, webauthnService *auth.WebAuthnService) *OAuthCompleteHandler {
	return &OAuthCompleteHandler{
		oauthService:    oauthService,
		userStorage:     userStorage,
		webauthnService: webauthnService,
	}
}

// CompleteHandler completes the OAuth flow after successful authentication
func (och *OAuthCompleteHandler) CompleteHandler(w http.ResponseWriter, r *http.Request) {
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

	// Validate required fields
	if request.Username == "" || request.ClientID == "" || request.RedirectURI == "" {
		http.Error(w, "username, client_id, and redirect_uri are required", http.StatusBadRequest)
		return
	}

	// Get user from storage
	user, err := och.userStorage.GetUser(r.Context(), request.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// Create authorization request
	authRequest, err := och.oauthService.CreateAuthorizationRequest(request.ClientID, request.RedirectURI, request.State)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid authorization request: %v", err), http.StatusBadRequest)
		return
	}

	// Create authorization code
	authCode, err := och.oauthService.CreateAuthorizationCode(r.Context(), authRequest, user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create authorization code: %v", err), http.StatusInternalServerError)
		return
	}

	// Build redirect URL
	redirectURL := och.oauthService.BuildRedirectURL(request.RedirectURI, authCode.Code, request.State)

	response := map[string]string{
		"redirect_url": redirectURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/andyleap/passkey/internal/models"
	"github.com/andyleap/passkey/internal/oauth"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed assets/dist/*.css assets/dist/*.js
var assetsFS embed.FS

type OAuthUIHandlers struct {
	oauthService *oauth.OAuthService
	templates    *template.Template
}

func NewOAuthUIHandlers(oauthService *oauth.OAuthService) (*OAuthUIHandlers, error) {
	// Parse embedded templates
	templates, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded templates: %w", err)
	}

	return &OAuthUIHandlers{
		oauthService: oauthService,
		templates:    templates,
	}, nil
}

// AuthorizeHandler handles OAuth authorization requests
// GET /authorize?client_id=myapp&redirect_uri=https://myapp.com/callback&state=xyz123
func (oh *OAuthUIHandlers) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	if clientID == "" {
		oh.renderErrorPage(w, "Invalid Request", "client_id is required")
		return
	}
	if redirectURI == "" {
		oh.renderErrorPage(w, "Invalid Request", "redirect_uri is required")
		return
	}

	// Validate the authorization request
	client, err := oh.oauthService.ValidateAuthorizationRequest(clientID, redirectURI)
	if err != nil {
		// For invalid client, we can't redirect back, so show error page
		oh.renderErrorPage(w, "Invalid Request", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	// Create authorization request
	authRequest, err := oh.oauthService.CreateAuthorizationRequest(clientID, redirectURI, state)
	if err != nil {
		redirectURL := oh.oauthService.BuildErrorRedirectURL(redirectURI, "server_error", "Failed to process request", state)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Render the authorization page with client info and auth request
	oh.renderAuthorizePage(w, client, authRequest)
}

// AssetsHandler serves embedded static assets
func (oh *OAuthUIHandlers) AssetsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL path (/oauth/filename.ext)
	filename := r.URL.Path[7:] // Remove "/oauth/" prefix
	
	// Security: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.NotFound(w, r)
		return
	}
	
	// Build asset path
	assetPath := "assets/dist/" + filename
	
	// Determine content type from file extension
	var contentType string
	switch {
	case strings.HasSuffix(filename, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(filename, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(filename, ".map"):
		contentType = "application/json"
	default:
		http.NotFound(w, r)
		return
	}

	data, err := assetsFS.ReadFile(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.Write(data)
}

func (oh *OAuthUIHandlers) renderAuthorizePage(w http.ResponseWriter, client *models.Client, authRequest *models.AuthorizationRequest) {
	// Prepare data for the template
	authData, _ := json.Marshal(map[string]string{
		"client_id":    authRequest.ClientID,
		"redirect_uri": authRequest.RedirectURI,
		"state":        authRequest.State,
	})

	data := struct {
		ClientName   string
		AuthDataJSON template.JS
	}{
		ClientName:   client.Name,
		AuthDataJSON: template.JS(authData),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := oh.templates.ExecuteTemplate(w, "authorize.html", data); err != nil {
		slog.Error("Failed to render authorize template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (oh *OAuthUIHandlers) renderErrorPage(w http.ResponseWriter, title, message string) {
	data := struct {
		Title   string
		Message string
	}{
		Title:   title,
		Message: message,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	if err := oh.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		slog.Error("Failed to render error template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RenderLandingPage renders the service landing page
func (oh *OAuthUIHandlers) RenderLandingPage(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/html")
	return oh.templates.ExecuteTemplate(w, "landing.html", nil)
}

// RenderControlPanel renders the user control panel
func (oh *OAuthUIHandlers) RenderControlPanel(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/html")
	return oh.templates.ExecuteTemplate(w, "control_panel.html", nil)
}

// RenderRegisterPage renders the registration page
func (oh *OAuthUIHandlers) RenderRegisterPage(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/html")
	return oh.templates.ExecuteTemplate(w, "register.html", nil)
}
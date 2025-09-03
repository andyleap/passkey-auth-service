package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/andyleap/passkey/internal/api"
	"github.com/andyleap/passkey/internal/auth"
	"github.com/andyleap/passkey/internal/oauth"
	"github.com/andyleap/passkey/internal/storage"
	"github.com/andyleap/passkey/internal/ui"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup WebAuthn
	wconfig := &webauthn.Config{
		RPDisplayName: "Passkey Authentication Service",
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		slog.Error("Failed to create WebAuthn instance", "error", err)
		os.Exit(1)
	}

	// Setup user storage
	var userStorage storage.UserStorage
	switch cfg.StorageMode {
	case "s3":
		s3Storage, err := storage.NewS3Storage(cfg.S3.Endpoint, cfg.S3.AccessKey, cfg.S3.SecretKey, cfg.S3.Bucket, cfg.S3.UseSSL)
		if err != nil {
			slog.Error("Failed to create S3 storage", "error", err)
			os.Exit(1)
		}
		userStorage = s3Storage
		slog.Info("Using S3 storage", "endpoint", cfg.S3.Endpoint, "bucket", cfg.S3.Bucket)
	case "filesystem":
		fsStorage, err := storage.NewFilesystemStorage(cfg.DataPath)
		if err != nil {
			slog.Error("Failed to create filesystem storage", "error", err)
			os.Exit(1)
		}
		userStorage = fsStorage
		slog.Info("Using filesystem storage", "path", cfg.DataPath)
	default:
		slog.Error("Invalid STORAGE_MODE", "mode", cfg.StorageMode, "valid_modes", []string{"s3", "filesystem"})
		os.Exit(1)
	}

	// Setup session storage
	var sessionStorage storage.SessionStorage
	switch cfg.SessionMode {
	case "redis":
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		// Test Redis connection
		ctx := context.Background()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Error("Failed to connect to Redis", "error", err)
			os.Exit(1)
		}

		sessionStorage = storage.NewRedisStorage(redisClient)
		slog.Info("Using Redis sessions", "addr", cfg.Redis.Addr)
	case "memory":
		sessionStorage = storage.NewMemoryStorage()
		slog.Warn("Using in-memory sessions (not persistent)")
	default:
		slog.Error("Invalid SESSION_MODE", "mode", cfg.SessionMode, "valid_modes", []string{"redis", "memory"})
		os.Exit(1)
	}

	// Setup services
	webauthnService := auth.NewWebAuthnService(webAuthn, userStorage, sessionStorage)
	oauthService := oauth.NewOAuthService(sessionStorage, LoadedOAuthClients)
	apiServer := api.NewServer(webauthnService, sessionStorage)

	// Setup OAuth handlers
	oauthUIHandlers, err := ui.NewOAuthUIHandlers(oauthService)
	if err != nil {
		slog.Error("Failed to create OAuth UI handlers", "error", err)
		os.Exit(1)
	}
	oauthAPIHandlers := api.NewOAuthAPIHandlers(oauthService)

	// Setup routes
	mux := http.NewServeMux()

	// OAuth routes (main flow)
	mux.HandleFunc("GET /authorize", oauthUIHandlers.AuthorizeHandler)
	mux.HandleFunc("POST /oauth/complete", oauthAPIHandlers.CompleteHandler)
	mux.HandleFunc("POST /oauth/token", oauthAPIHandlers.TokenHandler)

	// OAuth static assets (embedded) - simplified wildcard handler
	mux.HandleFunc("GET /oauth/{filename}", oauthUIHandlers.AssetsHandler)

	// API routes (for direct integration)
	mux.HandleFunc("POST /api/v1/register/begin", webauthnService.RegisterBeginHandler)
	mux.HandleFunc("POST /api/v1/register/finish", webauthnService.RegisterFinishHandler)
	mux.HandleFunc("POST /api/v1/login/begin", webauthnService.LoginBeginHandler)
	mux.HandleFunc("POST /api/v1/login/finish", webauthnService.LoginFinishHandler)
	mux.HandleFunc("POST /api/v1/logout", apiServer.LogoutHandler)
	mux.HandleFunc("GET /api/v1/validate/{sessionId}", apiServer.ValidateSessionHandler)
	mux.HandleFunc("GET /health", apiServer.HealthHandler)

	// Control panel API routes
	mux.HandleFunc("GET /api/v1/user/credentials", apiServer.UserCredentialsHandler)
	mux.HandleFunc("GET /api/v1/user/sessions", apiServer.UserSessionsHandler)
	mux.HandleFunc("DELETE /api/v1/user/credentials/{credentialId}", apiServer.DeleteCredentialHandler)
	mux.HandleFunc("DELETE /api/v1/user/sessions/{sessionId}", apiServer.DeleteSessionHandler)

	// Index page (landing or redirect)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveIndex(w, r, cfg, oauthUIHandlers, apiServer, sessionStorage)
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if err := oauthUIHandlers.RenderRegisterPage(w); err != nil {
			slog.Error("Failed to render register page", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Apply middleware
	handler := api.LoggingMiddleware(api.CORSMiddleware(mux))

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	fmt.Printf("Passkey Authentication Service starting on http://localhost:%s\n", cfg.Port)
	fmt.Println("OAuth endpoints:")
	fmt.Println("  GET  /authorize              - OAuth authorization (redirect apps here)")
	fmt.Println("  POST /oauth/token            - Token exchange")
	fmt.Println("API endpoints:")
	fmt.Println("  POST /api/v1/register/begin  - WebAuthn registration")
	fmt.Println("  POST /api/v1/register/finish")
	fmt.Println("  POST /api/v1/login/begin     - WebAuthn login")
	fmt.Println("  POST /api/v1/login/finish")
	fmt.Println("  POST /api/v1/logout          - Logout")
	fmt.Println("  GET  /api/v1/validate/{sessionId} - Session validation")
	fmt.Println("  GET  /health                 - Health check")
	fmt.Println()
	fmt.Printf("Demo clients configured: demo-app, test-app\n")
	fmt.Printf("Example OAuth URL: http://localhost:%s/authorize?client_id=demo-app&redirect_uri=http://localhost:3000/callback&state=xyz123\n", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request, cfg *Config, uiHandlers *ui.OAuthUIHandlers, apiServer *api.Server, sessionStorage storage.SessionStorage) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// If redirect URL is configured, redirect to it
	if cfg.IndexRedirect != "" {
		http.Redirect(w, r, cfg.IndexRedirect, http.StatusFound)
		return
	}

	// Check if user is authenticated - if so, show control panel
	if isAuthenticated(r, sessionStorage) {
		if err := uiHandlers.RenderControlPanel(w); err != nil {
			slog.Error("Failed to render control panel", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Otherwise serve the landing page using the UI templates
	if err := uiHandlers.RenderLandingPage(w); err != nil {
		slog.Error("Failed to render landing page", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// isAuthenticated checks if the request has a valid session
func isAuthenticated(r *http.Request, sessionStorage storage.SessionStorage) bool {
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
		return false
	}

	session, err := sessionStorage.GetSession(r.Context(), sessionID)
	if err != nil || session == nil {
		return false
	}

	return session.ExpiresAt.After(time.Now())
}

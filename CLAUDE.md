# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

```bash
# Quick development start (no external dependencies)
make run                 # Local filesystem + memory storage

# Development with asset watching
make dev                 # Starts asset watcher + server

# Build application
make build               # Builds assets and Go binary to bin/
make build-assets        # Build only CSS/JS assets with esbuild

# Testing and code quality
go test ./...            # Run all tests
make fmt                 # Format Go code
make lint                # Lint code (requires golangci-lint)

# External services for testing
make dev-deps           # Start Redis + MinIO containers
make run-external       # Run with S3 + Redis backends
make docker-up          # Full stack with Docker

# Frontend asset building
npm run build           # Build assets with esbuild
npm run build:watch     # Watch and rebuild assets
```

## Architecture Overview

### Core Components

**Passkey Authentication Service** - A production-ready WebAuthn/Passkey authentication service that provides authentication for other services using pluggable storage backends.

**Storage Layer** (`internal/storage/`):
- **UserStorage**: Handles user data with filesystem or S3-compatible backends
- **SessionStorage**: Manages WebAuthn and user sessions with memory or Redis backends
- Pluggable architecture allows switching between development (filesystem/memory) and production (S3/Redis) configurations

**Authentication Flow** (`internal/auth/`):
- WebAuthn service handles discoverable credential flows (no username required)
- Supports resident keys with user verification for secure authentication
- Automatic session creation and management after successful authentication

**API Layer** (`internal/api/`):
- RESTful endpoints for WebAuthn registration/login
- OAuth 2.0 authorization code flow for service integration
- Session validation endpoints for other services

**UI Layer** (`internal/ui/`):
- HTML templates with embedded CSS/JS assets
- Preact-based control panel for credential management
- OAuth authorization flow with clean user interface

### Configuration

Environment-driven configuration with sensible defaults:
- `STORAGE_MODE`: "filesystem" (dev) or "s3" (production)
- `SESSION_MODE`: "memory" (dev) or "redis" (production)
- WebAuthn configured for localhost by default, customizable via `RP_ID`/`RP_ORIGIN`

### Frontend Build System

- **esbuild** via Node.js build script (`build.js`)
- Builds both CSS (including blue-design-system) and JSX (Preact) components
- Output to `internal/ui/assets/dist/` for Go embed
- Supports watch mode for development

## Project Structure

```
├── cmd/server/          # Main application entry point
├── internal/
│   ├── api/            # HTTP handlers, middleware, OAuth API
│   ├── auth/           # WebAuthn service implementation
│   ├── models/         # Data models (User, Session, OAuth)
│   ├── oauth/          # OAuth 2.0 service logic
│   ├── storage/        # Storage interfaces and implementations
│   └── ui/             # UI handlers, templates, assets
├── blue-design-system/ # Git submodule for design system
├── build.js           # esbuild configuration and build script
├── Makefile           # Development commands
└── docker-compose.yml # Full stack development environment
```

## WebAuthn Implementation

Uses discoverable credentials (resident keys) for passwordless authentication:
- Registration requires user verification and resident key support
- Login flow doesn't require username input - credentials are discovered automatically
- Supports adding multiple passkeys per user (if authenticated)
- Prevents registration conflicts when user already exists

## OAuth 2.0 Flow

Standard authorization code flow:
1. Client redirects to `/authorize` with client_id, redirect_uri, state
2. User authenticates with passkey
3. User authorizes the client application
4. Service redirects back with authorization code
5. Client exchanges code for session token at `/oauth/token`

## Session Management

Dual session system:
- **WebAuthn sessions**: Temporary (5min) for authentication flows
- **User sessions**: Long-lived (24h) for authenticated access
- Sessions validate via cookie or Bearer token
- Integration with other services via session validation endpoint

## Development Notes

- Frontend assets are embedded in Go binary via `embed` directive
- Blue design system is included as git submodule
- Local development uses filesystem storage in `./data/` directory
- Production supports S3-compatible storage (AWS S3, MinIO, etc.)
- HTTPS with self-signed certificates for WebAuthn compliance
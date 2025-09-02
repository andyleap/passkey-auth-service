.PHONY: build-assets build run test docker-up docker-down clean dev

# Build CSS and JS assets with esbuild
build-assets:
	@echo "🔨 Building assets..."
	@npm run build

# Build the application
build: build-assets
	@echo "🏗️  Building server..."
	go build -o bin/passkey-auth ./cmd/server

# Run the application (local filesystem + memory, no external deps)
run: build-assets
	@echo "🚀 Starting server..."
	STORAGE_MODE=filesystem SESSION_MODE=memory go run ./cmd/server

# Development mode with asset watching
dev:
	@echo "👀 Starting development server with asset watching..."
	npm run build:watch &
	STORAGE_MODE=filesystem SESSION_MODE=memory go run ./cmd/server

# Run with S3 and Redis (requires external services)
run-external:
	STORAGE_MODE=s3 SESSION_MODE=redis go run ./cmd/server

# Run tests
test:
	go test ./...

# Start development environment with Docker
docker-up:
	docker-compose up -d

# Stop development environment
docker-down:
	docker-compose down

# Build Docker image
docker-build:
	docker build -t passkey-auth:latest .

# Run with Docker (full stack)
docker-run: docker-build docker-up

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Development setup (start dependencies)
dev-deps:
	docker-compose up -d redis minio minio-setup

# Stop dev dependencies
dev-deps-stop:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Tidy go modules
tidy:
	go mod tidy

# Help
help:
	@echo "Available targets:"
	@echo "  build-assets - Build CSS/JS assets with esbuild"
	@echo "  build        - Build the application binary (includes assets)"
	@echo "  run          - Run locally (filesystem + memory, no deps)"
	@echo "  dev          - Run with asset watching"
	@echo "  run-external - Run with S3 + Redis (requires external services)"
	@echo "  test         - Run tests"
	@echo "  docker-up    - Start full development environment"
	@echo "  docker-down  - Stop development environment"
	@echo "  docker-run   - Build and run with Docker"
	@echo "  dev-deps     - Start only Redis and MinIO for development"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  tidy         - Tidy go modules"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Configuration options:"
	@echo "  INDEX_REDIRECT - Redirect index page to URL (default: show landing page)"
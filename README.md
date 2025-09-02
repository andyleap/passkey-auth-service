# Passkey Authentication Service

A production-ready WebAuthn/Passkey authentication service that provides authentication for other services using Redis session management and S3-compatible storage.

## Features

- 🔐 WebAuthn/Passkey authentication
- 🗄️ S3-compatible storage for user data (AWS S3, MinIO, etc.)
- ⚡ Redis-based session management
- 🚀 RESTful API for service integration
- 🐳 Docker containerization
- 🔧 Environment-based configuration
- 🎨 Clean demo UI

## Architecture

- **API Layer**: RESTful endpoints for authentication and session management
- **Storage**: Lightweight MinIO client for S3-compatible storage, Redis for sessions
- **Session Flow**: Redis-based session validation for other services
- **Security**: HTTPS with auto-generated certificates, secure session handling

## Quick Start

### Option 1: Local Development (No Docker Required)

The fastest way to get started - zero external dependencies:

```bash
# 1. Clone and build
git clone <repository>
cd passkey
go mod download

# 2. Run locally (uses filesystem + memory storage)
make run
# or: go run ./cmd/server

# 3. Visit https://localhost:8443 and test!
```

That's it! No Docker, Redis, or MinIO required for development.

### Option 2: Docker Compose (Full Production Stack)

1. **Start the full stack:**
```bash
make docker-up
```

2. Visit `https://localhost:8443` and accept the self-signed certificate
3. Test the authentication flow in the demo interface

### Option 3: Local with External Services

For testing production-like setup locally:

1. **Start external services:**
```bash
make dev-deps  # Start Redis + MinIO in Docker
```

2. **Run with external storage:**
```bash
make run-external  # Uses S3 + Redis
```

### Option 4: Manual Setup

1. **Prerequisites:**
   - Go 1.25+
   - Optional: Redis server, MinIO/S3

2. **Build and run:**
```bash
make build
./bin/passkey-auth
```

## API Endpoints

### Authentication Endpoints

- `POST /api/v1/register/begin?username=user` - Begin passkey registration
- `POST /api/v1/register/finish?username=user` - Complete passkey registration
- `POST /api/v1/login/begin?username=user` - Begin passkey login
- `POST /api/v1/login/finish?username=user` - Complete passkey login (returns sessionId)

### Session Management

- `GET /api/v1/validate/{sessionId}` - Validate session for other services
- `POST /api/v1/logout` - Logout (requires X-Session-ID header or sessionId param)

### Health

- `GET /health` - Service health check

## Integration with Other Services

Other services can integrate by:

1. **Direct Redis Access**: Check sessions directly in Redis with key pattern `session:{sessionId}`
2. **HTTP Validation**: Use `GET /api/v1/validate/{sessionId}` endpoint

### Example Session Validation

```javascript
// Direct Redis check (Node.js example)
const session = await redis.get(`session:${sessionId}`);
if (session) {
    const userData = JSON.parse(session);
    // User is authenticated
}

// HTTP validation
const response = await fetch(`https://auth-service/api/v1/validate/${sessionId}`);
if (response.ok) {
    const userData = await response.json();
    // User is authenticated
}
```

## Environment Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8443` |
| `RP_ID` | Relying party ID | `localhost` |
| `RP_ORIGIN` | Relying party origin | `https://localhost:8443` |
| `STORAGE_MODE` | User storage: "filesystem" or "s3" | `filesystem` |
| `SESSION_MODE` | Session storage: "memory" or "redis" | `memory` |
| `DATA_PATH` | Filesystem storage path | `./data` |
| `S3_ENDPOINT` | S3/MinIO endpoint (host:port) | `localhost:9000` |
| `S3_BUCKET` | S3 bucket name | `passkey-auth` |
| `S3_ACCESS_KEY` | S3 access key | `minioadmin` |
| `S3_SECRET_KEY` | S3 secret key | `minioadmin` |
| `S3_USE_SSL` | Use SSL for S3 connections | `false` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database | `0` |

### Storage Modes

- **filesystem**: Stores user data as JSON files in `DATA_PATH` (great for development)
- **s3**: Uses S3-compatible storage (MinIO, AWS S3, DigitalOcean Spaces, etc.)

### Session Modes

- **memory**: In-memory sessions (fast, but lost on restart)
- **redis**: Persistent Redis sessions (survives restarts, can be shared across instances)

## Development

### Project Structure

```
├── cmd/server/          # Main application
├── internal/
│   ├── api/            # HTTP handlers and middleware
│   ├── auth/           # WebAuthn service
│   ├── models/         # Data models
│   └── storage/        # S3 and Redis storage
├── web/
│   ├── static/         # CSS and JavaScript
│   └── templates/      # HTML templates
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### Building

```bash
# Build binary
go build -o passkey-auth ./cmd/server

# Build Docker image
docker build -t passkey-auth .
```

## Security Considerations

- Uses HTTPS with auto-generated self-signed certificates
- Sessions have configurable TTL (default: 24 hours)
- WebAuthn sessions expire after 5 minutes
- Credentials stored encrypted in S3
- CORS configured for cross-origin requests

## Production Deployment

1. **Use proper certificates**: Replace self-signed certs with valid SSL certificates
2. **Secure Redis**: Configure Redis password and network security
3. **S3 Security**: Use proper IAM roles and bucket policies
4. **Environment**: Set proper RP_ID and RP_ORIGIN for your domain
5. **Monitoring**: Add health checks and monitoring
6. **Backup**: Implement backup strategies for Redis and S3

## License

[Add your license here]
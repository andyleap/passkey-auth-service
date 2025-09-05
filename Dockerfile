FROM node:20-alpine AS asset-builder

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci

# Copy source files for asset building
COPY build.js ./
COPY blue-design-system ./blue-design-system
COPY internal/ui/assets ./internal/ui/assets

# Build assets
RUN npm run build

# Go builder stage
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache ca-certificates git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built assets from asset-builder
COPY --from=asset-builder /app/internal/ui/assets/dist ./internal/ui/assets/dist

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o passkey-auth ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary and set execute permissions
COPY --from=go-builder /app/passkey-auth .
RUN chmod +x ./passkey-auth

# Create non-root user
RUN addgroup -g 1001 -S passkey && \
    adduser -S -D -H -u 1001 -s /sbin/nologin passkey -G passkey

# Change ownership
RUN chown -R passkey:passkey /root

USER passkey

# Expose port
EXPOSE 8443

# Run the application
CMD ["./passkey-auth"]
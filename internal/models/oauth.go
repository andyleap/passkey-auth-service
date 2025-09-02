package models

import (
	"time"
)

// Client represents an OAuth client application
type Client struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RedirectURIs []string  `json:"redirect_uris"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthorizationRequest represents an OAuth authorization request
type AuthorizationRequest struct {
	ClientID     string    `json:"client_id"`
	RedirectURI  string    `json:"redirect_uri"`
	State        string    `json:"state"`
	Username     string    `json:"username,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthorizationCode represents an authorization code
type AuthorizationCode struct {
	Code         string    `json:"code"`
	ClientID     string    `json:"client_id"`
	RedirectURI  string    `json:"redirect_uri"`
	State        string    `json:"state"`
	Username     string    `json:"username"`
	UserID       []byte    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}
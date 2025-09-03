package models

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

type Session struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	UserID    []byte    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type WebAuthnSession struct {
	Username  string                `json:"username"`
	Data      *webauthn.SessionData `json:"data"`
	ExpiresAt time.Time             `json:"expiresAt"`
}

package storage

import (
	"context"

	"github.com/andyleap/passkey/internal/models"
)

type UserStorage interface {
	GetUser(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, userID []byte) (*models.User, error)
	SaveUser(ctx context.Context, user *models.User) error
	UserExists(ctx context.Context, username string) (bool, error)
}

type SessionStorage interface {
	SaveWebAuthnSession(ctx context.Context, username string, session *models.WebAuthnSession) error
	GetWebAuthnSession(ctx context.Context, username string) (*models.WebAuthnSession, error)
	DeleteWebAuthnSession(ctx context.Context, username string) error
	
	SaveSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, sessionID string) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	GetUserSessions(ctx context.Context, username string) ([]*models.Session, error)
}
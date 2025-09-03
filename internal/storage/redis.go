package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andyleap/passkey/internal/models"
	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

func (r *RedisStorage) SaveWebAuthnSession(ctx context.Context, username string, session *models.WebAuthnSession) error {
	key := fmt.Sprintf("webauthn_session:%s", username)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal webauthn session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save webauthn session: %w", err)
	}

	return nil
}

func (r *RedisStorage) GetWebAuthnSession(ctx context.Context, username string) (*models.WebAuthnSession, error) {
	key := fmt.Sprintf("webauthn_session:%s", username)

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webauthn session: %w", err)
	}

	var session models.WebAuthnSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webauthn session: %w", err)
	}

	return &session, nil
}

func (r *RedisStorage) DeleteWebAuthnSession(ctx context.Context, username string) error {
	key := fmt.Sprintf("webauthn_session:%s", username)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisStorage) SaveSession(ctx context.Context, session *models.Session) error {
	key := fmt.Sprintf("session:%s", session.ID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (r *RedisStorage) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		r.client.Del(ctx, key)
		return nil, nil
	}

	return &session, nil
}

func (r *RedisStorage) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisStorage) GetUserSessions(ctx context.Context, username string) ([]*models.Session, error) {
	// Get all session keys
	keys, err := r.client.Keys(ctx, "session:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session keys: %w", err)
	}

	var userSessions []*models.Session
	now := time.Now()

	// Check each session to see if it belongs to the user
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Result()
		if err == redis.Nil {
			continue // Session was deleted between keys() and get()
		}
		if err != nil {
			continue // Skip problematic sessions
		}

		var session models.Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue // Skip malformed sessions
		}

		// Check if session belongs to user and is not expired
		if session.Username == username && now.Before(session.ExpiresAt) {
			userSessions = append(userSessions, &session)
		}
	}

	return userSessions, nil
}

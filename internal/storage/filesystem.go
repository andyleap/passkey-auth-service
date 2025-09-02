package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andyleap/passkey/internal/models"
)

type FilesystemStorage struct {
	basePath string
}

func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	// Ensure the base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	// Create users subdirectory
	usersPath := filepath.Join(basePath, "users")
	if err := os.MkdirAll(usersPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create users path: %w", err)
	}

	return &FilesystemStorage{
		basePath: basePath,
	}, nil
}

func (f *FilesystemStorage) GetUser(ctx context.Context, username string) (*models.User, error) {
	userPath := filepath.Join(f.basePath, "users", username+".json")
	
	data, err := os.ReadFile(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read user file: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (f *FilesystemStorage) SaveUser(ctx context.Context, user *models.User) error {
	userPath := filepath.Join(f.basePath, "users", user.Name+".json")
	
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	if err := os.WriteFile(userPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write user file: %w", err)
	}

	return nil
}

func (f *FilesystemStorage) UserExists(ctx context.Context, username string) (bool, error) {
	userPath := filepath.Join(f.basePath, "users", username+".json")
	
	_, err := os.Stat(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user file: %w", err)
	}
	
	return true, nil
}
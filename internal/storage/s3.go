package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/andyleap/passkey/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewS3Storage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &S3Storage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *S3Storage) GetUser(ctx context.Context, username string) (*models.User, error) {
	key := fmt.Sprintf("users/%s.json", username)
	
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user from S3: %w", err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read user data: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (s *S3Storage) GetUserByID(ctx context.Context, userID []byte) (*models.User, error) {
	// For S3 storage, we need to list all users and search for the matching ID
	// This is not optimal but works for the current implementation
	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix: "users/",
	})

	for object := range objectCh {
		if object.Err != nil {
			continue
		}

		if !strings.HasSuffix(object.Key, ".json") {
			continue
		}

		// Get the user object
		obj, err := s.client.GetObject(ctx, s.bucket, object.Key, minio.GetObjectOptions{})
		if err != nil {
			continue // Skip problematic objects
		}

		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			continue // Skip objects that can't be read
		}

		var user models.User
		if err := json.Unmarshal(data, &user); err != nil {
			continue // Skip malformed objects
		}

		// Check if this user's ID matches
		if string(user.ID) == string(userID) {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

func (s *S3Storage) SaveUser(ctx context.Context, user *models.User) error {
	key := fmt.Sprintf("users/%s.json", user.Name)
	
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to save user to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) UserExists(ctx context.Context, username string) (bool, error) {
	key := fmt.Sprintf("users/%s.json", username)
	
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if it's a "not found" error
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}
	
	return true, nil
}
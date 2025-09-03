package main

import (
	"fmt"
	"os"

	"github.com/andyleap/passkey/internal/models"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration options
type Config struct {
	// Server config
	Port          string   `long:"port" env:"PORT" default:"8443" description:"Server port"`
	RPID          string   `long:"rp-id" env:"RP_ID" default:"localhost" description:"Relying party ID"`
	RPOrigins     []string `long:"rp-origin" env:"RP_ORIGIN" env-delim:"," default:"http://localhost:8443" description:"Relying party origins"`
	IndexRedirect string   `long:"index-redirect" env:"INDEX_REDIRECT" description:"URL to redirect index page to (leave empty for landing page)"`

	// Storage config
	StorageMode string `long:"storage-mode" env:"STORAGE_MODE" default:"filesystem" choice:"filesystem" choice:"s3" description:"User storage backend"`
	SessionMode string `long:"session-mode" env:"SESSION_MODE" default:"memory" choice:"memory" choice:"redis" description:"Session storage backend"`

	// Filesystem storage
	DataPath string `long:"data-path" env:"DATA_PATH" default:"./data" description:"Filesystem storage directory"`

	// S3 storage
	S3 struct {
		Endpoint  string `long:"s3-endpoint" env:"S3_ENDPOINT" default:"localhost:9000" description:"S3 endpoint (host:port)"`
		Bucket    string `long:"s3-bucket" env:"S3_BUCKET" default:"passkey-auth" description:"S3 bucket name"`
		AccessKey string `long:"s3-access-key" env:"S3_ACCESS_KEY" default:"minioadmin" description:"S3 access key"`
		SecretKey string `long:"s3-secret-key" env:"S3_SECRET_KEY" default:"minioadmin" description:"S3 secret key"`
		UseSSL    bool   `long:"s3-use-ssl" env:"S3_USE_SSL" description:"Use SSL for S3 connections"`
	} `group:"S3 Storage Options"`

	// Redis config
	Redis struct {
		Addr     string `long:"redis-addr" env:"REDIS_ADDR" default:"localhost:6379" description:"Redis address"`
		Password string `long:"redis-password" env:"REDIS_PASSWORD" description:"Redis password"`
		DB       int    `long:"redis-db" env:"REDIS_DB" default:"0" description:"Redis database number"`
	} `group:"Redis Options"`

	// OAuth config
	OAuthClientsFile string `long:"oauth-clients-file" env:"OAUTH_CLIENTS_FILE" description:"Path to OAuth clients YAML configuration file"`
}

// LoadConfig parses configuration from environment variables and command line flags
func LoadConfig() (*Config, error) {
	var config Config

	parser := flags.NewParser(&config, flags.Default)
	parser.Usage = "[OPTIONS]"

	if _, err := parser.Parse(); err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		}
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Load OAuth clients if configured
	if err := config.loadOAuthClients(); err != nil {
		return nil, fmt.Errorf("failed to load OAuth clients: %w", err)
	}

	return &config, nil
}

// OAuthClientsConfig holds the YAML OAuth client configurations
type OAuthClientsConfig struct {
	Clients []*models.Client `yaml:"clients"`
}

// LoadedOAuthClients stores the loaded OAuth clients
var LoadedOAuthClients map[string]*models.Client

// loadOAuthClients loads OAuth clients from YAML file
func (c *Config) loadOAuthClients() error {
	if c.OAuthClientsFile == "" {
		// No file specified, use defaults
		LoadedOAuthClients = getDefaultOAuthClients()
		return nil
	}

	data, err := os.ReadFile(c.OAuthClientsFile)
	if err != nil {
		return fmt.Errorf("failed to read OAuth clients file %s: %w", c.OAuthClientsFile, err)
	}

	var config OAuthClientsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML OAuth clients file: %w", err)
	}

	// Convert to map for easy lookup and validate
	LoadedOAuthClients = make(map[string]*models.Client)
	for _, client := range config.Clients {
		if client.ID == "" {
			return fmt.Errorf("OAuth client missing required 'id' field")
		}
		if client.Name == "" {
			client.Name = client.ID // Default name to ID
		}
		if len(client.RedirectURIs) == 0 {
			return fmt.Errorf("OAuth client '%s' missing required 'redirect_uris' field", client.ID)
		}
		LoadedOAuthClients[client.ID] = client
	}

	return nil
}

// getDefaultOAuthClients returns the default OAuth clients for development
func getDefaultOAuthClients() map[string]*models.Client {
	return map[string]*models.Client{
		"demo-app": {
			ID:   "demo-app",
			Name: "Demo Application",
			RedirectURIs: []string{
				"http://localhost:3000/callback",
				"https://localhost:3000/callback",
				"http://localhost:8080/callback",
				"https://localhost:8080/callback",
			},
		},
		"test-app": {
			ID:   "test-app",
			Name: "Test Application",
			RedirectURIs: []string{
				"http://localhost:3001/callback",
				"https://localhost:3001/callback",
			},
		},
	}
}

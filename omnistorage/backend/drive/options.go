package drive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Config holds configuration for the Google Drive backend.
type Config struct {
	// RootFolderID is the ID of the folder to use as the root.
	// If empty, "root" (My Drive root) is used.
	// This can be a folder ID or a shared drive ID.
	RootFolderID string

	// CredentialsFile is the path to a service account JSON key file
	// or OAuth2 credentials file.
	CredentialsFile string

	// CredentialsJSON is the raw JSON credentials (alternative to CredentialsFile).
	// Service account or OAuth2 credentials.
	CredentialsJSON []byte

	// TokenFile is the path to store/read OAuth2 tokens (for user credentials).
	// Only used with OAuth2 user credentials, not service accounts.
	TokenFile string

	// Token is an existing OAuth2 token (alternative to TokenFile).
	Token *oauth2.Token

	// Scopes defines the OAuth2 scopes to request.
	// Default is drive.DriveFileScope for full file access.
	Scopes []string

	// SharedDrive enables shared drive (Team Drive) support.
	// When true, operations will include shared drives.
	SharedDrive bool

	// ChunkSize is the size of chunks for resumable uploads (in bytes).
	// Default is 8MB. Must be a multiple of 256KB.
	ChunkSize int64
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		RootFolderID: "root",
		Scopes:       []string{drive.DriveFileScope},
		ChunkSize:    8 * 1024 * 1024, // 8MB
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.CredentialsFile == "" && len(c.CredentialsJSON) == 0 {
		return errors.New("either CredentialsFile or CredentialsJSON must be provided")
	}
	if c.ChunkSize > 0 && c.ChunkSize%(256*1024) != 0 {
		return errors.New("ChunkSize must be a multiple of 256KB")
	}
	return nil
}

// ConfigFromMap creates a Config from a string map.
// Useful for configuration from environment variables or config files.
func ConfigFromMap(m map[string]string) Config {
	cfg := DefaultConfig()

	if v, ok := m["root_folder_id"]; ok {
		cfg.RootFolderID = v
	}
	if v, ok := m["credentials_file"]; ok {
		cfg.CredentialsFile = v
	}
	if v, ok := m["credentials_json"]; ok {
		cfg.CredentialsJSON = []byte(v)
	}
	if v, ok := m["token_file"]; ok {
		cfg.TokenFile = v
	}
	if v, ok := m["shared_drive"]; ok {
		cfg.SharedDrive = v == "true" || v == "1"
	}
	if v, ok := m["chunk_size"]; ok {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.ChunkSize = size
		}
	}

	return cfg
}

// ConfigFromEnv creates a Config from environment variables.
// Environment variables:
//   - OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID
//   - OMNISTORAGE_GDRIVE_CREDENTIALS_FILE
//   - OMNISTORAGE_GDRIVE_CREDENTIALS_JSON
//   - OMNISTORAGE_GDRIVE_TOKEN_FILE
//   - OMNISTORAGE_GDRIVE_SHARED_DRIVE
//   - OMNISTORAGE_GDRIVE_CHUNK_SIZE
func ConfigFromEnv() Config {
	return ConfigFromMap(map[string]string{
		"root_folder_id":   os.Getenv("OMNISTORAGE_GDRIVE_ROOT_FOLDER_ID"),
		"credentials_file": os.Getenv("OMNISTORAGE_GDRIVE_CREDENTIALS_FILE"),
		"credentials_json": os.Getenv("OMNISTORAGE_GDRIVE_CREDENTIALS_JSON"),
		"token_file":       os.Getenv("OMNISTORAGE_GDRIVE_TOKEN_FILE"),
		"shared_drive":     os.Getenv("OMNISTORAGE_GDRIVE_SHARED_DRIVE"),
		"chunk_size":       os.Getenv("OMNISTORAGE_GDRIVE_CHUNK_SIZE"),
	})
}

// createDriveService creates a Google Drive service from the config.
func (c *Config) createDriveService(ctx context.Context) (*drive.Service, error) {
	var opts []option.ClientOption

	// Get credentials JSON
	var credJSON []byte
	if len(c.CredentialsJSON) > 0 {
		credJSON = c.CredentialsJSON
	} else if c.CredentialsFile != "" {
		var err error
		credJSON, err = os.ReadFile(c.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", err)
		}
	}

	// Determine scopes
	scopes := c.Scopes
	if len(scopes) == 0 {
		scopes = []string{drive.DriveFileScope}
	}

	// Try to parse as service account first
	//nolint:staticcheck // SA1019: credentials source is controlled by application config
	if creds, err := google.CredentialsFromJSON(ctx, credJSON, scopes...); err == nil {
		// Check if it's a service account by looking at the type field
		var credType struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(credJSON, &credType) == nil && credType.Type == "service_account" {
			opts = append(opts, option.WithCredentials(creds))
			return drive.NewService(ctx, opts...)
		}
	}

	// Otherwise, treat as OAuth2 client credentials
	config, err := google.ConfigFromJSON(credJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Get token
	var token *oauth2.Token
	if c.Token != nil {
		token = c.Token
	} else if c.TokenFile != "" {
		token, err = tokenFromFile(c.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file: %w", err)
		}
	} else {
		return nil, errors.New("OAuth2 credentials require a token (Token or TokenFile)")
	}

	client := config.Client(ctx, token)
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

// tokenFromFile reads an OAuth2 token from a file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

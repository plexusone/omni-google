package gcs

import (
	"errors"
	"os"
	"strconv"
)

// Errors specific to the GCS backend.
var (
	ErrBucketRequired = errors.New("gcs: bucket is required")
)

// Config holds configuration for the GCS backend.
type Config struct {
	// Bucket is the GCS bucket name (required).
	Bucket string

	// Project is the Google Cloud project ID.
	// If empty, uses GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT environment variable.
	Project string

	// Prefix is an optional prefix for all object paths.
	// Useful for organizing data within a bucket.
	Prefix string

	// CredentialsFile is the path to a service account JSON key file.
	// If empty, uses GOOGLE_APPLICATION_CREDENTIALS environment variable
	// or Application Default Credentials.
	CredentialsFile string

	// CredentialsJSON is the raw JSON credentials content.
	// Takes precedence over CredentialsFile if both are set.
	CredentialsJSON []byte

	// ChunkSize is the size in bytes for resumable upload chunks.
	// Default: 16MB.
	// Must be a multiple of 256KB.
	ChunkSize int

	// Concurrency is the number of concurrent operations.
	// Default: 5.
	Concurrency int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		ChunkSize:   16 * 1024 * 1024, // 16MB
		Concurrency: 5,
	}
}

// ConfigFromEnv creates a Config from environment variables.
// Environment variables:
//   - OMNISTORAGE_GCS_BUCKET or GCS_BUCKET: bucket name
//   - OMNISTORAGE_GCS_PROJECT or GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT: project ID
//   - OMNISTORAGE_GCS_PREFIX: object prefix
//   - GOOGLE_APPLICATION_CREDENTIALS: path to credentials file
//   - OMNISTORAGE_GCS_CHUNK_SIZE: chunk size in bytes
//   - OMNISTORAGE_GCS_CONCURRENCY: number of concurrent operations
func ConfigFromEnv() Config {
	config := DefaultConfig()

	// Bucket
	if v := os.Getenv("OMNISTORAGE_GCS_BUCKET"); v != "" {
		config.Bucket = v
	} else if v := os.Getenv("GCS_BUCKET"); v != "" {
		config.Bucket = v
	}

	// Project
	if v := os.Getenv("OMNISTORAGE_GCS_PROJECT"); v != "" {
		config.Project = v
	} else if v := os.Getenv("GOOGLE_CLOUD_PROJECT"); v != "" {
		config.Project = v
	} else if v := os.Getenv("GCLOUD_PROJECT"); v != "" {
		config.Project = v
	}

	// Prefix
	if v := os.Getenv("OMNISTORAGE_GCS_PREFIX"); v != "" {
		config.Prefix = v
	}

	// Credentials file
	if v := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); v != "" {
		config.CredentialsFile = v
	}

	// Chunk size
	if v := os.Getenv("OMNISTORAGE_GCS_CHUNK_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil && size > 0 {
			config.ChunkSize = size
		}
	}

	// Concurrency
	if v := os.Getenv("OMNISTORAGE_GCS_CONCURRENCY"); v != "" {
		if c, err := strconv.Atoi(v); err == nil && c > 0 {
			config.Concurrency = c
		}
	}

	return config
}

// ConfigFromMap creates a Config from a string map.
// Supported keys:
//   - bucket: bucket name (required)
//   - project: Google Cloud project ID
//   - prefix: object prefix
//   - credentials_file: path to service account JSON key
//   - credentials_json: raw JSON credentials
//   - chunk_size: resumable upload chunk size in bytes
//   - concurrency: number of concurrent operations
func ConfigFromMap(m map[string]string) Config {
	config := DefaultConfig()

	if v, ok := m["bucket"]; ok {
		config.Bucket = v
	}
	if v, ok := m["project"]; ok {
		config.Project = v
	}
	if v, ok := m["prefix"]; ok {
		config.Prefix = v
	}
	if v, ok := m["credentials_file"]; ok {
		config.CredentialsFile = v
	}
	if v, ok := m["credentials_json"]; ok {
		config.CredentialsJSON = []byte(v)
	}
	if v, ok := m["chunk_size"]; ok {
		if size, err := strconv.Atoi(v); err == nil && size > 0 {
			config.ChunkSize = size
		}
	}
	if v, ok := m["concurrency"]; ok {
		if c, err := strconv.Atoi(v); err == nil && c > 0 {
			config.Concurrency = c
		}
	}

	return config
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Bucket == "" {
		return ErrBucketRequired
	}
	return nil
}

package gcs

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ChunkSize != 16*1024*1024 {
		t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 16*1024*1024)
	}
	if cfg.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want %d", cfg.Concurrency, 5)
	}
	if cfg.Bucket != "" {
		t.Errorf("Bucket = %q, want empty", cfg.Bucket)
	}
	if cfg.Project != "" {
		t.Errorf("Project = %q, want empty", cfg.Project)
	}
	if cfg.Prefix != "" {
		t.Errorf("Prefix = %q, want empty", cfg.Prefix)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "empty bucket",
			config:  Config{},
			wantErr: true,
		},
		{
			name: "valid with bucket",
			config: Config{
				Bucket: "my-bucket",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			config: Config{
				Bucket:          "my-bucket",
				Project:         "my-project",
				Prefix:          "prefix/",
				CredentialsFile: "/path/to/creds.json",
				ChunkSize:       8 * 1024 * 1024,
				Concurrency:     10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrBucketRequired {
				t.Errorf("Validate() error = %v, want %v", err, ErrBucketRequired)
			}
		})
	}
}

func TestConfigFromMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected Config
	}{
		{
			name:     "empty map uses defaults",
			input:    map[string]string{},
			expected: DefaultConfig(),
		},
		{
			name: "all fields",
			input: map[string]string{
				"bucket":           "test-bucket",
				"project":          "test-project",
				"prefix":           "test-prefix/",
				"credentials_file": "/path/to/creds.json",
				"credentials_json": `{"type": "service_account"}`,
				"chunk_size":       "8388608",
				"concurrency":      "10",
			},
			expected: Config{
				Bucket:          "test-bucket",
				Project:         "test-project",
				Prefix:          "test-prefix/",
				CredentialsFile: "/path/to/creds.json",
				CredentialsJSON: []byte(`{"type": "service_account"}`),
				ChunkSize:       8388608,
				Concurrency:     10,
			},
		},
		{
			name: "partial fields",
			input: map[string]string{
				"bucket":  "partial-bucket",
				"project": "partial-project",
			},
			expected: Config{
				Bucket:      "partial-bucket",
				Project:     "partial-project",
				ChunkSize:   16 * 1024 * 1024,
				Concurrency: 5,
			},
		},
		{
			name: "invalid chunk_size ignored",
			input: map[string]string{
				"bucket":     "test",
				"chunk_size": "invalid",
			},
			expected: Config{
				Bucket:      "test",
				ChunkSize:   16 * 1024 * 1024,
				Concurrency: 5,
			},
		},
		{
			name: "invalid concurrency ignored",
			input: map[string]string{
				"bucket":      "test",
				"concurrency": "invalid",
			},
			expected: Config{
				Bucket:      "test",
				ChunkSize:   16 * 1024 * 1024,
				Concurrency: 5,
			},
		},
		{
			name: "zero chunk_size uses default",
			input: map[string]string{
				"bucket":     "test",
				"chunk_size": "0",
			},
			expected: Config{
				Bucket:      "test",
				ChunkSize:   16 * 1024 * 1024,
				Concurrency: 5,
			},
		},
		{
			name: "negative concurrency uses default",
			input: map[string]string{
				"bucket":      "test",
				"concurrency": "-1",
			},
			expected: Config{
				Bucket:      "test",
				ChunkSize:   16 * 1024 * 1024,
				Concurrency: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConfigFromMap(tt.input)

			if result.Bucket != tt.expected.Bucket {
				t.Errorf("Bucket = %q, want %q", result.Bucket, tt.expected.Bucket)
			}
			if result.Project != tt.expected.Project {
				t.Errorf("Project = %q, want %q", result.Project, tt.expected.Project)
			}
			if result.Prefix != tt.expected.Prefix {
				t.Errorf("Prefix = %q, want %q", result.Prefix, tt.expected.Prefix)
			}
			if result.CredentialsFile != tt.expected.CredentialsFile {
				t.Errorf("CredentialsFile = %q, want %q", result.CredentialsFile, tt.expected.CredentialsFile)
			}
			if string(result.CredentialsJSON) != string(tt.expected.CredentialsJSON) {
				t.Errorf("CredentialsJSON = %q, want %q", result.CredentialsJSON, tt.expected.CredentialsJSON)
			}
			if result.ChunkSize != tt.expected.ChunkSize {
				t.Errorf("ChunkSize = %d, want %d", result.ChunkSize, tt.expected.ChunkSize)
			}
			if result.Concurrency != tt.expected.Concurrency {
				t.Errorf("Concurrency = %d, want %d", result.Concurrency, tt.expected.Concurrency)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Save and restore original env vars
	origBucket := os.Getenv("OMNISTORAGE_GCS_BUCKET")
	origProject := os.Getenv("OMNISTORAGE_GCS_PROJECT")
	origPrefix := os.Getenv("OMNISTORAGE_GCS_PREFIX")
	origCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	origChunk := os.Getenv("OMNISTORAGE_GCS_CHUNK_SIZE")
	origConc := os.Getenv("OMNISTORAGE_GCS_CONCURRENCY")
	origGCSBucket := os.Getenv("GCS_BUCKET")
	origGoogleProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	origGCloudProject := os.Getenv("GCLOUD_PROJECT")

	defer func() {
		os.Setenv("OMNISTORAGE_GCS_BUCKET", origBucket)
		os.Setenv("OMNISTORAGE_GCS_PROJECT", origProject)
		os.Setenv("OMNISTORAGE_GCS_PREFIX", origPrefix)
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", origCreds)
		os.Setenv("OMNISTORAGE_GCS_CHUNK_SIZE", origChunk)
		os.Setenv("OMNISTORAGE_GCS_CONCURRENCY", origConc)
		os.Setenv("GCS_BUCKET", origGCSBucket)
		os.Setenv("GOOGLE_CLOUD_PROJECT", origGoogleProject)
		os.Setenv("GCLOUD_PROJECT", origGCloudProject)
	}()

	t.Run("all env vars set", func(t *testing.T) {
		os.Setenv("OMNISTORAGE_GCS_BUCKET", "env-bucket")
		os.Setenv("OMNISTORAGE_GCS_PROJECT", "env-project")
		os.Setenv("OMNISTORAGE_GCS_PREFIX", "env-prefix/")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/env/creds.json")
		os.Setenv("OMNISTORAGE_GCS_CHUNK_SIZE", "4194304")
		os.Setenv("OMNISTORAGE_GCS_CONCURRENCY", "8")

		cfg := ConfigFromEnv()

		if cfg.Bucket != "env-bucket" {
			t.Errorf("Bucket = %q, want %q", cfg.Bucket, "env-bucket")
		}
		if cfg.Project != "env-project" {
			t.Errorf("Project = %q, want %q", cfg.Project, "env-project")
		}
		if cfg.Prefix != "env-prefix/" {
			t.Errorf("Prefix = %q, want %q", cfg.Prefix, "env-prefix/")
		}
		if cfg.CredentialsFile != "/env/creds.json" {
			t.Errorf("CredentialsFile = %q, want %q", cfg.CredentialsFile, "/env/creds.json")
		}
		if cfg.ChunkSize != 4194304 {
			t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 4194304)
		}
		if cfg.Concurrency != 8 {
			t.Errorf("Concurrency = %d, want %d", cfg.Concurrency, 8)
		}
	})

	t.Run("fallback env vars", func(t *testing.T) {
		// Clear primary vars
		os.Unsetenv("OMNISTORAGE_GCS_BUCKET")
		os.Unsetenv("OMNISTORAGE_GCS_PROJECT")

		// Set fallback vars
		os.Setenv("GCS_BUCKET", "fallback-bucket")
		os.Setenv("GOOGLE_CLOUD_PROJECT", "fallback-project")

		cfg := ConfigFromEnv()

		if cfg.Bucket != "fallback-bucket" {
			t.Errorf("Bucket = %q, want %q", cfg.Bucket, "fallback-bucket")
		}
		if cfg.Project != "fallback-project" {
			t.Errorf("Project = %q, want %q", cfg.Project, "fallback-project")
		}
	})

	t.Run("gcloud project fallback", func(t *testing.T) {
		os.Unsetenv("OMNISTORAGE_GCS_PROJECT")
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Setenv("GCLOUD_PROJECT", "gcloud-project")

		cfg := ConfigFromEnv()

		if cfg.Project != "gcloud-project" {
			t.Errorf("Project = %q, want %q", cfg.Project, "gcloud-project")
		}
	})
}

func TestErrBucketRequired(t *testing.T) {
	if ErrBucketRequired == nil {
		t.Error("ErrBucketRequired should not be nil")
	}
	if ErrBucketRequired.Error() != "gcs: bucket is required" {
		t.Errorf("ErrBucketRequired.Error() = %q, want %q", ErrBucketRequired.Error(), "gcs: bucket is required")
	}
}

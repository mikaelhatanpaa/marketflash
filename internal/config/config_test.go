package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("failed to set %s: %v", k, err)
		}
	}
}

func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file after writing: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("failed to remove temp file %s: %v", tmpFile.Name(), err)
		}
	})

	return tmpFile.Name()
}

func TestLoadConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		os.Clearenv()

		configContent := `
database_url: postgres://localhost:5432/test
port: 8080
environment: production
debug: true
api_key: test-key
`
		path := createTempConfigFile(t, configContent)

		cfg, err := LoadConfig(path)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		want := config{
			DatabaseURL: "postgres://localhost:5432/test",
			Port:        8080,
			Environment: "production",
			Debug:       true,
			APIKey:      "test-key",
		}
		if cfg != want {
			t.Errorf("expected config %+v, got: %+v", want, cfg)
		}
	})

	t.Run("empty config file", func(t *testing.T) {
		os.Clearenv()

		path := createTempConfigFile(t, "")

		_, err := LoadConfig(path)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, ErrReadConfig) {
			t.Errorf("expected error %v, got: %v", ErrReadConfig, err)
		}
	})

	t.Run("non-existent config file", func(t *testing.T) {
		os.Clearenv()

		setEnv(t, map[string]string{
			"DATABASE_URL": "postgres://localhost:5432/test",
			"PORT":         "8080",
			"ENVIRONMENT":  "production",
			"DEBUG":        "true",
			"API_KEY":      "test-key",
		})

		cfg, err := LoadConfig("nonexistent.yaml")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		want := config{
			DatabaseURL: "postgres://localhost:5432/test",
			Port:        8080,
			Environment: "production",
			Debug:       true,
			APIKey:      "test-key",
		}
		if cfg != want {
			t.Errorf("expected config %+v, got: %+v", want, cfg)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		os.Clearenv()

		path := createTempConfigFile(t, "invalid: yaml: content")

		_, err := LoadConfig(path)
		if err == nil || !errors.Is(err, ErrParseYAML) {
			t.Errorf("expected error %v for invalid YAML file %s, got: %v", ErrParseYAML, path, err)
		}
	})

	t.Run("partial config file", func(t *testing.T) {
		os.Clearenv()

		configContent := `
database_url: postgres://localhost:5432/test
port: 8080
environment: production
`
		path := createTempConfigFile(t, configContent)

		_, err := LoadConfig(path)
		if err == nil {
			t.Errorf("expected validation error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, ErrMissingAPIKey.Error()) {
			t.Errorf("expected error %v, got: %v", ErrMissingAPIKey, err)
		}
	})

	t.Run("invalid port in yaml", func(t *testing.T) {
		os.Clearenv()

		configContent := `
database_url: postgres://localhost:5432/test
port: 65536
environment: production
api_key: test-key
`
		path := createTempConfigFile(t, configContent)

		_, err := LoadConfig(path)
		if err == nil {
			t.Errorf("expected validation error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, ErrInvalidPortRange.Error()) {
			t.Errorf("expected error %v, got: %v", ErrInvalidPortRange, err)
		}
	})

	t.Run("default values with minimal env", func(t *testing.T) {
		os.Clearenv()

		setEnv(t, map[string]string{
			"DATABASE_URL": "postgres://localhost:5432/test",
			"API_KEY":      "test-key",
		})

		cfg, err := LoadConfig("")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		want := config{
			DatabaseURL: "postgres://localhost:5432/test",
			Port:        8080,
			Environment: "development",
			Debug:       false,
			APIKey:      "test-key",
		}
		if cfg != want {
			t.Errorf("expected config %+v, got: %+v", want, cfg)
		}
	})

	t.Run("file read failure", func(t *testing.T) {
		os.Clearenv()

		path := t.TempDir()

		_, err := LoadConfig(path)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, ErrReadConfig) {
			t.Errorf("expected error %v, got: %v", ErrReadConfig, err)
		}
	})

	tests := []struct {
		name    string
		env     map[string]string
		wantErr error
	}{
		{
			name: "invalid port env",
			env: map[string]string{
				"PORT":         "invalid",
				"DATABASE_URL": "postgres://localhost:5432/test",
				"ENVIRONMENT":  "production",
				"API_KEY":      "test-key",
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "invalid debug env",
			env: map[string]string{
				"DEBUG":        "invalid",
				"DATABASE_URL": "postgres://localhost:5432/test",
				"PORT":         "8080",
				"ENVIRONMENT":  "production",
				"API_KEY":      "test-key",
			},
			wantErr: ErrInvalidDebug,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			setEnv(t, tt.env)

			_, err := LoadConfig("")
			if err == nil || !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}

	t.Run("missing required fields", func(t *testing.T) {
		os.Clearenv()

		_, err := LoadConfig("")
		if err == nil {
			t.Errorf("expected validation error, got nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, ErrMissingDatabaseURL.Error()) {
			t.Errorf("expected error %v, got: %v", ErrMissingDatabaseURL, err)
		}
		if !strings.Contains(errStr, ErrMissingAPIKey.Error()) {
			t.Errorf("expected error %v, got: %v", ErrMissingAPIKey, err)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   config
		wantErrs []error
	}{
		{
			name: "valid config",
			config: config{
				DatabaseURL: "postgres://localhost:5432/test",
				Port:        8080,
				Environment: "production",
				Debug:       true,
				APIKey:      "test-key",
			},
			wantErrs: nil,
		},
		{
			name: "missing database_url",
			config: config{
				Port:        8080,
				Environment: "production",
				APIKey:      "test-key",
			},
			wantErrs: []error{ErrMissingDatabaseURL},
		},
		{
			name: "missing api_key",
			config: config{
				DatabaseURL: "postgres://localhost:5432/test",
				Port:        8080,
				Environment: "production",
			},
			wantErrs: []error{ErrMissingAPIKey},
		},
		{
			name: "invalid port",
			config: config{
				DatabaseURL: "postgres://localhost:5432/test",
				Port:        0,
				Environment: "production",
				APIKey:      "test-key",
			},
			wantErrs: []error{ErrInvalidPortRange},
		},
		{
			name: "invalid environment",
			config: config{
				DatabaseURL: "postgres://localhost:5432/test",
				Port:        8080,
				Environment: "invalid",
				APIKey:      "test-key",
			},
			wantErrs: []error{ErrInvalidEnvironment},
		},
		{
			name: "missing database_url and invalid port",
			config: config{
				Environment: "production",
				APIKey:      "test-key",
			},
			wantErrs: []error{ErrMissingDatabaseURL, ErrInvalidPortRange},
		},
		{
			name: "invalid environment and missing api_key",
			config: config{
				DatabaseURL: "postgres://localhost:5432/test",
				Port:        8080,
				Environment: "invalid",
			},
			wantErrs: []error{ErrInvalidEnvironment, ErrMissingAPIKey},
		},
		{
			name: "multiple errors",
			config: config{
				Port: 0,
			},
			wantErrs: []error{
				ErrMissingDatabaseURL,
				ErrInvalidPortRange,
				ErrMissingAPIKey,
				ErrInvalidEnvironment,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected validation error, got nil")
				return
			}

			errStr := err.Error()
			for _, wantErr := range tt.wantErrs {
				if !strings.Contains(errStr, wantErr.Error()) {
					t.Errorf("expected error %v, got: %v", wantErr, err)
				}
			}

			if len(tt.wantErrs) > 1 {
				errorCount := strings.Count(errStr, ";") + 1
				if errorCount != len(tt.wantErrs) {
					t.Errorf("expected %d errors, got %d: %v", len(tt.wantErrs), errorCount, err)
				}
			}
		})
	}
}

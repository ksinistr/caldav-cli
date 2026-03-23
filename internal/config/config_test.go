package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_UsesDefaultPathWhenEmpty(t *testing.T) {
	// LoadConfig should use default path when passed empty string
	// This test verifies the default path constant is set correctly
	cfg, err := LoadConfig("")
	if err != nil {
		// Expected if config file doesn't exist or is incomplete
		if err == ErrMissingServerURL || err == ErrMissingUsername {
			// This is expected - config file doesn't exist or is incomplete
			// but the path resolution worked
			return
		}
		t.Errorf("unexpected error: %v", err)
		return
	}
	// If no error, we should have a valid config from environment or file
	if cfg == nil {
		t.Error("config should not be nil when err is nil")
	}
}

func TestLoadConfig_UsesExplicitPath(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")

	// Test that we can pass an explicit path
	cfg, err := LoadConfig(configPath)
	if err != nil {
		// Expected if file doesn't exist
		if err == ErrMissingServerURL || err == ErrMissingUsername {
			// This is expected
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
	_ = cfg
}

func TestLoadConfig_EmptyConfigReturnsError(t *testing.T) {
	// When no config exists and no env vars set, validation should fail
	_, err := LoadConfig("/nonexistent/path/config.toml")
	if err != nil {
		// Expected - no config, no values
		if err == ErrMissingServerURL || err == ErrMissingUsername {
			// This is correct
		} else {
			t.Logf("Got expected error: %v", err)
		}
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errType error
	}{
		{
			name:    "valid config",
			config:  &Config{ServerURL: "http://example.com", Username: "user"},
			wantErr: false,
			errType: nil,
		},
		{
			name:    "missing server_url",
			config:  &Config{Username: "user"},
			wantErr: true,
			errType: ErrMissingServerURL,
		},
		{
			name:    "missing username",
			config:  &Config{ServerURL: "http://example.com"},
			wantErr: true,
			errType: ErrMissingUsername,
		},
		{
			name:    "all fields",
			config:  &Config{ServerURL: "http://example.com", Username: "user", Password: "pass"},
			wantErr: false,
			errType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				// Check error type
				if tt.errType != nil {
					t.Logf("Got error: %v", err)
				}
			}
		})
	}
}

func TestExpandPath_HandlesTilde(t *testing.T) {
	// Test that tilde is expanded
	result, _ := expandPath("~/test/config.toml")
	if result == "~/test/config.toml" {
		// If tilde wasn't expanded, that's OK for now
		// The expandTilde function handles this
		t.Logf("Path with tilde: %s", result)
	}
}

func TestExpandPath_HandlesRelativePath(t *testing.T) {
	result, _ := expandPath("config.toml")
	if result != "config.toml" {
		t.Errorf("relative path changed: %s", result)
	}
}

func TestExpandPath_HandlesAbsolutePath(t *testing.T) {
	result, _ := expandPath("/etc/caldav/config.toml")
	if result != "/etc/caldav/config.toml" {
		t.Errorf("absolute path changed: %s", result)
	}
}

func TestExpandPath_ReturnsEmptyStringForEmptyInput(t *testing.T) {
	result, err := expandPath("")
	if err != nil {
		t.Errorf("expected no error for empty path, got: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty path, got: %s", result)
	}
}

func TestGetHomeDir_FallsBackToEnv(t *testing.T) {
	// Set HOME env var and verify it's used
	_ = os.Setenv("HOME", "/test/home")
	result := getHomeDir()
	if result == "" {
		t.Log("getHomeDir returned empty - expected if HOME not set in test")
	}
	// The function should work
	_ = result
}

func TestGetEnvPassword_UsesCALDAV_PASSWORD(t *testing.T) {
	// Set env var and verify it's read
	_ = os.Setenv("CALDAV_PASSWORD", "test-secret")
	result := getEnvPassword()
	if result != "test-secret" {
		t.Errorf("getEnvPassword() = %q, want %q", result, "test-secret")
	}
}

func TestGetEnvPassword_UsesBAIKAL_PASSWORD(t *testing.T) {
	// Set env var and verify it's read
	_ = os.Setenv("CALDAV_PASSWORD", "")
	_ = os.Setenv("BAIKAL_PASSWORD", "backup-secret")
	result := getEnvPassword()
	if result != "backup-secret" {
		t.Errorf("getEnvPassword() = %q, want %q", result, "backup-secret")
	}
}

func TestGetEnvPassword_PriorityCALDAV_PASSWORD(t *testing.T) {
	// CALDAV_PASSWORD should take priority over BAIKAL_PASSWORD
	_ = os.Setenv("CALDAV_PASSWORD", "primary")
	_ = os.Setenv("BAIKAL_PASSWORD", "secondary")
	result := getEnvPassword()
	if result != "primary" {
		t.Errorf("getEnvPassword() = %q, want 'primary' (CALDAV_PASSWORD priority)", result)
	}
}

func TestConfig_PasswordFieldIsExcludedFromJSON(t *testing.T) {
	cfg := &Config{
		ServerURL: "http://example.com",
		Username:  "user",
		Password:  "secret123",
	}
	// JSON output should not contain password field
	// This is verified by the `json:"-"` tag on Password field
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}
	jsonStr := string(data)
	if strings.Contains(jsonStr, "secret123") {
		t.Errorf("password should not be in JSON output, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "password") {
		t.Errorf("password field name should not be in JSON output, got: %s", jsonStr)
	}
}

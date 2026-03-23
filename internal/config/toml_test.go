package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_LoadsValidTOML(t *testing.T) {
	// Clear any environment variables that might interfere
	_ = os.Unsetenv("CALDAV_PASSWORD")
	_ = os.Unsetenv("BAIKAL_PASSWORD")

	// Create a valid TOML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
insecure_skip_verify = false
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ServerURL != "https://baikal.example.com/dav.php/" {
		t.Errorf("ServerURL = %q, want %q", cfg.ServerURL, "https://baikal.example.com/dav.php/")
	}
	if cfg.Username != "alice" {
		t.Errorf("Username = %q, want %q", cfg.Username, "alice")
	}
	// Empty password in file is expected
	// Password should be empty since no env var is set
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty string", cfg.Password)
	}
	if cfg.InsecureSkipVerify != false {
		t.Errorf("InsecureSkipVerify = %v, want false", cfg.InsecureSkipVerify)
	}
}

func TestLoadConfig_PrioritizesEnvPasswordOverFile(t *testing.T) {
	// Create config with empty password
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set CALDAV_PASSWORD env var
	_ = os.Setenv("CALDAV_PASSWORD", "env-secret-123")
	defer func() { _ = os.Unsetenv("CALDAV_PASSWORD") }()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Password != "env-secret-123" {
		t.Errorf("Password = %q, want 'env-secret-123' (from env)", cfg.Password)
	}
}

func TestLoadConfig_EnvPasswordOverridesNonEmptyFilePassword(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = "file-password-should-be-overridden"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_ = os.Setenv("CALDAV_PASSWORD", "env-password-wins")
	defer func() { _ = os.Unsetenv("CALDAV_PASSWORD") }()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Password != "env-password-wins" {
		t.Errorf("Password = %q, want 'env-password-wins' (env should override file)", cfg.Password)
	}
}

func TestLoadConfig_CALDAV_PASSWORD_HasPriorityOverBAIKAL_PASSWORD(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_ = os.Setenv("CALDAV_PASSWORD", "caldao-priority")
	_ = os.Setenv("BAIKAL_PASSWORD", "baikal-fallback")
	defer func() { _ = os.Unsetenv("CALDAV_PASSWORD") }()
	defer func() { _ = os.Unsetenv("BAIKAL_PASSWORD") }()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Password != "caldao-priority" {
		t.Errorf("Password = %q, want 'caldao-priority' (CALDAV_PASSWORD priority)", cfg.Password)
	}
}

func TestLoadConfig_UsesBAIKAL_PASSWORDAsFallback(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = ""
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Only set BAIKAL_PASSWORD
	_ = os.Setenv("CALDAV_PASSWORD", "")
	_ = os.Setenv("BAIKAL_PASSWORD", "baikal-backup")
	defer func() { _ = os.Unsetenv("CALDAV_PASSWORD") }()
	defer func() { _ = os.Unsetenv("BAIKAL_PASSWORD") }()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Password != "baikal-backup" {
		t.Errorf("Password = %q, want 'baikal-backup' (BAIKAL_PASSWORD fallback)", cfg.Password)
	}
}

func TestLoadConfig_UsesFilePasswordWhenNoEnvSet(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = "file-password-123"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Ensure no env vars are set
	_ = os.Unsetenv("CALDAV_PASSWORD")
	_ = os.Unsetenv("BAIKAL_PASSWORD")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Password != "file-password-123" {
		t.Errorf("Password = %q, want 'file-password-123' (from file)", cfg.Password)
	}
}

func TestLoadConfig_MissingServerURLErrors(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `username = "alice"
password = "pass"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err != ErrMissingServerURL {
		t.Errorf("LoadConfig() error = %v, want %v", err, ErrMissingServerURL)
	}
}

func TestLoadConfig_MissingUsernameErrors(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://example.com"
password = "pass"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err != ErrMissingUsername {
		t.Errorf("LoadConfig() error = %v, want %v", err, ErrMissingUsername)
	}
}

func TestLoadConfig_AllTLSOptions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	content := `server_url = "https://baikal.example.com/dav.php/"
username = "alice"
password = "pass"
insecure_skip_verify = true
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.InsecureSkipVerify != true {
		t.Errorf("InsecureSkipVerify = %v, want true", cfg.InsecureSkipVerify)
	}
}

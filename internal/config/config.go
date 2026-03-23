package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jinzhu/configor"
)

const (
	defaultConfigPath = "~/.config/caldav-cli/config.toml"
)

// Errors for config validation
var (
	ErrMissingServerURL = errors.New("server_url is required")
	ErrMissingUsername  = errors.New("username is required")
)

// Config holds the CalDAV server configuration for a single profile.
type Config struct {
	ServerURL string `toml:"server_url" yaml:"server_url" json:"server_url" mapstructure:"server_url"`
	Username  string `toml:"username" yaml:"username" json:"username" mapstructure:"username"`
	Password  string `toml:"password,omitempty" yaml:"password,omitempty" json:"-" mapstructure:"password"`
	// Optional TLS settings for self-signed certificates
	InsecureSkipVerify bool `toml:"insecure_skip_verify" yaml:"insecure_skip_verify" json:"insecure_skip_verify" mapstructure:"insecure_skip_verify"`
}

// LoadConfig loads configuration from the default path or explicit path.
// It also falls back to environment variables for sensitive fields.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = defaultConfigPath
	}

	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = configor.Load(cfg, expandedPath)
	if err != nil {
		// Check if error indicates file not found
		// configor returns a nil error if file doesn't exist, but we need to handle this case
		// For simplicity, we assume missing file is OK - validation will catch required fields
		// Try to check error type using string matching
		errStr := err.Error()
		if !isNotFoundError(errStr) {
			return nil, err
		}
	}

	// Environment variables always take priority for password
	if pwd := getEnvPassword(); pwd != "" {
		cfg.Password = pwd
	}

	// Validate loaded config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// isNotFoundError checks if an error indicates file not found
func isNotFoundError(errStr string) bool {
	return containsAny(errStr, []string{
		"no such file or directory",
		"file not found",
		"cannot find",
		"doesn't exist",
	})
}

func containsAny(str string, substrs []string) bool {
	for _, substr := range substrs {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// Validate checks that all required configuration fields are present.
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return ErrMissingServerURL
	}
	if c.Username == "" {
		return ErrMissingUsername
	}
	// Password is optional if provided via env - we'll check during actual auth
	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	return expandTilde(path), nil
}

func expandTilde(path string) string {
	if len(path) >= 2 && path[0] == '~' && (len(path) == 2 || path[1] == '/' || path[1] == '\\') {
		home := getHomeDir()
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	// Fallback to user.HomeDir
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

func getEnvPassword() string {
	// Try both CALDAV_PASSWORD and BAIKAL_PASSWORD
	if pwd := os.Getenv("CALDAV_PASSWORD"); pwd != "" {
		return pwd
	}
	if pwd := os.Getenv("BAIKAL_PASSWORD"); pwd != "" {
		return pwd
	}
	return ""
}

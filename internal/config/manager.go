package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
// It supports multiple configuration sources with environment variables
// taking precedence over file-based configuration.
type Config struct {
	LogLevel        string                       `yaml:"log_level"`        // Logging verbosity: debug, info, warn, error
	LogFile         string                       `yaml:"log_file,omitempty"` // Optional log file path
	PollingInterval string                       `yaml:"polling_interval"`   // How often to check for updates (e.g., "60s")
	Linear          LinearConfig                 `yaml:"linear,omitempty"`   // Global Linear OAuth credentials
	Workspaces      map[string]WorkspaceConfig   `yaml:"workspaces,omitempty"` // Per-workspace configurations
}

// LinearConfig holds Linear OAuth credentials
type LinearConfig struct {
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
	Port         int    `yaml:"port,omitempty"` // OAuth callback port (default: 37412)
}

// WorkspaceConfig represents a Linear workspace configuration
type WorkspaceConfig struct {
	Name               string `yaml:"name"`
	LinearClientID     string `yaml:"linear_client_id"`
	LinearClientSecret string `yaml:"linear_client_secret"`
	Description        string `yaml:"description,omitempty"`
}

// Manager handles configuration file operations with support for:
// - YAML file persistence
// - Environment variable overrides
// - Secure file permissions
// - Default value initialization
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	// If no path provided, use default (XDG standard)
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory
			configPath = ".config/linear/config.yaml"
		} else {
			configPath = filepath.Join(homeDir, ".config", "linear", "config.yaml")
		}
	}

	return &Manager{
		configPath: configPath,
	}
}

// Load reads the configuration from file with environment variable overrides.
//
// Configuration precedence (highest to lowest):
// 1. Environment variables (LINEAR_CLIENT_ID, LINEAR_CLIENT_SECRET, etc.)
// 2. Configuration file (~/.config/linear/config.yaml)
// 3. Default values
//
// Why this order: Environment variables allow secure credential injection
// in CI/CD and containerized environments without modifying config files.
func (m *Manager) Load() (*Config, error) {
	// Create default config
	// Why these defaults: "info" provides useful output without being verbose,
	// 60s polling is frequent enough for updates without overloading the API
	cfg := &Config{
		LogLevel:        "info",
		PollingInterval: "60s",
		Workspaces:      make(map[string]WorkspaceConfig),
	}
	
	// Check if config file exists with proper error handling
	exists, err := m.ConfigExistsWithError()
	if err != nil {
		return nil, fmt.Errorf("failed to check config file existence: %w", err)
	}
	if !exists {
		// File doesn't exist, return defaults
		return cfg, nil
	}
	
	// Read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Apply environment variable overrides
	m.applyEnvironmentOverrides(cfg)
	
	return cfg, nil
}

// Save writes the configuration to file with secure permissions.
//
// Security considerations:
// - Directory created with 0700 (owner access only)
// - Config file created with 0600 (owner read/write only)
// - Credentials are stored in the config file
//
// Why YAML: Human-readable format that's easy to edit manually,
// supports comments, and handles complex nested structures well.
func (m *Manager) Save(cfg *Config) error {
	// Create directory if it doesn't exist
	// 0700 ensures only the owner can access the config directory
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file with secure permissions
	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// ConfigExistsWithError checks if the config file exists and returns detailed error information
// Returns (false, nil) if file doesn't exist - this is not an error condition
// Returns (false, error) if there's an actual error like permission denied
func (m *Manager) ConfigExistsWithError() (bool, error) {
	_, err := os.Stat(m.configPath)
	if err == nil {
		return true, nil
	}
	
	if os.IsNotExist(err) {
		return false, nil // File not existing is not an error
	}
	
	// Other errors (permission denied, etc.) should be reported
	return false, fmt.Errorf("failed to check config file: %w", err)
}

// applyEnvironmentOverrides applies environment variable overrides to the config
func (m *Manager) applyEnvironmentOverrides(cfg *Config) {
	// Override log level
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	
	// Override log file
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		cfg.LogFile = logFile
	}
	
	// Override polling interval
	if interval := os.Getenv("POLLING_INTERVAL"); interval != "" {
		cfg.PollingInterval = interval
	}
	
	// Override Linear credentials
	if clientID := os.Getenv("LINEAR_CLIENT_ID"); clientID != "" {
		cfg.Linear.ClientID = clientID
	}
	if clientSecret := os.Getenv("LINEAR_CLIENT_SECRET"); clientSecret != "" {
		cfg.Linear.ClientSecret = clientSecret
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log level: %s (must be one of: debug, info, warn, error)", c.LogLevel)
	}
	
	// Validate polling interval
	if _, err := time.ParseDuration(c.PollingInterval); err != nil {
		return fmt.Errorf("invalid polling interval: %s", c.PollingInterval)
	}
	
	// Validate workspaces - only check if credentials are provided
	for name, ws := range c.Workspaces {
		if ws.LinearClientID == "" || ws.LinearClientSecret == "" {
			return fmt.Errorf("workspace '%s' missing Linear credentials", name)
		}
	}
	
	// Note: We allow empty global Linear credentials as users might only use workspace-specific ones
	
	return nil
}

// GetWorkspace returns a workspace configuration by name
func (c *Config) GetWorkspace(name string) (WorkspaceConfig, bool) {
	ws, exists := c.Workspaces[name]
	return ws, exists
}

// SetWorkspace adds or updates a workspace configuration
func (c *Config) SetWorkspace(name string, ws WorkspaceConfig) {
	if c.Workspaces == nil {
		c.Workspaces = make(map[string]WorkspaceConfig)
	}
	c.Workspaces[name] = ws
}

// DeleteWorkspace removes a workspace configuration
func (c *Config) DeleteWorkspace(name string) {
	delete(c.Workspaces, name)
}

// GetLinearCredentials returns the Linear credentials to use
// It checks workspace-specific credentials first, then falls back to global
func (c *Config) GetLinearCredentials(workspace string) (clientID, clientSecret string) {
	// Check if workspace is specified and exists
	if workspace != "" {
		if ws, exists := c.GetWorkspace(workspace); exists {
			return ws.LinearClientID, ws.LinearClientSecret
		}
	}
	
	// Fall back to global credentials
	return c.Linear.ClientID, c.Linear.ClientSecret
}
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager(t *testing.T) {
	t.Run("Load default config when no file exists", func(t *testing.T) {
		// Create temp directory for test
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Create manager
		manager := NewManager(configPath)
		
		// Load config (should create defaults)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		
		// Check defaults
		if cfg.LogLevel != "info" {
			t.Errorf("expected default log level 'info', got %s", cfg.LogLevel)
		}
		if cfg.PollingInterval != "60s" {
			t.Errorf("expected default polling interval '60s', got %s", cfg.PollingInterval)
		}
	})
	
	t.Run("Load existing config file", func(t *testing.T) {
		// Create temp directory and config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Write test config
		configContent := `log_level: debug
polling_interval: 30s
linear:
  client_id: test-client-id
  client_secret: test-client-secret
workspaces:
  default:
    name: Default Workspace
    linear_client_id: workspace-client-id
    linear_client_secret: workspace-client-secret
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		if err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}
		
		// Create manager and load
		manager := NewManager(configPath)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		
		// Verify loaded values
		if cfg.LogLevel != "debug" {
			t.Errorf("expected log level 'debug', got %s", cfg.LogLevel)
		}
		if cfg.PollingInterval != "30s" {
			t.Errorf("expected polling interval '30s', got %s", cfg.PollingInterval)
		}
		if cfg.Linear.ClientID != "test-client-id" {
			t.Errorf("expected client ID 'test-client-id', got %s", cfg.Linear.ClientID)
		}
		if len(cfg.Workspaces) != 1 {
			t.Errorf("expected 1 workspace, got %d", len(cfg.Workspaces))
		}
	})
	
	t.Run("Save config creates directory if needed", func(t *testing.T) {
		// Create temp directory
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".linear")
		configPath := filepath.Join(configDir, "config.yaml")
		
		// Create manager
		manager := NewManager(configPath)
		
		// Create config to save
		cfg := &Config{
			LogLevel:        "warn",
			PollingInterval: "45s",
			Linear: LinearConfig{
				ClientID:     "save-test-id",
				ClientSecret: "save-test-secret",
			},
		}
		
		// Save config
		err := manager.Save(cfg)
		if err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		
		// Verify directory was created
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			t.Error("config directory was not created")
		}
		
		// Verify file was created with correct permissions
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("config file was not created: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("expected permissions 0600, got %v", info.Mode().Perm())
		}
		
		// Load and verify saved content
		loadedCfg, err := manager.Load()
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}
		if loadedCfg.LogLevel != "warn" {
			t.Errorf("expected log level 'warn', got %s", loadedCfg.LogLevel)
		}
		if loadedCfg.Linear.ClientID != "save-test-id" {
			t.Errorf("expected client ID 'save-test-id', got %s", loadedCfg.Linear.ClientID)
		}
	})
	
	t.Run("Environment variables override config file", func(t *testing.T) {
		// Create temp directory and config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Write test config
		configContent := `log_level: info
linear:
  client_id: file-client-id
  client_secret: file-client-secret
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		if err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}
		
		// Set environment variables
		os.Setenv("LINEAR_CLIENT_ID", "env-client-id")
		os.Setenv("LINEAR_CLIENT_SECRET", "env-client-secret")
		os.Setenv("LOG_LEVEL", "debug")
		defer func() {
			os.Unsetenv("LINEAR_CLIENT_ID")
			os.Unsetenv("LINEAR_CLIENT_SECRET")
			os.Unsetenv("LOG_LEVEL")
		}()
		
		// Create manager and load
		manager := NewManager(configPath)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		
		// Verify environment overrides
		if cfg.LogLevel != "debug" {
			t.Errorf("expected log level 'debug' from env, got %s", cfg.LogLevel)
		}
		if cfg.Linear.ClientID != "env-client-id" {
			t.Errorf("expected client ID 'env-client-id' from env, got %s", cfg.Linear.ClientID)
		}
		if cfg.Linear.ClientSecret != "env-client-secret" {
			t.Errorf("expected client secret 'env-client-secret' from env, got %s", cfg.Linear.ClientSecret)
		}
	})
	
	t.Run("Get workspace by name", func(t *testing.T) {
		// Create temp directory and config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Create manager
		manager := NewManager(configPath)
		
		// Create config with workspaces
		cfg := &Config{
			Workspaces: map[string]WorkspaceConfig{
				"dev": {
					Name:               "Development",
					LinearClientID:     "dev-client-id",
					LinearClientSecret: "dev-client-secret",
				},
				"prod": {
					Name:               "Production",
					LinearClientID:     "prod-client-id",
					LinearClientSecret: "prod-client-secret",
				},
			},
		}
		
		// Save config
		err := manager.Save(cfg)
		if err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		
		// Load and get workspace
		loadedCfg, err := manager.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		
		// Get existing workspace
		ws, exists := loadedCfg.GetWorkspace("dev")
		if !exists {
			t.Error("expected workspace 'dev' to exist")
		}
		if ws.LinearClientID != "dev-client-id" {
			t.Errorf("expected client ID 'dev-client-id', got %s", ws.LinearClientID)
		}
		
		// Get non-existent workspace
		_, exists = loadedCfg.GetWorkspace("nonexistent")
		if exists {
			t.Error("expected workspace 'nonexistent' to not exist")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("Validate valid config", func(t *testing.T) {
		cfg := &Config{
			LogLevel:        "info",
			PollingInterval: "60s",
			Linear: LinearConfig{
				ClientID:     "valid-id",
				ClientSecret: "valid-secret",
			},
		}
		
		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected valid config, got error: %v", err)
		}
	})
	
	t.Run("Validate invalid log level", func(t *testing.T) {
		cfg := &Config{
			LogLevel:        "invalid",
			PollingInterval: "60s",
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("expected validation error for invalid log level")
		}
	})
	
	t.Run("Validate invalid polling interval", func(t *testing.T) {
		cfg := &Config{
			LogLevel:        "info",
			PollingInterval: "invalid",
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("expected validation error for invalid polling interval")
		}
	})
	
	t.Run("Validate workspace with missing credentials", func(t *testing.T) {
		cfg := &Config{
			LogLevel:        "info",
			PollingInterval: "60s",
			Workspaces: map[string]WorkspaceConfig{
				"test": {
					Name:           "Test",
					LinearClientID: "", // Missing
				},
			},
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("expected validation error for missing workspace credentials")
		}
	})
}
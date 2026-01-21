package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigManagerErrorHandling tests proper error handling in config manager
func TestConfigManagerErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupPath   func() string
		expectError bool
		errorType   string // "permission" or "other"
	}{
		{
			name: "config file does not exist",
			setupPath: func() string {
				return filepath.Join(t.TempDir(), "nonexistent", "config.yaml")
			},
			expectError: false, // Should return default config
		},
		{
			name: "config file exists",
			setupPath: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `log_level: "debug"
polling_interval: "30s"`
				err := os.WriteFile(configPath, []byte(configContent), 0600)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				return configPath
			},
			expectError: false,
		},
		{
			name: "permission denied to config directory",
			setupPath: func() string {
				tmpDir := t.TempDir()
				restrictedDir := filepath.Join(tmpDir, "restricted")
				err := os.Mkdir(restrictedDir, 0000) // No permissions
				if err != nil {
					t.Fatalf("Failed to create restricted directory: %v", err)
				}
				
				// Clean up permissions after test
				t.Cleanup(func() {
					os.Chmod(restrictedDir, 0755)
				})
				
				return filepath.Join(restrictedDir, "config.yaml")
			},
			expectError: true,
			errorType:   "permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupPath()
			manager := NewManager(configPath)
			
			cfg, err := manager.Load()
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if !tt.expectError && cfg == nil {
				t.Error("Expected config to be returned but got nil")
			}
			
			// For permission errors, we should get a meaningful error message
			if tt.errorType == "permission" && err != nil {
				if !contains(err.Error(), "permission denied") {
					t.Errorf("Expected permission denied error, got: %v", err)
				}
			}
		})
	}
}

// TestConfigManagerFileExistsWithError tests an improved version that handles errors properly
func TestConfigManagerFileExistsWithError(t *testing.T) {
	tests := []struct {
		name        string
		setupPath   func() string
		expectError bool
	}{
		{
			name: "file does not exist",
			setupPath: func() string {
				return filepath.Join(t.TempDir(), "nonexistent", "config.yaml")
			},
			expectError: false, // File not existing is not an error
		},
		{
			name: "file exists",
			setupPath: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				err := os.WriteFile(configPath, []byte("test: value"), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupPath()
			manager := NewManager(configPath)
			
			// Test the improved method (will be implemented)
			exists, err := manager.ConfigExistsWithError()
			
			if (err != nil) != tt.expectError {
				t.Errorf("ConfigExistsWithError() error = %v, expected error: %v", err, tt.expectError)
			}
			
			// Verify the exists flag makes sense
			if !tt.expectError {
				expectedExists := tt.name == "file exists"
				if exists != expectedExists {
					t.Errorf("ConfigExistsWithError() exists = %v, expected %v", exists, expectedExists)
				}
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
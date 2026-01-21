package token

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTokenExistsErrorHandling tests proper error handling in TokenExists method
func TestTokenExistsErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		setupPath      func() string
		expectedExists bool
		shouldLogError bool
	}{
		{
			name: "file does not exist",
			setupPath: func() string {
				return "/tmp/nonexistent/path/token"
			},
			expectedExists: false,
			shouldLogError: false,
		},
		{
			name: "file exists",
			setupPath: func() string {
				tmpDir := t.TempDir()
				tokenPath := filepath.Join(tmpDir, "token")
				err := os.WriteFile(tokenPath, []byte("test-token"), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return tokenPath
			},
			expectedExists: true,
			shouldLogError: false,
		},
		{
			name: "permission denied directory",
			setupPath: func() string {
				// Create a directory with no read permissions
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
				
				return filepath.Join(restrictedDir, "token")
			},
			expectedExists: false,
			shouldLogError: true, // This should log an error since it's not just "file not found"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPath := tt.setupPath()
			storage := NewStorage(tokenPath)
			
			exists := storage.TokenExists()
			
			if exists != tt.expectedExists {
				t.Errorf("TokenExists() = %v, expected %v", exists, tt.expectedExists)
			}
		})
	}
}

// TestTokenExistsWithErrorReturn tests an improved version that returns error information
func TestTokenExistsWithErrorReturn(t *testing.T) {
	tests := []struct {
		name           string
		setupPath      func() string
		expectedExists bool
		expectError    bool
	}{
		{
			name: "file does not exist",
			setupPath: func() string {
				return "/tmp/nonexistent/path/token"
			},
			expectedExists: false,
			expectError:    false, // File not existing is not an error condition
		},
		{
			name: "file exists",
			setupPath: func() string {
				tmpDir := t.TempDir()
				tokenPath := filepath.Join(tmpDir, "token")
				err := os.WriteFile(tokenPath, []byte("test-token"), 0600)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return tokenPath
			},
			expectedExists: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPath := tt.setupPath()
			storage := NewStorage(tokenPath)
			
			// Test the improved method (will be implemented)
			exists, err := storage.TokenExistsWithError()
			
			if exists != tt.expectedExists {
				t.Errorf("TokenExistsWithError() exists = %v, expected %v", exists, tt.expectedExists)
			}
			
			if (err != nil) != tt.expectError {
				t.Errorf("TokenExistsWithError() error = %v, expected error: %v", err, tt.expectError)
			}
		})
	}
}
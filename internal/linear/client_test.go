package linear

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joa23/linear-cli/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientWithTokenPath_PreservesAuthMode(t *testing.T) {
	tests := []struct {
		name             string
		tokenData        token.TokenData
		expectedAuthMode string
	}{
		{
			name: "agent auth mode preserved",
			tokenData: token.TokenData{
				AccessToken: "lin_api_test_token_agent",
				TokenType:   "Bearer",
				AuthMode:    "agent",
			},
			expectedAuthMode: "agent",
		},
		{
			name: "user auth mode preserved",
			tokenData: token.TokenData{
				AccessToken: "lin_api_test_token_user",
				TokenType:   "Bearer",
				AuthMode:    "user",
			},
			expectedAuthMode: "user",
		},
		{
			name: "empty auth mode for legacy tokens",
			tokenData: token.TokenData{
				AccessToken: "lin_api_test_token_legacy",
				TokenType:   "Bearer",
			},
			expectedAuthMode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tokenPath := filepath.Join(tempDir, "token")

			// Write token data as JSON
			data, err := json.Marshal(tt.tokenData)
			require.NoError(t, err)
			err = os.WriteFile(tokenPath, data, 0600)
			require.NoError(t, err)

			client := NewClientWithTokenPath(tokenPath)
			require.NotNil(t, client)
			assert.Equal(t, tt.expectedAuthMode, client.GetAuthMode())
		})
	}
}

func TestNewClientWithTokenPath_ReturnsNilWhenNoToken(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	client := NewClientWithTokenPath(tokenPath)
	assert.Nil(t, client)
}

func TestNewClientWithTokenPath_FallsBackToEnvVar(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	// Set env var
	t.Setenv("LINEAR_API_KEY", "lin_api_env_key_123")

	client := NewClientWithTokenPath(tokenPath)
	require.NotNil(t, client)
	assert.Equal(t, "lin_api_env_key_123", client.GetAPIToken())
	assert.Equal(t, "", client.GetAuthMode()) // env var tokens have no auth mode
}

func TestNewClientWithTokenPath_EnvTokenNotSupported(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	t.Setenv("LINEAR_API_KEY", "")
	t.Setenv("LINEAR_API_TOKEN", "lin_api_env_token_legacy")

	client := NewClientWithTokenPath(tokenPath)
	assert.Nil(t, client)
}

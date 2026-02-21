package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joa23/linear-cli/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmdExists(t *testing.T) {
	cmd := NewRootCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "linear", cmd.Use)
}

func TestRootCmdHasSubcommands(t *testing.T) {
	cmd := NewRootCmd()

	// Only test commands that are actually registered
	expectedCommands := map[string]bool{
		"onboard": false,
		"auth":    false,
		"issues":  false,
	}

	for _, subCmd := range cmd.Commands() {
		if _, exists := expectedCommands[subCmd.Name()]; exists {
			expectedCommands[subCmd.Name()] = true
		}
	}

	for cmdName, found := range expectedCommands {
		assert.True(t, found, "Expected command %q to be registered", cmdName)
	}
}

func TestRootCmdGlobalFlags(t *testing.T) {
	cmd := NewRootCmd()

	// Check for --verbose flag
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)
}

func TestAuthSubcommands(t *testing.T) {
	cmd := NewRootCmd()
	authCmd, _, _ := cmd.Find([]string{"auth"})

	require.NotNil(t, authCmd)

	expectedSubCmds := []string{"login", "logout", "status"}
	for _, subCmdName := range expectedSubCmds {
		found := false
		for _, c := range authCmd.Commands() {
			if c.Name() == subCmdName {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected auth subcommand %q", subCmdName)
	}
}

func TestIssuesSubcommands(t *testing.T) {
	cmd := NewRootCmd()
	issuesCmd, _, _ := cmd.Find([]string{"issues"})

	require.NotNil(t, issuesCmd)

	// Only test subcommands that are actually implemented
	expectedSubCmds := []string{"list", "get", "dependencies", "blocked-by", "blocking"}
	for _, subCmdName := range expectedSubCmds {
		found := false
		for _, c := range issuesCmd.Commands() {
			if c.Name() == subCmdName {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected issues subcommand %q", subCmdName)
	}
}

func TestIssuesListCommand(t *testing.T) {
	cmd := NewRootCmd()
	issuesListCmd, _, _ := cmd.Find([]string{"issues", "list"})

	require.NotNil(t, issuesListCmd)
	assert.Equal(t, "list", issuesListCmd.Name())
	assert.Contains(t, issuesListCmd.Short, "List")
}

func TestIssuesGetCommand(t *testing.T) {
	cmd := NewRootCmd()
	issuesGetCmd, _, _ := cmd.Find([]string{"issues", "get"})

	require.NotNil(t, issuesGetCmd)
	assert.Equal(t, "get <issue-id>", issuesGetCmd.Use)
}

func TestOnboardCommand(t *testing.T) {
	cmd := NewRootCmd()
	onboardCmd, _, _ := cmd.Find([]string{"onboard"})

	require.NotNil(t, onboardCmd)
	assert.Equal(t, "onboard", onboardCmd.Name())
	assert.Contains(t, onboardCmd.Short, "setup status")
}

func TestInitializeClientWithTokenPath_NoTokenFile(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	client, err := initializeClientWithTokenPath(tokenPath)
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestInitializeClientWithTokenPath_StaticToken(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")

	// Write a token without refresh token — should use static provider
	tokenData := token.TokenData{
		AccessToken: "lin_api_test_static_token",
		TokenType:   "Bearer",
		AuthMode:    "user",
	}
	data, err := json.Marshal(tokenData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, data, 0600))

	client, err := initializeClientWithTokenPath(tokenPath)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "user", client.GetAuthMode())
}

func TestInitializeClientWithTokenPath_AgentMode(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")

	tokenData := token.TokenData{
		AccessToken: "lin_api_test_agent_token",
		TokenType:   "Bearer",
		AuthMode:    "agent",
	}
	data, err := json.Marshal(tokenData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, data, 0600))

	client, err := initializeClientWithTokenPath(tokenPath)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "agent", client.GetAuthMode())
	assert.True(t, client.IsAgentMode())
}

func TestInitializeClientWithTokenPath_LegacyPlainToken(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")

	// Write a legacy plain string token (not JSON)
	require.NoError(t, os.WriteFile(tokenPath, []byte("lin_api_legacy_token_123"), 0600))

	client, err := initializeClientWithTokenPath(tokenPath)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "", client.GetAuthMode()) // Legacy tokens have no auth mode
}

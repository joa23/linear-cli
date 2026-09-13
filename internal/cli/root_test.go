package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
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
		"onboard":   false,
		"auth":      false,
		"issues":    false,
		"documents": false,
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

func TestInitializeClientWithTokenPath_EnvKeyFallback(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	t.Setenv("LINEAR_API_KEY", "lin_api_env_key_456")

	client, err := initializeClientWithTokenPath(tokenPath)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "lin_api_env_key_456", client.GetAPIToken())
}

func TestInitializeClientWithTokenPath_EnvTokenNotSupported(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "nonexistent_token")

	t.Setenv("LINEAR_API_KEY", "")
	t.Setenv("LINEAR_API_TOKEN", "lin_api_env_token_legacy")

	client, err := initializeClientWithTokenPath(tokenPath)
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LINEAR_API_KEY")
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

func TestIsNoAuthInvocation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare invocation prints help", []string{}, true},
		{"root help flag", []string{"--help"}, true},
		{"root short help flag", []string{"-h"}, true},
		{"version flag", []string{"--version"}, true},
		{"help subcommand", []string{"help", "issues"}, true},
		{"completion subcommand", []string{"completion", "zsh"}, true},
		{"auth login", []string{"auth", "login"}, true},
		{"auth after global flag", []string{"--workspace", "auth"}, true},
		{"subcommand help flag", []string{"issues", "create", "--help"}, true},
		{"real command needs auth", []string{"issues", "list"}, false},
		{"real command with flags needs auth", []string{"issues", "list", "--limit", "5"}, false},
		{"search needs auth", []string{"search", "help wanted"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNoAuthInvocation(tc.args); got != tc.want {
				t.Errorf("isNoAuthInvocation(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestDocumentsSubcommands(t *testing.T) {
	cmd := NewRootCmd()
	docsCmd, _, _ := cmd.Find([]string{"documents"})
	require.NotNil(t, docsCmd)
	assert.Contains(t, docsCmd.Aliases, "docs")

	expectedSubCmds := []string{"list", "get", "create", "update", "delete"}
	for _, subCmdName := range expectedSubCmds {
		sub, _, err := cmd.Find([]string{"documents", subCmdName})
		require.NoError(t, err)
		assert.Equal(t, subCmdName, sub.Name())
	}

	createCmd, _, _ := cmd.Find([]string{"documents", "create"})
	for _, flag := range []string{"title", "content", "content-file", "project", "issue", "team", "output", "format"} {
		assert.NotNil(t, createCmd.Flags().Lookup(flag), "create should have --%s", flag)
	}

	listCmd, _, _ := cmd.Find([]string{"documents", "list"})
	for _, flag := range []string{"project", "issue", "team", "query", "include-archived", "limit", "output", "format"} {
		assert.NotNil(t, listCmd.Flags().Lookup(flag), "list should have --%s", flag)
	}
}

func TestReadContentFlags(t *testing.T) {
	file := t.TempDir() + "/body.md"
	require.NoError(t, os.WriteFile(file, []byte("# From file\n"), 0o644))

	newCmd := func() *cobra.Command {
		c := &cobra.Command{Use: "x", Run: func(*cobra.Command, []string) {}}
		c.Flags().String("content", "", "")
		c.Flags().String("content-file", "", "")
		return c
	}

	t.Run("neither", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags(nil))
		body, set, err := readContentFlags(c, "", "")
		require.NoError(t, err)
		assert.False(t, set)
		assert.Equal(t, "", body)
	})

	t.Run("content flag", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags([]string{"--content", "hi"}))
		body, set, err := readContentFlags(c, "hi", "")
		require.NoError(t, err)
		assert.True(t, set)
		assert.Equal(t, "hi", body)
	})

	t.Run("explicit empty content counts as set", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags([]string{"--content", ""}))
		_, set, err := readContentFlags(c, "", "")
		require.NoError(t, err)
		assert.True(t, set)
	})

	t.Run("content file", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags([]string{"--content-file", file}))
		body, set, err := readContentFlags(c, "", file)
		require.NoError(t, err)
		assert.True(t, set)
		assert.Equal(t, "# From file\n", body)
	})

	t.Run("both is an error", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags([]string{"--content", "a", "--content-file", file}))
		_, _, err := readContentFlags(c, "a", file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
	})

	t.Run("missing file", func(t *testing.T) {
		c := newCmd()
		require.NoError(t, c.ParseFlags([]string{"--content-file", "/nonexistent/x.md"}))
		_, _, err := readContentFlags(c, "", "/nonexistent/x.md")
		require.Error(t, err)
	})
}

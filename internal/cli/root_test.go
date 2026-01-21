package cli

import (
	"testing"

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

	// Check for --format flag
	formatFlag := cmd.PersistentFlags().Lookup("format")
	require.NotNil(t, formatFlag)
	assert.Equal(t, "table", formatFlag.DefValue)

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

package cli

import (
	"context"

	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/spf13/cobra"
)

// NewTestDependencies creates a Dependencies instance for testing
// Uses the provided client (which can be a mock or test client)
func NewTestDependencies(client *linear.Client) *Dependencies {
	return NewDependencies(client)
}

// NewCmdWithDeps creates a command with dependencies injected for testing
// This allows tests to provide mock dependencies
func NewCmdWithDeps(deps *Dependencies, cmdFactory func() *cobra.Command) *cobra.Command {
	cmd := cmdFactory()
	ctx := context.WithValue(context.Background(), dependenciesKey, deps)
	cmd.SetContext(ctx)
	return cmd
}

// NewTestClient creates a Linear client with a test token
// Use this with a mock server for integration tests
func NewTestClient(token string) *linear.Client {
	return linear.NewClient(token)
}

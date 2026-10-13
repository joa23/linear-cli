package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBodyCommandHelpRequiresExplicitStdinFlag(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		want string
	}{
		{name: "issue create", cmd: newIssuesCreateCmd, want: "--description -"},
		{name: "issue update", cmd: newIssuesUpdateCmd, want: "--description -"},
		{name: "project create", cmd: newProjectsCreateCmd, want: "--description -"},
		{name: "project update", cmd: newProjectsUpdateCmd, want: "--description -"},
		{name: "comment", cmd: newIssuesCommentCmd, want: "--body -"},
		{name: "reply", cmd: newIssuesReplyCmd, want: "--body -"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"--help"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(output.String(), tt.want) {
				t.Fatalf("help output does not contain %q:\n%s", tt.want, output.String())
			}
			if strings.Contains(output.String(), "or pipe to stdin") || strings.Contains(output.String(), "piped from stdin") {
				t.Fatalf("help output still advertises implicit stdin:\n%s", output.String())
			}
		})
	}
}

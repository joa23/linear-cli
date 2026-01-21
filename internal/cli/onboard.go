package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
)

func newOnboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "onboard",
		Short: "Show setup status and quick start guide",
		Long:  "Display authentication status, available teams, and quick reference for getting started.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOnboard()
		},
	}
}

func runOnboard() error {
	fmt.Println("Light Linear MCP - Setup Status")
	fmt.Println("================================")
	fmt.Println()

	// Check authentication
	tokenStorage := token.NewStorage(token.GetDefaultTokenPath())
	if !tokenStorage.TokenExists() {
		printNotLoggedIn()
		return nil
	}

	linearToken, err := tokenStorage.LoadToken()
	if err != nil {
		printNotLoggedIn()
		return nil
	}

	client := linear.NewClient(linearToken)

	// Get current user
	viewer, err := client.GetViewer()
	if err != nil {
		fmt.Println("Authentication")
		fmt.Println("--------------")
		fmt.Println("  Status: ⚠️  Token invalid or expired")
		fmt.Println("  Action: Run 'linear auth login' to re-authenticate")
		fmt.Println()
		return nil
	}

	// Print auth status
	fmt.Println("Authentication")
	fmt.Println("--------------")
	fmt.Printf("  Status: ✅ Logged in\n")
	fmt.Printf("  User:   %s\n", viewer.Name)
	fmt.Printf("  Email:  %s\n", viewer.Email)
	fmt.Println()

	// Get teams
	teams, err := client.GetTeams()
	if err != nil {
		fmt.Println("Teams: ⚠️  Could not fetch teams")
	} else {
		fmt.Println("Available Teams")
		fmt.Println("---------------")
		if len(teams) == 0 {
			fmt.Println("  No teams found")
		} else {
			for _, team := range teams {
				fmt.Printf("  %s (%s)\n", team.Name, team.Key)

				// Get team members
				users, err := client.ListUsersWithPagination(&linear.UserFilter{
					TeamID: team.ID,
					First:  50,
				})
				if err == nil && len(users.Users) > 0 {
					var memberNames []string
					for _, u := range users.Users {
						if len(memberNames) < 5 {
							memberNames = append(memberNames, u.Name)
						}
					}
					memberStr := strings.Join(memberNames, ", ")
					if len(users.Users) > 5 {
						memberStr += fmt.Sprintf(" +%d more", len(users.Users)-5)
					}
					fmt.Printf("    └─ Members: %s\n", memberStr)
				}
			}
		}
		fmt.Println()
	}

	// Quick reference
	fmt.Println("Quick Reference")
	fmt.Println("---------------")
	fmt.Println("  linear issues list              List your assigned issues")
	fmt.Println("  linear issues get <ID>          Show issue details (e.g., CEN-123)")
	fmt.Println("  linear auth status              Check login status")
	fmt.Println("  linear auth logout              Log out")
	fmt.Println()

	// MCP setup
	fmt.Println("Claude Code Setup")
	fmt.Println("-----------------")
	fmt.Println("  Add to MCP settings:")
	fmt.Println()
	fmt.Println("    {")
	fmt.Println("      \"mcpServers\": {")
	fmt.Println("        \"linear\": {")
	fmt.Printf("          \"command\": \"%s/.local/bin/linear-mcp\"\n", getHomeDir())
	fmt.Println("        }")
	fmt.Println("      }")
	fmt.Println("    }")
	fmt.Println()
	fmt.Println("  Or run: claude mcp add linear ~/.local/bin/linear-mcp")
	fmt.Println()

	return nil
}

func printNotLoggedIn() {
	fmt.Println("Authentication")
	fmt.Println("--------------")
	fmt.Println("  Status: ❌ Not logged in")
	fmt.Println()
	fmt.Println("Getting Started")
	fmt.Println("---------------")
	fmt.Println("  1. Run 'linear auth login' to authenticate with Linear")
	fmt.Println("  2. Run 'linear onboard' again to see your teams")
	fmt.Println("  3. Run 'linear issues list' to see your issues")
	fmt.Println()
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~"
	}
	return home
}

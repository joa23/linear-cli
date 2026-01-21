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
			for i, team := range teams {
				fmt.Printf("  %s (%s)\n", team.Name, team.Key)

				// Get team members
				users, err := client.ListUsersWithPagination(&linear.UserFilter{
					TeamID: team.ID,
					First:  100, // Increased to get more members
				})
				if err == nil && len(users.Users) > 0 {
					var memberNames []string
					for _, u := range users.Users {
						memberNames = append(memberNames, u.Name)
					}
					memberStr := strings.Join(memberNames, ", ")
					fmt.Printf("    ├─ Members (%d): %s\n", len(users.Users), memberStr)
				}

				// Get workflow states for this team
				states, err := client.Workflows.GetWorkflowStates(team.ID)
				if err != nil {
					fmt.Printf("    └─ Workflow States: ⚠️  Could not fetch states: %v\n", err)
				} else if len(states) > 0 {
					fmt.Println("    └─ Workflow States:")
					for j, state := range states {
						prefix := "       ├─"
						if j == len(states)-1 {
							prefix = "       └─"
						}
						fmt.Printf("%s %s (%s)\n", prefix, state.Name, state.Type)
					}
				} else {
					fmt.Println("    └─ Workflow States: (none found)")
				}

				// Add spacing between teams
				if i < len(teams)-1 {
					fmt.Println()
				}
			}
		}
		fmt.Println()
	}

	// Quick reference
	fmt.Println("Quick Reference")
	fmt.Println("---------------")
	fmt.Println()
	fmt.Println("Basic commands:")
	fmt.Println("  linear issues list                    List your assigned issues")
	fmt.Println("  linear issues get CEN-123             Show issue details")
	fmt.Println("  linear auth status                    Check login status")
	fmt.Println()
	fmt.Println("Create issue (full example):")
	fmt.Println("  cat feature.md | linear i create \"Add user authentication\" \\")
	fmt.Println("    -t CEN \\")
	fmt.Println("    -s \"In Progress\" \\")
	fmt.Println("    -p 1 \\")
	fmt.Println("    -a me \\")
	fmt.Println("    -P \"Q1 Goals\" \\")
	fmt.Println("    -e 5 \\")
	fmt.Println("    --due 2026-02-01")
	fmt.Println()
	fmt.Println("Update issue:")
	fmt.Println("  linear i update CEN-123 -s Done -a \"alice@company.com\"")
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

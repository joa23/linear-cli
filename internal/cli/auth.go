package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/joa23/linear-cli/internal/config"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/oauth"
	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Linear authentication",
		Long:  "Authenticate with Linear, check authentication status, and manage credentials.",
	}

	authCmd.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newStatusCmd(),
	)

	return authCmd
}

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Linear",
		Long:  "Authenticate with Linear using OAuth2. Opens your browser for authorization.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleLogin()
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out from Linear",
		Long:  "Remove stored Linear credentials from your system.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleLogout()
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Linear authentication status",
		Long:  "Display your current Linear authentication status and user information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleStatus()
		},
	}
}

func handleLogin() error {
	fmt.Println("\nWelcome to Linear CLI!")

	// Step 1: Ask auth mode
	authMode, err := promptAuthMode()
	if err != nil {
		return err
	}

	// Step 2: Load or prompt for credentials
	cfgManager := config.NewManager("")
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientID := cfg.Linear.ClientID
	clientSecret := cfg.Linear.ClientSecret
	port := cfg.Linear.Port

	// Also check environment variables
	if clientID == "" {
		clientID = os.Getenv("LINEAR_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("LINEAR_CLIENT_SECRET")
	}

	// If credentials missing, prompt for them
	if clientID == "" || clientSecret == "" || port == 0 {
		clientID, clientSecret, port, err = promptCredentials()
		if err != nil {
			return err
		}

		// Ask to save credentials
		if promptConfirmation(fmt.Sprintf("\nSave credentials to %s?", cfgManager.GetConfigPath())) {
			cfg.Linear.ClientID = clientID
			cfg.Linear.ClientSecret = clientSecret
			cfg.Linear.Port = port
			if err := cfgManager.Save(cfg); err != nil {
				fmt.Printf("Warning: Could not save config: %v\n", err)
			} else {
				fmt.Printf("Credentials saved to %s\n", cfgManager.GetConfigPath())
			}
		}
	}

	// Step 3: Run OAuth flow
	oauthHandler := oauth.NewHandlerWithClient(clientID, clientSecret, linear.GetSharedHTTPClient())
	state := oauth.GenerateState()

	// Use specified port (must match OAuth app configuration)
	portStr := fmt.Sprintf("%d", port)
	redirectURI := fmt.Sprintf("http://localhost:%s/oauth-callback", portStr)

	var authURL string
	if authMode == "user" {
		authURL = oauthHandler.GetAuthorizationURL(redirectURI, state)
	} else {
		authURL = oauthHandler.GetAppAuthorizationURL(redirectURI, state)
	}

	fmt.Println("\nOpening browser for Linear authentication...")
	fmt.Printf("If browser doesn't open, visit: %s\n", authURL)

	// Open browser
	openBrowser(authURL)

	// Handle OAuth callback and get full token response
	tokenResponse, err := oauthHandler.HandleCallbackWithFullResponse(portStr, state)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			return fmt.Errorf("port %d is already in use.\n\nTo fix this:\n  1. Find the process using port %d: lsof -i :%d\n  2. Kill it, or wait and try again\n  3. Or use a different port: linear auth login --port <PORT>", port, port, port)
		}
		return fmt.Errorf("OAuth callback failed: %w", err)
	}

	// Convert to structured format and save
	tokenData := tokenResponse.ToTokenData()
	tokenStorage := token.NewStorage(token.GetDefaultTokenPath())
	if err := tokenStorage.SaveTokenData(tokenData); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\nSuccessfully authenticated with Linear!")
	fmt.Println("Token saved to:", token.GetDefaultTokenPath())
	if tokenData.RefreshToken != "" {
		fmt.Println("✓ Token will be automatically refreshed before expiration")
	}

	// Extract access token for showing teams
	accessToken := tokenData.AccessToken

	// Show available teams
	fmt.Println("\nAvailable teams:")
	client := linear.NewClient(accessToken)
	if teams, err := client.GetTeams(); err == nil {
		for _, team := range teams {
			fmt.Printf("  - %s (ID: %s, Key: %s)\n", team.Name, team.ID, team.Key)
		}
		fmt.Println("\nYou can use these team IDs when creating issues.")
	} else {
		fmt.Printf("Warning: Could not fetch teams: %v\n", err)
	}

	// Show user info
	if viewer, err := client.GetViewer(); err == nil {
		fmt.Printf("\nLogged in as: %s (%s)\n", viewer.Name, viewer.Email)
	}

	return nil
}

// promptAuthMode asks the user to choose between user and agent authentication
func promptAuthMode() (string, error) {
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("Authentication Mode")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("\n[1] As yourself (personal use)")
	fmt.Println("    • Your actions appear under your Linear account")
	fmt.Println("    • For personal task management")
	fmt.Println("\n[2] As an agent (automation, bots)")
	fmt.Println("    • Agent appears as a separate entity in Linear")
	fmt.Println("    • Requires admin approval to install")
	fmt.Println("    • Agent can be @mentioned and assigned issues")
	fmt.Println("    • For automated workflows and integrations")
	fmt.Print("\nChoice [1/2]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	switch input {
	case "1", "":
		return "user", nil
	case "2":
		fmt.Println("\n⚠️  Agent mode requires Linear workspace admin approval")
		return "agent", nil
	default:
		return "", fmt.Errorf("invalid choice: %s (expected 1 or 2)", input)
	}
}

// promptCredentials prompts the user for OAuth credentials
func promptCredentials() (clientID, clientSecret string, port int, err error) {
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("Linear OAuth credentials not found.")
	fmt.Println("\nTo get credentials, create an OAuth app at:")
	fmt.Println("  Linear → Settings → API → OAuth Applications → New")
	fmt.Println(strings.Repeat("─", 50))

	reader := bufio.NewReader(os.Stdin)

	// Prompt for port first
	fmt.Print("\nOAuth callback port [37412]: ")
	portInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read port: %w", err)
	}
	portInput = strings.TrimSpace(portInput)

	// Use default if empty
	if portInput == "" {
		port = 37412
	} else {
		port, err = strconv.Atoi(portInput)
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid port number: %w", err)
		}
	}

	// Show the callback URL they should configure
	fmt.Printf("\nCallback URL: http://localhost:%d/oauth-callback\n", port)
	fmt.Println("(Configure this in your Linear OAuth app)")

	fmt.Print("\nClient ID: ")
	clientID, err = reader.ReadString('\n')
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read client ID: %w", err)
	}
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Client Secret: ")
	clientSecret, err = reader.ReadString('\n')
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read client secret: %w", err)
	}
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" || clientSecret == "" {
		return "", "", 0, fmt.Errorf("client ID and client secret are required")
	}

	return clientID, clientSecret, port, nil
}

// promptConfirmation asks a yes/no question and returns true for yes
func promptConfirmation(question string) bool {
	fmt.Printf("%s [Y/n]: ", question)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "" || input == "y" || input == "yes"
}

// openBrowser opens the given URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}

	if cmd != nil {
		cmd.Run()
	}
}

func handleLogout() error {
	tokenStorage := token.NewStorage(token.GetDefaultTokenPath())
	if err := tokenStorage.DeleteToken(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	fmt.Println("✅ Successfully logged out from Linear")
	fmt.Println("Token removed from:", token.GetDefaultTokenPath())
	return nil
}

func handleStatus() error {
	tokenStorage := token.NewStorage(token.GetDefaultTokenPath())
	if !tokenStorage.TokenExists() {
		fmt.Println("❌ Not logged in to Linear")
		fmt.Println("Run 'linear auth login' to authenticate")
		return nil
	}

	// Try to load token and verify it
	tokenData, err := tokenStorage.LoadTokenData()
	if err != nil || tokenData.AccessToken == "" {
		fmt.Println("⚠️  Token file exists but could not be read")
		return nil
	}

	// Create Linear client and test connection
	client := linear.NewClient(tokenData.AccessToken)
	if viewer, err := client.GetViewer(); err == nil {
		fmt.Println("✅ Logged in to Linear")
		fmt.Printf("User: %s (%s)\n", viewer.Name, viewer.Email)
		fmt.Printf("ID: %s\n", viewer.ID)
	} else {
		fmt.Println("⚠️  Token exists but may be invalid")
		fmt.Println("Error:", err)
		fmt.Println("Try running 'linear auth login' to re-authenticate")
	}

	return nil
}

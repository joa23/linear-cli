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
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/oauth"
	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:         "auth",
		Short:       "Manage Linear authentication",
		Long:        "Authenticate with Linear, check authentication status, and manage credentials.",
		Annotations: map[string]string{"skipAuth": "true"},
	}

	authCmd.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newStatusCmd(),
		newAuthListCmd(),
	)

	return authCmd
}

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Linear",
		Long: `Authenticate with Linear using OAuth2. Opens your browser for authorization.

You'll choose an authentication mode:
  - User mode:  --assignee me assigns to your personal account
  - Agent mode: --assignee me assigns to the OAuth app (delegate)

Run 'linear auth status' to check your current mode.`,
		Annotations: map[string]string{"skipAuth": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleLogin(resolveWorkspaceExplicit())
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "logout",
		Short:       "Log out from Linear",
		Long:        "Remove stored Linear credentials from your system.",
		Annotations: map[string]string{"skipAuth": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleLogout(resolveWorkspace())
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Linear authentication status",
		Long: `Display your current Linear authentication status and user information.

Shows your auth mode which determines how --assignee me behaves:
  - User mode:  assigns to your personal account
  - Agent mode: assigns to the OAuth app (delegate)`,
		Annotations: map[string]string{"skipAuth": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleStatus(resolveWorkspace())
		},
	}
}

func handleLogin(p string) error {
	// If no workspace specified via --workspace flag, show workspace picker when workspaces exist
	isNew := false
	if p == "" {
		p, isNew = promptLoginWorkspacePicker()
	}

	if p != "" {
		fmt.Printf("\nWelcome to Linear CLI! (workspace: %s)\n", p)
	} else {
		fmt.Println("\nWelcome to Linear CLI!")
	}

	// Step 1: Determine auth mode — reuse existing if set, only prompt for new workspaces
	var authMode string
	var existingStorage *token.Storage
	hasExistingToken := false
	if !isNew {
		existingStorage = token.NewStorage(token.GetWorkspaceTokenPath(p))
		hasExistingToken = existingStorage.TokenExists()
	}

	if hasExistingToken {
		if existingData, err := existingStorage.LoadTokenData(); err == nil && existingData.AuthMode != "" {
			authMode = existingData.AuthMode
			fmt.Printf("Auth mode: %s (from existing workspace)\n", authMode)
		}
	}

	if authMode == "" {
		var err error
		authMode, err = promptAuthMode()
		if err != nil {
			return err
		}
	}

	// Step 2: Load or prompt for credentials
	// First, check existing token for stored credentials (reuse on re-login).
	// Credentials are intentionally stored in both config (for re-login discovery)
	// and token (for self-contained refresh without config dependency).
	var clientID, clientSecret string
	var port int

	if hasExistingToken {
		if existingData, err := existingStorage.LoadTokenData(); err == nil {
			if existingData.ClientID != "" && existingData.ClientSecret != "" {
				clientID = existingData.ClientID
				clientSecret = existingData.ClientSecret
				fmt.Println("Reusing stored credentials for this profile.")
			}
		}
	}

	// Fall back to config file / env vars if not found in token
	cfgManager := config.NewManager("")
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	noExistingToken := !hasExistingToken

	if clientID == "" || clientSecret == "" {
		// If workspace specified but doesn't exist yet, always prompt for new credentials
		// (don't fall back to global credentials for a new workspace)
		_, workspaceExists := cfg.GetWorkspace(p)
		if p != "" && !workspaceExists {
			fmt.Printf("No credentials found for workspace %q. Let's set them up.\n", p)
			clientID, clientSecret, port = "", "", 0
		} else {
			clientID, clientSecret = cfg.GetLinearCredentials(p)
			port = cfg.GetLinearPort(p)

			// Also check environment variables (only for non-workspace login)
			if p == "" {
				if clientID == "" {
					clientID = os.Getenv("LINEAR_CLIENT_ID")
				}
				if clientSecret == "" {
					clientSecret = os.Getenv("LINEAR_CLIENT_SECRET")
				}
			}
		}
	} else {
		// We have credentials from token, still need port from config
		port = cfg.GetLinearPort(p)
	}

	// If credentials missing, OR this is a fresh token setup (no existing token), prompt for them.
	// The port is always re-asked on a fresh auth so the user can confirm or change it.
	if clientID == "" || clientSecret == "" || port == 0 || noExistingToken {
		clientID, clientSecret, port, err = promptCredentials()
		if err != nil {
			return err
		}

		// Ask to save credentials
		if promptConfirmation(fmt.Sprintf("\nSave credentials to %s?", cfgManager.GetConfigPath())) {
			if p != "" {
				cfg.SetWorkspace(p, config.WorkspaceConfig{
					Name:               p,
					LinearClientID:     clientID,
					LinearClientSecret: clientSecret,
					Port:               port,
				})
			} else {
				cfg.Linear.ClientID = clientID
				cfg.Linear.ClientSecret = clientSecret
				cfg.Linear.Port = port
			}
			if err := cfgManager.Save(cfg); err != nil {
				fmt.Printf("Warning: Could not save config: %v\n", err)
			} else {
				fmt.Printf("Credentials saved to %s\n", cfgManager.GetConfigPath())
			}
		}
	}

	// Step 3: Try client credentials first (no browser needed if app already installed)
	oauthHandler := oauth.NewHandlerWithClient(clientID, clientSecret, core.GetSharedHTTPClient())

	fmt.Println("\nAttempting client credentials grant (no browser needed)...")
	agentMode := authMode == "agent"
	tokenResponse, err := oauthHandler.TryClientCredentials(agentMode)
	if err != nil {
		return fmt.Errorf("client credentials: %w", err)
	}

	if tokenResponse == nil {
		// Fall back to browser OAuth flow
		state := oauth.GenerateState()
		portStr := fmt.Sprintf("%d", port)
		redirectURI := fmt.Sprintf("http://localhost:%s/oauth-callback", portStr)

		var authURL string
		if authMode == "user" {
			authURL = oauthHandler.GetAuthorizationURL(redirectURI, state)
		} else {
			authURL = oauthHandler.GetAppAuthorizationURL(redirectURI, state)
		}

		fmt.Println("Client credentials not available, falling back to browser flow...")
		fmt.Printf("If browser doesn't open, visit: %s\n", authURL)

		// Open browser
		openBrowser(authURL)

		// Handle OAuth callback and get full token response
		tokenResponse, err = oauthHandler.HandleCallbackWithFullResponse(portStr, state)
		if err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				return fmt.Errorf("port %d is already in use.\n\nTo fix this:\n  1. Find the process using port %d: lsof -i :%d\n  2. Kill it, or wait and try again\n  3. Or use a different port: linear auth login --port <PORT>", port, port, port)
			}
			return fmt.Errorf("OAuth callback failed: %w", err)
		}
	} else {
		fmt.Println("Got token via client credentials.")
	}

	// Convert to structured format and save with auth mode + credentials
	tokenData := tokenResponse.ToTokenData()
	tokenData.AuthMode = authMode         // Store "user" or "agent" for correct "me" resolution
	tokenData.ClientID = clientID         // Store for self-contained token refresh
	tokenData.ClientSecret = clientSecret // Store for self-contained token refresh

	// For new workspaces, derive the workspace name from the org's URL key — no user prompt needed.
	if isNew {
		tempClient := linear.NewClient(tokenData.AccessToken)
		if viewer, err := tempClient.GetViewer(); err == nil && viewer.Organization.URLKey != "" {
			p = viewer.Organization.URLKey
			tokenData.OrgName = viewer.Organization.Name
			tokenData.OrgURLKey = p
			fmt.Printf("\nOrganization: %s (%s.linear.app)\n", viewer.Organization.Name, p)
			fmt.Printf("Saving as workspace: %s\n", p)
		} else {
			// Fallback: API unreachable — use "default" so we never write to the legacy path.
			p = "default"
			fmt.Println("Saving as workspace: default")
		}
	}

	tokenPath := token.GetWorkspaceTokenPath(p)
	tokenStorage := token.NewStorage(tokenPath)
	if err := tokenStorage.SaveTokenData(tokenData); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\nSuccessfully authenticated with Linear!")
	fmt.Println("Token saved to:", tokenPath)
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

	// Show user info and persist org metadata for workspace display
	if viewer, err := client.GetViewer(); err == nil {
		fmt.Printf("\nLogged in as: %s (%s)\n", viewer.Name, viewer.Email)
		if tokenData.OrgName == "" && viewer.Organization.Name != "" {
			tokenData.OrgName = viewer.Organization.Name
			tokenData.OrgURLKey = viewer.Organization.URLKey
			// Re-save with org metadata now populated
			if err := tokenStorage.SaveTokenData(tokenData); err != nil {
				fmt.Printf("Warning: could not update token with org info: %v\n", err)
			}
		}
	}

	return nil
}

// promptLoginWorkspacePicker shows a workspace picker when existing workspaces are found.
// Returns the selected workspace name and whether this is a new workspace setup.
// For new workspace: returns ("", true) — name will be derived post-OAuth from the org's urlKey.
// For existing workspace: returns (name, false).
// If no existing workspaces, returns ("", true) immediately (first-time setup).
func promptLoginWorkspacePicker() (string, bool) {
	tokens := token.ListWorkspaceTokens()
	if len(tokens) == 0 {
		return "", true // No existing workspaces — new setup
	}

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("Existing workspaces:")
	fmt.Println(strings.Repeat("-", 50))

	for i, wt := range tokens {
		name := "default"
		if wt.Name != "" {
			name = wt.Name
		}
		// Try to show identity for context
		storage := token.NewStorage(wt.Path)
		label := name
		if td, err := storage.LoadTokenData(); err == nil {
			if td.AuthMode != "" {
				label += fmt.Sprintf(" [%s]", td.AuthMode)
			}
		}
		fmt.Printf("  [%d] %s (refresh)\n", i+1, label)
	}
	newIdx := len(tokens) + 1
	fmt.Printf("  [%d] Add new workspace\n", newIdx)

	fmt.Printf("\nChoice [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", false // default on error
	}
	input = strings.TrimSpace(input)

	selection := 1
	if input != "" {
		sel, err := strconv.Atoi(input)
		if err != nil || sel < 1 || sel > newIdx {
			fmt.Println("Invalid selection, using default.")
			return "", false
		}
		selection = sel
	}

	// "Add new workspace" selected — defer naming to post-OAuth
	if selection == newIdx {
		return "", true
	}

	// Existing workspace selected
	return tokens[selection-1].Name, false
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
		_ = cmd.Run()
	}
}

func handleLogout(p string) error {
	tokenPath := token.GetWorkspaceTokenPath(p)
	tokenStorage := token.NewStorage(tokenPath)
	if err := tokenStorage.DeleteToken(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	fmt.Println("✅ Successfully logged out from Linear")
	if p != "" {
		fmt.Printf("Token removed for workspace %q: %s\n", p, tokenPath)
	} else {
		fmt.Println("Token removed from:", tokenPath)
	}
	return nil
}

func handleStatus(p string) error {
	// If a specific workspace is requested, show just that one
	if p != "" {
		return showWorkspaceStatus(p, false)
	}

	// No workspace specified: enumerate all tokens
	tokens := token.ListWorkspaceTokens()

	// Also check LINEAR_API_KEY env var
	envAuth := os.Getenv("LINEAR_API_KEY") != ""
	if envAuth {
		fmt.Println("LINEAR_API_KEY: set (overrides stored tokens)")
		fmt.Println()
	}

	if len(tokens) == 0 {
		if envAuth {
			return showWorkspaceStatus("", false)
		}
		fmt.Println("❌ Not logged in to Linear")
		fmt.Println("Run 'linear auth login' to authenticate")
		return nil
	}

	// Determine the active workspace from .linear.yaml
	activeWorkspace := GetDefaultWorkspace()

	for i, wt := range tokens {
		if i > 0 {
			fmt.Println()
		}
		isActive := wt.Name == activeWorkspace
		showSingleTokenStatus(wt, isActive)
	}

	return nil
}

// showWorkspaceStatus shows status for a specific workspace.
func showWorkspaceStatus(p string, showActive bool) error {
	tokenPath := token.GetWorkspaceTokenPath(p)

	if p != "" {
		fmt.Printf("Workspace: %s\n", p)
	}

	client, err := initializeClientWithTokenPath(p, tokenPath)
	if err != nil {
		fmt.Println("❌ Not logged in to Linear")
		if p != "" {
			fmt.Printf("Run 'linear auth login --workspace %s' to authenticate\n", p)
		} else {
			fmt.Println("Run 'linear auth login' to authenticate")
		}
		return nil
	}

	if viewer, err := client.GetViewer(); err == nil {
		fmt.Println("✅ Logged in to Linear")
		fmt.Printf("User: %s (%s)\n", viewer.Name, viewer.Email)
		fmt.Printf("ID: %s\n", viewer.ID)
		switch client.GetAuthMode() {
		case "agent":
			fmt.Println("Mode: Agent (--assignee me uses delegate)")
		case "user":
			fmt.Println("Mode: User (--assignee me uses assignee)")
		default:
			if os.Getenv("LINEAR_API_KEY") != "" {
				fmt.Println("Mode: API Key (LINEAR_API_KEY)")
			} else {
				loginCmd := "linear auth login"
				if p != "" {
					loginCmd += " --workspace " + p
				}
				fmt.Printf("\n⚠️  Auth mode not set. Run '%s' to configure.\n", loginCmd)
			}
		}
	} else {
		fmt.Println("⚠️  Credentials exist but may be invalid")
		fmt.Println("Error:", err)
		fmt.Println("Try running 'linear auth login' or set a valid LINEAR_API_KEY")
	}

	return nil
}

// showSingleTokenStatus displays status for a single workspace token entry.
func showSingleTokenStatus(wt token.WorkspaceToken, isActive bool) {
	// Header
	name := "default"
	if wt.Name != "" {
		name = wt.Name
	}
	activeMarker := ""
	if isActive {
		activeMarker = " (active)"
	}
	fmt.Printf("-- %s%s --\n", name, activeMarker)

	storage := token.NewStorage(wt.Path)
	tokenData, err := storage.LoadTokenData()
	if err != nil || tokenData.AccessToken == "" {
		fmt.Println("  ⚠️  Token file exists but could not be read")
		return
	}

	// Try to get identity
	client := linear.NewClientWithAuthMode(tokenData.AccessToken, tokenData.AuthMode)
	if viewer, err := client.GetViewer(); err == nil {
		fmt.Printf("  User: %s (%s)\n", viewer.Name, viewer.Email)
	} else {
		fmt.Printf("  ⚠️  Token may be invalid: %v\n", err)
	}

	// Auth mode
	switch tokenData.AuthMode {
	case "agent":
		fmt.Println("  Mode: Agent")
	case "user":
		fmt.Println("  Mode: User")
	default:
		fmt.Println("  Mode: (not set)")
	}

	// Expiry
	if !tokenData.ExpiresAt.IsZero() {
		if token.IsExpired(tokenData) {
			fmt.Println("  Token: Expired")
		} else {
			fmt.Printf("  Token: Expires %s\n", tokenData.ExpiresAt.Format("2006-01-02 15:04"))
		}
	}
}

// newAuthListCmd creates the 'auth list' subcommand.
// Lists all configured workspaces offline (no API calls).
func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured authentication workspaces",
		Long: `List all configured authentication workspaces. This command works offline
and does not make any API calls. Shows workspace name, auth mode, and token status.`,
		Annotations: map[string]string{"skipAuth": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleAuthList()
		},
	}
}

func handleAuthList() error {
	tokens := token.ListWorkspaceTokens()

	if len(tokens) == 0 {
		fmt.Println("No workspaces configured.")
		fmt.Println("Run 'linear auth login' to set up authentication.")
		return nil
	}

	// Determine the active workspace from .linear.yaml
	activeWorkspace := GetDefaultWorkspace()

	for i, wt := range tokens {
		if i > 0 {
			fmt.Println()
		}

		name := "default"
		if wt.Name != "" {
			name = wt.Name
		}

		// Active marker
		activeMarker := ""
		if wt.Name == activeWorkspace {
			activeMarker = " *"
		}

		// Load token data for mode and expiry (all local, no API)
		storage := token.NewStorage(wt.Path)
		tokenData, err := storage.LoadTokenData()
		if err != nil {
			fmt.Printf("%s%s  mode=?  token=unreadable\n", name, activeMarker)
			continue
		}

		// Auth mode
		mode := tokenData.AuthMode
		if mode == "" {
			mode = "(not set)"
		}

		// Token status
		var tokenStatus string
		if tokenData.ExpiresAt.IsZero() {
			tokenStatus = "valid (no expiry)"
		} else if token.IsExpired(tokenData) {
			tokenStatus = "expired"
		} else {
			tokenStatus = fmt.Sprintf("valid (expires %s)", tokenData.ExpiresAt.Format("2006-01-02 15:04"))
		}

		fmt.Printf("%s%s  mode=%s  token=%s\n", name, activeMarker, mode, tokenStatus)
	}

	return nil
}

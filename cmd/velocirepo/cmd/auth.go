package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/auth"
	"github.com/spf13/cobra"
)

const linkedInRedirectURI = "http://localhost:9876/callback"

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with an OAuth provider",
	}
	cmd.AddCommand(authLinkedInCmd())
	return cmd
}

func authLinkedInCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "linkedin",
		Short: "Authenticate with LinkedIn (OAuth 2.0)",
		RunE:  runAuthLinkedIn,
	}
}

func runAuthLinkedIn(cmd *cobra.Command, args []string) error {
	envPath := filepath.Join(cfg.Dir, ".env")

	clientID := auth.ReadEnvValue(envPath, "LINKEDIN_CLIENT_ID")
	clientSecret := auth.ReadEnvValue(envPath, "LINKEDIN_CLIENT_SECRET")

	reader := bufio.NewReader(os.Stdin)

	if clientID == "" || clientSecret == "" {
		fmt.Println("LinkedIn OAuth Setup")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Println()
		fmt.Println("1. Create a LinkedIn Developer App at https://www.linkedin.com/developers/apps/new")
		fmt.Println("2. Request the \"Community Management API\" product in the Products tab")
		fmt.Printf("3. Add %s as an Authorized Redirect URL in the Auth tab\n", linkedInRedirectURI)
		fmt.Println()
	}

	if clientID == "" {
		fmt.Print("Client ID: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		clientID = strings.TrimSpace(input)
		if clientID == "" {
			return fmt.Errorf("client ID is required")
		}
	}

	if clientSecret == "" {
		fmt.Print("Client Secret: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		clientSecret = strings.TrimSpace(input)
		if clientSecret == "" {
			return fmt.Errorf("client secret is required")
		}
	}

	flow := &auth.OAuthFlow{
		AuthURL:      "https://www.linkedin.com/oauth/v2/authorization",
		TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  linkedInRedirectURI,
		Scopes:       []string{"r_organization_social"},
	}

	fmt.Println()
	fmt.Println("Waiting for callback...")

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	tokenResp, err := flow.Run(ctx, func(authURL string) {
		fmt.Println("Opening browser for authorization...")
		fmt.Printf("If it doesn't open, visit: %s\n", authURL)
		fmt.Println()
		openBrowser(authURL)
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	updates := map[string]string{
		"LINKEDIN_CLIENT_ID":     clientID,
		"LINKEDIN_CLIENT_SECRET": clientSecret,
		"LINKEDIN_TOKEN":         tokenResp.AccessToken,
	}

	if err := auth.UpsertEnvFile(envPath, updates); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	expiryDays := tokenResp.ExpiresIn / 86400
	if expiryDays > 0 {
		fmt.Printf("\nToken saved to .env (expires in %d days)\n", expiryDays)
	} else {
		fmt.Println("\nToken saved to .env")
	}

	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}

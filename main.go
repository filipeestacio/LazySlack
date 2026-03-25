package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/app"
	"github.com/filipeestacio/lazyslack/internal/auth"
	"github.com/filipeestacio/lazyslack/internal/config"
	islack "github.com/filipeestacio/lazyslack/internal/slack"
)

func main() {
	debug := len(os.Args) > 1 && os.Args[1] == "--debug"

	if debug {
		f, err := os.OpenFile("lazyslack.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Auth.Method == "oauth" && cfg.Auth.OAuthClientID != "" && cfg.Auth.Token == "" {
		scopes := []string{
			"channels:read", "channels:history", "chat:write", "reactions:write",
			"users:read", "im:read", "im:history", "mpim:read", "mpim:history",
			"groups:read", "groups:history",
		}
		token, err := auth.RunOAuthFlow(cfg.Auth.OAuthClientID, cfg.Auth.OAuthClientSecret, scopes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "OAuth failed: %v\n", err)
			os.Exit(1)
		}
		cfg.Auth.Token = token
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	if cfg.Auth.Token == "" {
		fmt.Println("No auth token configured.")
		fmt.Println("Edit ~/.config/lazyslack/config.yaml with your auth details.")
		fmt.Println()
		fmt.Println("Session token method:")
		fmt.Println("  1. Open Slack in your browser")
		fmt.Println("  2. Open Developer Tools (F12)")
		fmt.Println("  3. Go to Application > Cookies, copy the 'd' cookie value")
		fmt.Println("  4. In Network tab, find any API call, copy the 'Authorization: Bearer xoxc-...' token")
		fmt.Println()
		fmt.Println("OAuth method:")
		fmt.Println("  1. Create a Slack app at https://api.slack.com/apps")
		fmt.Println("  2. Add user token scopes: channels:read, channels:history, chat:write,")
		fmt.Println("     reactions:write, users:read, im:read, im:history, mpim:read,")
		fmt.Println("     mpim:history, groups:read, groups:history")
		fmt.Println("  3. Set oauth_client_id and oauth_client_secret in config.yaml")
		fmt.Println("  4. Run lazyslack again to complete the OAuth flow")
		os.Exit(0)
	}

	info, err := auth.ValidateToken(cfg, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Auth failed: %v\nPlease check your token in ~/.config/lazyslack/config.yaml\n", err)
		os.Exit(1)
	}

	workspace := info.Team
	if cfg.Workspace.Name != "" {
		workspace = cfg.Workspace.Name
	}

	client := islack.NewClient(cfg.Auth.Token, cfg.Auth.Cookie)
	userCache := islack.NewUserCache(client)
	model := app.AsTeaModel(app.New(client, userCache, workspace))

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

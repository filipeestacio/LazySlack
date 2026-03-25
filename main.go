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

	if cfg.Auth.Token == "" {
		fmt.Println("No auth token configured.")
		fmt.Println("Edit ~/.config/lazyslack/config.yaml with your Slack session token:")
		fmt.Println()
		fmt.Println("  1. Open Slack in your browser")
		fmt.Println("  2. Open Developer Tools (F12) and go to the Network tab")
		fmt.Println("  3. Find any request to slack.com, copy the 'Authorization: Bearer xoxc-...' header value")
		fmt.Println("  4. Go to Application > Storage > Cookies, copy the value of the 'd' cookie")
		fmt.Println()
		fmt.Println("Set token and cookie in ~/.config/lazyslack/config.yaml:")
		fmt.Println("  auth:")
		fmt.Println("    method: session_token")
		fmt.Println("    token: xoxc-...")
		fmt.Println("    cookie: \"d=xoxd-...\"")
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

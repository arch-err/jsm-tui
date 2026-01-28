package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/tui"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease create a config file at ~/.config/jsm-tui/config.yaml with the following format:\n\n")
		fmt.Fprintf(os.Stderr, "url: https://your-jira-instance.com\n")
		fmt.Fprintf(os.Stderr, "auth:\n")
		fmt.Fprintf(os.Stderr, "  type: pat  # or 'basic'\n")
		fmt.Fprintf(os.Stderr, "  token: your-personal-access-token\n")
		fmt.Fprintf(os.Stderr, "  # For basic auth, use username and password instead:\n")
		fmt.Fprintf(os.Stderr, "  # username: your-username\n")
		fmt.Fprintf(os.Stderr, "  # password: your-password\n")
		fmt.Fprintf(os.Stderr, "project: YOUR-PROJECT-KEY\n")
		os.Exit(1)
	}

	// Create and run the TUI
	p := tea.NewProgram(
		tui.NewModel(cfg),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

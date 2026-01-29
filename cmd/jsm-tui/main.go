package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/tui"
)

var (
	issueKey string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "jsm-tui",
		Short: "A TUI for Jira Service Management",
		Long:  "A terminal user interface for managing Jira Service Management issues and queues.",
		RunE:  run,
	}

	// Add flags
	rootCmd.Flags().StringVarP(&issueKey, "key", "K", "", "Open directly to a specific issue (e.g., FKS-2417)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
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
		return err
	}

	// Create the model with optional initial issue
	model := tui.NewModelWithOptions(cfg, tui.ModelOptions{
		InitialIssueKey: issueKey,
	})

	// Create and run the TUI
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running application: %w", err)
	}

	return nil
}

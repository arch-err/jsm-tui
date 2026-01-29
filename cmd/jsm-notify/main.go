package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
)

var (
	queueName    string
	pollInterval time.Duration
	verbose      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "jsm-notify",
		Short: "Desktop notifications for new Jira Service Management issues",
		Long: `A notification daemon that polls a JSM queue and sends desktop
notifications when new issues are created.

Uses the same config file as jsm-tui (~/.config/jsm-tui/config.yaml).

Example:
  jsm-notify --queue "To Do" --interval 30s
  jsm-notify -q "Support Queue" -i 1m`,
		RunE: run,
	}

	rootCmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue name to monitor (required)")
	rootCmd.Flags().DurationVarP(&pollInterval, "interval", "i", 30*time.Second, "Poll interval (e.g., 30s, 1m, 5m)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("queue")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	client := jira.NewClient(cfg)

	// Find the queue ID by name
	queues, err := client.GetQueues(cfg.Project)
	if err != nil {
		return fmt.Errorf("failed to get queues: %w", err)
	}

	var queueID string
	for _, q := range queues {
		if q.Name == queueName {
			queueID = q.ID
			break
		}
	}

	if queueID == "" {
		fmt.Fprintf(os.Stderr, "Queue '%s' not found. Available queues:\n", queueName)
		for _, q := range queues {
			fmt.Fprintf(os.Stderr, "  - %s\n", q.Name)
		}
		return fmt.Errorf("queue not found")
	}

	if verbose {
		fmt.Printf("Monitoring queue: %s (ID: %s)\n", queueName, queueID)
		fmt.Printf("Poll interval: %s\n", pollInterval)
	}

	// Track seen issues
	seenIssues := make(map[string]bool)

	// Initial fetch to populate seen issues (don't notify for existing)
	issues, err := client.GetQueueIssues(cfg.Project, queueID, 0, 50)
	if err != nil {
		return fmt.Errorf("failed to fetch initial issues: %w", err)
	}

	for _, issue := range issues {
		seenIssues[issue.Key] = true
	}

	if verbose {
		fmt.Printf("Found %d existing issues in queue\n", len(seenIssues))
		fmt.Println("Watching for new issues...")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			if verbose {
				fmt.Println("\nShutting down...")
			}
			return nil

		case <-ticker.C:
			issues, err := client.GetQueueIssues(cfg.Project, queueID, 0, 50)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
				continue
			}

			for _, issue := range issues {
				if !seenIssues[issue.Key] {
					seenIssues[issue.Key] = true
					notifyNewIssue(issue, queueName, verbose)
				}
			}
		}
	}
}

// notifyNewIssue sends a desktop notification for a new issue
func notifyNewIssue(issue jira.Issue, queueName string, verbose bool) {
	title := fmt.Sprintf("New issue in %s", queueName)
	body := fmt.Sprintf("%s: %s", issue.Key, issue.Fields.Summary)

	if verbose {
		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), body)
	}

	// Use notify-send for desktop notification
	cmd := exec.Command("notify-send",
		"--app-name=jsm-notify",
		"--urgency=normal",
		title,
		body,
	)
	cmd.Run() // Ignore errors - notification might not be available
}

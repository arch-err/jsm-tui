package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
)

func main() {
	root := &cobra.Command{
		Use:           "jsm-cli",
		Short:         "Non-interactive CLI for Jira Service Management",
		Long:          "JSON-output CLI cousin of jsm-tui. Designed for agent and script use.\nReads config from ~/.config/jsm-tui/config.yaml.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		meCmd(),
		queuesCmd(),
		queueCmd(),
		issueCmd(),
		transitionsCmd(),
		commentCmd(),
		transitionCmd(),
		assignCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func loadClient() (*jira.Client, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	return jira.NewClient(cfg), cfg, nil
}

func emitJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func meCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the currently authenticated user",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			user, err := c.GetCurrentUser()
			if err != nil {
				return err
			}
			return emitJSON(user)
		},
	}
}

func queuesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "queues",
		Short: "List service desk queues for the configured project",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, cfg, err := loadClient()
			if err != nil {
				return err
			}
			qs, err := c.GetQueues(cfg.Project)
			if err != nil {
				return err
			}
			return emitJSON(qs)
		},
	}
}

func queueCmd() *cobra.Command {
	var (
		start int
		limit int
	)
	cmd := &cobra.Command{
		Use:   "queue <name>",
		Short: "List issues in a queue (matched by name, case-insensitive)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			c, cfg, err := loadClient()
			if err != nil {
				return err
			}
			qs, err := c.GetQueues(cfg.Project)
			if err != nil {
				return err
			}
			var match *jira.Queue
			for i := range qs {
				if strings.EqualFold(qs[i].Name, name) {
					match = &qs[i]
					break
				}
			}
			if match == nil {
				return fmt.Errorf("queue not found: %q", name)
			}
			issues, err := c.GetQueueIssues(cfg.Project, match.ID, start, limit)
			if err != nil {
				return err
			}
			return emitJSON(issues)
		},
	}
	cmd.Flags().IntVar(&start, "start", 0, "pagination start offset")
	cmd.Flags().IntVar(&limit, "limit", 50, "pagination limit")
	return cmd
}

func issueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "issue <KEY>",
		Short: "Show full issue details (includes embedded comments)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			issue, err := c.GetIssue(args[0])
			if err != nil {
				return err
			}
			return emitJSON(issue)
		},
	}
}

func transitionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transitions <KEY>",
		Short: "List available workflow transitions for an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			ts, err := c.GetTransitions(args[0])
			if err != nil {
				return err
			}
			return emitJSON(ts)
		},
	}
}

func commentCmd() *cobra.Command {
	var internal bool
	cmd := &cobra.Command{
		Use:   "comment <KEY> <body>",
		Short: "Add a comment to an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			if internal {
				return c.AddInternalComment(args[0], args[1])
			}
			return c.AddComment(args[0], args[1])
		},
	}
	cmd.Flags().BoolVar(&internal, "internal", false, "post as an internal-only comment")
	return cmd
}

func transitionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transition <KEY> <status-name>",
		Short: "Transition issue to a target status by name",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			t, err := c.FindTransitionByStatusName(args[0], args[1])
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("no transition to status %q available on %s", args[1], args[0])
			}
			return c.ExecuteTransition(args[0], t.ID)
		},
	}
}

func assignCmd() *cobra.Command {
	var unassign bool
	cmd := &cobra.Command{
		Use:   "assign <KEY> [username]",
		Short: "Assign issue to user (or --unassign)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			if unassign {
				return c.AssignIssue(args[0], nil)
			}
			if len(args) < 2 {
				return fmt.Errorf("username required (or pass --unassign)")
			}
			user, err := c.GetUserByUsername(args[1])
			if err != nil {
				return err
			}
			return c.AssignIssue(args[0], user)
		},
	}
	cmd.Flags().BoolVar(&unassign, "unassign", false, "unassign the issue")
	return cmd
}

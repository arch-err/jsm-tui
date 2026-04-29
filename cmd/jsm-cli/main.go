package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/arch-err/jsm-tui/internal/config"
	"github.com/arch-err/jsm-tui/internal/jira"
)

var outputFormat string

func main() {
	root := &cobra.Command{
		Use:           "jsm-cli",
		Short:         "Non-interactive CLI for Jira Service Management",
		Long:          "CLI for Jira Service Management. Clean text output by default, full JSON with -o json.\nReads config from ~/.config/jsm-tui/config.yaml.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format: text or json")

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

func wantJSON() bool {
	return strings.EqualFold(outputFormat, "json")
}

func emitJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func shortDate(raw string) string {
	for _, layout := range []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z0700",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

func displayName(u *jira.User) string {
	if u == nil {
		return "(unassigned)"
	}
	return u.DisplayName
}

func requestTypeName(crt *jira.CustomerRequestType) string {
	if crt != nil && crt.RequestType != nil {
		return crt.RequestType.Name
	}
	return ""
}

// --- commands ---

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
			if wantJSON() {
				return emitJSON(user)
			}
			fmt.Printf("%s <%s>\n", user.DisplayName, user.EmailAddress)
			return nil
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
			if wantJSON() {
				return emitJSON(qs)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, q := range qs {
				fav := ""
				if q.IsFavorite {
					fav = "*"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", q.ID, q.Name, fav)
			}
			return w.Flush()
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
			if wantJSON() {
				return emitJSON(issues)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, iss := range issues {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					iss.Key,
					iss.Fields.Status.Name,
					displayName(iss.Fields.Assignee),
					iss.Fields.Summary,
				)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVar(&start, "start", 0, "pagination start offset")
	cmd.Flags().IntVar(&limit, "limit", 50, "pagination limit")
	return cmd
}

func issueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "issue <KEY>",
		Short: "Show full issue details including proforma forms and attachments",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, _, err := loadClient()
			if err != nil {
				return err
			}
			detail, err := c.GetIssueDetail(args[0])
			if err != nil {
				return err
			}
			if wantJSON() {
				return emitJSON(detail)
			}
			printIssueText(detail)
			return nil
		},
	}
}

func printIssueText(d *jira.IssueDetail) {
	f := d.Fields
	rt := requestTypeName(f.CustomerRequestType)

	fmt.Printf("%s  %s\n", d.Key, f.Summary)
	fmt.Printf("Status:    %s\n", f.Status.Name)
	fmt.Printf("Priority:  %s\n", f.Priority.Name)
	fmt.Printf("Assignee:  %s\n", displayName(f.Assignee))
	fmt.Printf("Reporter:  %s\n", displayName(f.Reporter))
	typeLine := f.IssueType.Name
	if rt != "" {
		typeLine += " (" + rt + ")"
	}
	fmt.Printf("Type:      %s\n", typeLine)
	fmt.Printf("Created:   %s\n", shortDate(f.Created))
	fmt.Printf("Updated:   %s\n", shortDate(f.Updated))

	if f.Description != "" {
		fmt.Printf("\n--- Description ---\n%s\n", f.Description)
	}

	for _, form := range d.ProformaForms {
		fmt.Printf("\n--- Form: %s ---\n", form.Name)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, field := range form.Fields {
			if field.Label == "" {
				continue
			}
			fmt.Fprintf(w, "  %s:\t%s\n", field.Label, field.Answer)
		}
		w.Flush()
	}

	if len(f.Attachment) > 0 {
		fmt.Printf("\n--- Attachments (%d) ---\n", len(f.Attachment))
		for _, a := range f.Attachment {
			fmt.Printf("  %s  (%s, %s)\n", a.Filename, humanSize(a.Size), a.MimeType)
		}
	}

	if len(f.Comment.Comments) > 0 {
		fmt.Printf("\n--- Comments (%d) ---\n", len(f.Comment.Comments))
		for _, c := range f.Comment.Comments {
			fmt.Printf("  %s  %s:\n", shortDate(c.GetCreated()), c.Author.DisplayName)
			for _, line := range strings.Split(c.Body, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	}
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
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
			if wantJSON() {
				return emitJSON(ts)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, t := range ts {
				fmt.Fprintf(w, "%s\t→ %s\n", t.Name, t.To.Name)
			}
			return w.Flush()
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

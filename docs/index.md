# jsm-tui

A terminal user interface (TUI) for Jira Service Management built with Go and Bubbletea.

## Overview

jsm-tui provides a fast, keyboard-driven interface for managing Jira Service Desk tickets directly from your terminal. Browse queues, view ticket details, transition issues, and add comments without leaving the command line.

## Features

- **Queue Browsing** - View all Service Desk queues for your project
- **Issue Lists** - Browse issues within a queue with key, summary, status, priority, and assignee
- **Ticket Details** - View full issue information including description, comments, and SLA data
- **Workflow Transitions** - Move tickets through workflow states
- **Comment Management** - Add comments to tickets
- **Keyboard Navigation** - Vim-style keybindings (j/k) and arrow keys
- **Fast & Lightweight** - Compiled Go binary with minimal dependencies

## Quick Start

1. [Install](installation.md) jsm-tui
2. [Configure](configuration.md) your Jira connection
3. Run `jsm-tui` to start browsing

## Requirements

- Jira Data Center instance
- Personal Access Token (PAT) or basic auth credentials
- Service Desk project key

## Navigation Flow

```
Queues → Issue List → Issue Detail
         ↑            ↓
         └────────────┴─→ Transition / Comment
```

## Key Features

### Queue List View
Browse all queues in your Service Desk project and select one to view its issues.

### Issue List View
See a table of issues with:
- Issue key
- Summary
- Status (color-coded)
- Priority (color-coded)
- Assignee

### Issue Detail View
View complete ticket information:
- Full description
- Current status and priority
- Assignee and reporter
- Creation and update timestamps
- All comments with authors and timestamps
- Available actions (transition, add comment)

### Transition View
Select from available workflow transitions and execute them to move tickets through states.

### Comment View
Add comments to tickets using a multi-line text input.

## Technology Stack

- **Language**: Go
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **Jira API**: REST API v2 (Jira Data Center)

## Contributing

Contributions are welcome! Please check the [GitHub repository](https://github.com/arch-err/jsm-tui) for issues and pull requests.

## License

See [LICENSE](https://github.com/arch-err/jsm-tui/blob/main/LICENSE) for details.

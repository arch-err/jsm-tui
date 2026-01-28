# Usage

This guide covers how to navigate and use jsm-tui.

## Starting the Application

Simply run:

```bash
jsm-tui
```

The application will load your configuration from `~/.config/jsm-tui/config.yaml` and connect to your Jira instance.

## Navigation Overview

jsm-tui uses a hierarchical navigation structure:

```
Queues → Issue List → Issue Detail → Actions
```

You can always go back to the previous view using ++esc++.

## Keyboard Shortcuts

### Global Keybindings

| Key | Action |
|-----|--------|
| ++q++ | Quit (from queue list view only) |
| ++ctrl+c++ | Force quit from anywhere |
| ++esc++ | Go back to previous view / Cancel action |
| ++r++ | Refresh current view |
| ++question++ | Toggle help (coming soon) |

### Navigation Keybindings

| Key | Action |
|-----|--------|
| ++up++ / ++k++ | Move selection up |
| ++down++ / ++j++ | Move selection down |
| ++enter++ | Select / Open item |
| ++page-up++ | Scroll up one page |
| ++page-down++ | Scroll down one page |

### Issue Detail View

| Key | Action |
|-----|--------|
| ++t++ | Open transition menu |
| ++c++ | Add a comment |
| ++esc++ | Return to issue list |
| ++r++ | Refresh issue details |

### Comment View

| Key | Action |
|-----|--------|
| ++ctrl+s++ | Submit comment |
| ++esc++ | Cancel and return |

## Views Explained

### 1. Queue List View

The first view you see after launching jsm-tui.

**Features:**
- Lists all queues in your Service Desk project
- Shows queue name
- Navigate with arrow keys or j/k
- Press ++enter++ to open a queue

**Available Actions:**
- ++up++ / ++k++ - Move selection up
- ++down++ / ++j++ - Move selection down
- ++enter++ - Open selected queue
- ++r++ - Refresh queue list
- ++q++ - Quit application

### 2. Issue List View

Shows all issues in the selected queue.

**Features:**
- Table view with columns: Key, Summary, Status, Priority, Assignee
- Color-coded status (blue = open, yellow = in progress, green = done)
- Color-coded priority (red = high, yellow = medium, blue = low)
- Truncated text for readability

**Available Actions:**
- ++up++ / ++k++ - Move selection up
- ++down++ / ++j++ - Move selection down
- ++enter++ - Open issue details
- ++esc++ - Return to queue list
- ++r++ - Refresh issue list

### 3. Issue Detail View

Displays complete information about a single issue.

**Features:**
- Issue key and summary
- Status, priority, assignee, reporter
- Issue type
- Creation and update timestamps
- Full description
- All comments with authors and timestamps
- Scrollable viewport for long content

**Available Actions:**
- ++up++ / ++k++ / ++down++ / ++j++ - Scroll content
- ++page-up++ / ++page-down++ - Scroll by page
- ++t++ - Open transition menu
- ++c++ - Add a comment
- ++esc++ - Return to issue list
- ++r++ - Refresh issue details

### 4. Transition View

Select and execute workflow transitions.

**Features:**
- Lists all available transitions for the current issue
- Shows transition name and target status
- Execute transitions to move issues through workflow

**Available Actions:**
- ++up++ / ++k++ - Move selection up
- ++down++ / ++j++ - Move selection down
- ++enter++ - Execute selected transition
- ++esc++ - Cancel and return to issue detail

### 5. Comment View

Add comments to issues.

**Features:**
- Multi-line text input
- Submit comments to the issue

**Available Actions:**
- Type your comment
- ++ctrl+s++ - Submit comment
- ++esc++ - Cancel and return to issue detail

## Common Workflows

### Viewing a Ticket

1. Launch jsm-tui
2. Navigate to desired queue using ++down++ / ++j++
3. Press ++enter++ to open the queue
4. Navigate to desired issue using ++down++ / ++j++
5. Press ++enter++ to view issue details
6. Scroll through description and comments

### Transitioning a Ticket

1. Open issue details (see above)
2. Press ++t++ to open transition menu
3. Navigate to desired transition
4. Press ++enter++ to execute
5. Issue details refresh automatically with new status

### Adding a Comment

1. Open issue details (see above)
2. Press ++c++ to open comment view
3. Type your comment
4. Press ++ctrl+s++ to submit
5. Issue details refresh automatically with new comment

### Searching for an Issue

Currently, jsm-tui doesn't have built-in search. To find an issue:

1. Navigate to the appropriate queue
2. Issues are listed in the order returned by Jira
3. Use ++j++ / ++k++ to scroll through the list
4. Look for the issue key or summary

## Tips & Tricks

### Vim-style Navigation

If you're familiar with Vim, jsm-tui uses similar keybindings:

- ++j++ = down
- ++k++ = up
- ++esc++ = back/cancel

### Quick Queue Switching

To switch queues quickly:

1. From issue list: press ++esc++ to return to queues
2. Navigate to new queue
3. Press ++enter++ to open it

### Refreshing Data

If you make changes in Jira's web UI or another client, press ++r++ to refresh the current view and see the latest data.

### Keyboard-Only Workflow

jsm-tui is designed for keyboard-only operation:

- No mouse required
- Fast navigation with j/k keys
- All actions accessible via keyboard shortcuts

## Troubleshooting

### Application Hangs on Startup

- Check your network connection
- Verify your Jira URL in the config is correct
- Ensure your credentials are valid

### Empty Queue List

- Verify the project key in your config
- Ensure your user has access to the Service Desk project
- Check that queues exist in the project

### Can't Transition Issues

- Verify your user has permission to transition issues
- Ensure the issue type supports the desired transition
- Check workflow configuration in Jira

### Comments Not Appearing

- Wait a moment and press ++r++ to refresh
- Verify the comment was submitted successfully
- Check Jira's web UI to confirm the comment exists

## Next Steps

- Explore the [configuration options](configuration.md)
- Check the [GitHub repository](https://github.com/arch-err/jsm-tui) for updates
- Report issues or request features on GitHub

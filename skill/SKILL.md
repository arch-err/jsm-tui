---
name: jsm
description: CLI for Jira Service Management. Use when the user mentions Jira, JSM, the service desk, tickets, queues, or an issue key like __PROJECT_KEY__-1234.
when_to_use: User talks about Jira tickets, queues, "my issues", "my queue", or any __PROJECT_KEY__-NUMBER reference.
---

# jsm

> Templated skill. If you see literal `__PLACEHOLDER__` strings, install was incomplete — re-run the flow in `docs/agent-usage.md` from the upstream repo.

## User identity

- Name: __USER_DISPLAY_NAME__
- Jira username: __JIRA_USERNAME__
- Default project: __PROJECT_KEY__
- Jira URL: __JIRA_URL__

## Interpretation rules

- "my issues" / "my queue" / "what's on me" → assignee = `__JIRA_USERNAME__`.
- "I commented" / "I assigned" → actions by `__JIRA_USERNAME__`.
- Bare numeric key (e.g. `1234`) → `__PROJECT_KEY__-1234`.
- "the queue" with no name → ask which one, or use a favorite.

## Tool

`jsm-cli` — JSON-output CLI. Call via the bash tool.

```bash
jsm-cli --help               # all commands
jsm-cli <command> --help     # flags + args for one command
jsm-cli me                   # verify auth (run first each session)
```

If `jsm-cli me` errors:
- `command not found` → `go install github.com/arch-err/jsm-tui/cmd/jsm-cli@latest`
- `failed to read config` → user re-bootstraps via `docs/agent-usage.md`
- `HTTP 401` / `HTTP 403` → tell the user, do not retry blindly

## Subcommand cheat sheet

| Command | Reads | Writes |
|---|---|---|
| `me` | ✓ | |
| `queues` | ✓ | |
| `queue <name>` | ✓ | |
| `issue <KEY>` | ✓ | |
| `transitions <KEY>` | ✓ | |
| `comment <KEY> <body> [--internal]` | | ✓ |
| `transition <KEY> <status>` | | ✓ |
| `assign <KEY> <user>` / `--unassign` | | ✓ |

Run `jsm-cli <command> --help` for flags and args.

## Hard rules

1. **Read before write.** `jsm-cli issue <KEY>` before any comment/transition/assign — confirms existence + current state.
2. **List transitions before transitioning.** `jsm-cli transitions <KEY>`. Pick by the `to.name` field, NOT the transition's own `name`. The `transition` command matches `to.name`.
3. **Comments default public** (customer-visible). Use `--internal` only when asked for a staff-only note.
4. **Never invent issue keys.** Always confirm with `jsm-cli issue <KEY>` first.
5. **No bulk ops.** One issue per command. No loops without explicit instruction.
6. **Write commands silent on success.** Empty stdout + zero exit = ok. Non-zero exit + stderr = failed.

## Patterns

Issue summary:
```bash
jsm-cli issue X | jq '{key, summary: .fields.summary, status: .fields.status.name, assignee: .fields.assignee.displayName}'
```

My queue (filter Main if no per-user queue exists):
```bash
jsm-cli queue "Main" | jq '.[] | select(.fields.assignee.name == "__JIRA_USERNAME__")'
```

Transition by target status:
```bash
jsm-cli transitions X    # find target name in `.to.name`
jsm-cli transition X "In Progress"
```

## Limitations

No bulk update, no JQL, no cross-project, no attachments, no proforma editing. Web UI at `__JIRA_URL__` for any of those.

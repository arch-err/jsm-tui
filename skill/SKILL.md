---
name: jsm
description: CLI for Jira Service Management. Use when the user mentions Jira, JSM, the service desk, tickets, queues, or an issue key like __PROJECT_KEY__-1234.
when_to_use: User talks about Jira tickets, queues, "my issues", "my queue", or any __PROJECT_KEY__-NUMBER reference.
---

# jsm

> Use **`jsm-cli`** (the non-interactive JSON CLI). Never `jsm-tui` (that's an interactive Bubbletea TUI for humans, you can't drive it). The repo is named `jsm-tui` because the TUI shipped first; the binary you want is `jsm-cli`.
>
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

`jsm-cli` — CLI for JSM. Clean text output by default, full JSON with `-o json`. Call via the bash tool.

```bash
jsm-cli --help               # all commands
jsm-cli <command> --help     # flags + args for one command
jsm-cli me                   # verify auth (run first each session)
jsm-cli issue FKS-1234       # concise text summary
jsm-cli issue FKS-1234 -o json  # full JSON with proforma + attachments
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
| `issue <KEY>` | ✓ | | includes proforma forms + attachments |
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

## Output format

All read commands default to concise text. Add `-o json` for full JSON.

- `jsm-cli issue X` → human-readable summary with proforma forms, attachments, comments
- `jsm-cli issue X -o json` → full JSON including `proformaForms` array at top level
- `jsm-cli queue "Main"` → tabular: KEY, STATUS, ASSIGNEE, SUMMARY
- `jsm-cli queue "Main" -o json` → full issue JSON array
- `jsm-cli transitions X` → tabular: TRANSITION → TARGET STATUS
- Write commands (`comment`, `transition`, `assign`) are silent on success regardless of `-o`

## Patterns

Issue summary (text mode — no jq needed):
```bash
jsm-cli issue X
```

Full JSON for scripting:
```bash
jsm-cli issue X -o json | jq '.proformaForms[0].Fields[] | select(.Label != "")'
```

My queue (filter Main if no per-user queue exists):
```bash
jsm-cli queue "Main" -o json | jq '.[] | select(.fields.assignee.name == "__JIRA_USERNAME__")'
```

Transition by target status:
```bash
jsm-cli transitions X    # find target name in the → column
jsm-cli transition X "In Progress"
```

## Limitations

No bulk update, no JQL, no cross-project, no attachment download, no proforma editing. Web UI at `__JIRA_URL__` for any of those.

---
name: jsm
description: CLI tool for Jira Service Management. Read tickets, list queues, comment, transition states, assign users. Use whenever the user mentions Jira, JSM, the service desk, tickets, queues, or uses an issue key like __PROJECT_KEY__-1234.
when_to_use: User talks about Jira tickets, the service desk, "my queue", "my tickets", "the ticket", any string matching __PROJECT_KEY__-NUMBER, or asks about service desk work.
---

# jsm — Jira Service Management

> This skill is templated. The placeholders below have been replaced with this user's values during install. If you see literal `__PLACEHOLDER__` strings anywhere, the install was incomplete — re-run the bootstrap from `docs/agent-usage.md` in the upstream repo.

## User identity

- **Display name**: __USER_DISPLAY_NAME__
- **Jira username / account ID**: __JIRA_USERNAME__
- **Default project key**: __PROJECT_KEY__
- **Jira URL**: __JIRA_URL__

## What "my" / "me" / "I" means

When the user says:
- "my issues", "my tickets", "my queue", "what's on me", "what do I have" → issues where assignee = `__JIRA_USERNAME__`.
- "I commented", "I assigned", "I transitioned" → actions taken by `__JIRA_USERNAME__`.
- An issue key without project prefix (e.g. `1234`) → assume `__PROJECT_KEY__-1234`.
- "the queue" without naming one → ask which queue, or default to a favorite if the user has one.

## Tool

Single CLI binary: `jsm-cli`. Outputs JSON. Call it via the bash tool.

## First check (every session)

Before any other command, verify the tool works:

```bash
jsm-cli me
```

Expect JSON with `name`, `displayName`, `emailAddress`. If it errors:
- `command not found` → install with `go install github.com/arch-err/jsm-tui/cmd/jsm-cli@latest`.
- `failed to read config file` → tell the user, point them at `docs/agent-usage.md` for re-bootstrap.
- `HTTP 401` → token expired; ask the user for a new one.
- `HTTP 403` → permission issue; tell the user, do not retry blindly.

## Read commands (safe to run anytime)

```bash
jsm-cli me                                      # current user
jsm-cli queues                                   # list queues for project __PROJECT_KEY__
jsm-cli queue "Main"                             # issues in queue named "Main"
jsm-cli issue __PROJECT_KEY__-1234               # full issue details (replace 1234)
jsm-cli transitions __PROJECT_KEY__-1234         # available transitions for the issue
```

## Write commands (only when user explicitly asks)

```bash
jsm-cli comment __PROJECT_KEY__-1234 "looking into this"
jsm-cli comment __PROJECT_KEY__-1234 --internal "duplicate of HELP-1198"
jsm-cli transition __PROJECT_KEY__-1234 "In Progress"
jsm-cli assign __PROJECT_KEY__-1234 __JIRA_USERNAME__       # assign to the user
jsm-cli assign __PROJECT_KEY__-1234 jdoe                     # assign to someone else
jsm-cli assign __PROJECT_KEY__-1234 --unassign               # clear assignee
```

## Common tasks

### "Show me ticket X" / "What's the status of X"

```bash
jsm-cli issue X | jq '{key, summary: .fields.summary, status: .fields.status.name, assignee: .fields.assignee.displayName}'
```

### "What's in my queue?" / "What's assigned to me?"

First, see if a queue named for the user exists:

```bash
jsm-cli queues | jq -r '.[].name'
```

If there's an "Assigned to me" or similar, use it:

```bash
jsm-cli queue "Assigned to me" | jq -r '.[] | "\(.key)\t\(.fields.summary)\t\(.fields.status.name)"'
```

If not, filter the main queue:

```bash
jsm-cli queue "Main" | jq '.[] | select(.fields.assignee.name == "__JIRA_USERNAME__")'
```

### "Move X to In Progress" (or any status change)

ALWAYS list transitions first — workflows differ depending on current status:

```bash
jsm-cli transitions X
```

Output is an array. Each entry has a `to.name` field. Pick the entry whose `to.name` matches the target status. THEN transition:

```bash
jsm-cli transition X "In Progress"
```

The `transition` command matches by **target status name** (the `to.name` field), not by the transition's own `name`.

### "Comment on X saying Y"

Default is **public** (visible to the customer):

```bash
jsm-cli comment X "Y"
```

For **internal-only** notes (visible only to staff, not the customer):

```bash
jsm-cli comment X --internal "Y"
```

### "Take X" / "Assign X to me"

```bash
jsm-cli assign X __JIRA_USERNAME__
```

### "Unassign X" / "Drop X"

```bash
jsm-cli assign X --unassign
```

## Hard rules

1. **Read before write.** Run `jsm-cli issue X` before commenting, transitioning, or assigning. Confirms the issue exists and shows current state.
2. **Read transitions before transitioning.** Available transitions depend on current status. What worked yesterday may not work now.
3. **Default comments are public.** Use `--internal` for staff-only notes. Don't post internal-only by default — be explicit when the user asks for one.
4. **No bulk operations.** One issue per command. Never loop over many issues without explicit user instruction.
5. **Issue keys are case-sensitive in some Jira deployments.** Use exactly as the user gives them.
6. **Write commands are silent on success.** No output = success. Non-zero exit + stderr message = failure.
7. **Never invent issue keys.** If the user mentions one you haven't read, run `jsm-cli issue <KEY>` first to confirm it exists.

## Common errors

| Error message | Likely cause | Fix |
|---|---|---|
| `command not found: jsm-cli` | not installed | `go install github.com/arch-err/jsm-tui/cmd/jsm-cli@latest` |
| `failed to read config file` | no config at `~/.config/jsm-tui/config.yaml` | re-run install bootstrap (see `docs/agent-usage.md` in repo) |
| `HTTP 401` | token expired or wrong | ask user for a new token, rewrite config |
| `HTTP 403` | token works but no permission for this issue/project | tell user, do NOT retry |
| `HTTP 404 ... issue` | wrong key or no read access | verify the key with the user |
| `queue not found: "X"` | typo in queue name | run `jsm-cli queues` to see real names |
| `no transition to status "X" available` | target status not reachable from current state | run `jsm-cli transitions <KEY>` to see what IS available |

## What this tool does NOT support

- No bulk update across multiple issues.
- No JQL search (only queue-by-name).
- No cross-project ops (single project per config).
- No proforma form editing (read-only on those).
- No attachment upload or download.
- No transition with field updates (some workflows require fields like resolution — those will fail).

If the user asks for any of the above, tell them this tool doesn't support it and suggest the Jira web UI at `__JIRA_URL__`.

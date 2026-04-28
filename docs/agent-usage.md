# jsm-cli — agent usage guide

Non-interactive JSON CLI for Jira Service Management. Read tickets, list queues, transition states, comment, assign — all from the shell, all output as JSON to stdout.

Sister binary to `jsm-tui` (interactive TUI). Both share the same config file and the same internal API client. If `jsm-tui` works against your Jira, `jsm-cli` will too.

## Bootstrap (first install)

When an agent encounters `jsm-cli` for the first time on a new system, walk the user through this flow before invoking any command. The answers populate `~/.config/jsm-tui/config.yaml` and feed back into the agent's own memory for future sessions.

### Questions to ask the user

| # | Question | Notes |
|---|---|---|
| 1 | What's your Jira URL? | e.g. `https://jira.example.com`. Trailing slashes are fine — they're stripped. |
| 2 | PAT or basic auth? | PAT recommended for Jira Server / Data Center. Basic only if PAT is unavailable. |
| 3 | (PAT) What's your token? | Generated in Jira under user profile → Personal Access Tokens. |
| 3 | (basic) Username + password? | Storing a password in plaintext config — only if PAT isn't an option. |
| 4 | Service desk project key? | The prefix in any issue key. `HELP-1234` → key is `HELP`. |
| 5 | What's your Jira username / account ID? | Used to identify "me" in queue listings. Optional but useful. |
| 6 | Any favorite queue names? | Optional. Pinned in TUI; useful for the agent to know default reading targets. |

### What the agent writes

After collecting answers, write `~/.config/jsm-tui/config.yaml` with mode `0600` (file contains a token):

```yaml
url: <answer 1>
auth:
  type: pat            # or 'basic'
  token: <answer 3>    # if pat
  # username: <answer 3a>   # if basic
  # password: <answer 3b>   # if basic
project: <answer 4>
username: <answer 5>   # optional, omit if not given
queues:
  favorites:           # optional, omit if not given
    - <answer 6 name 1>
    - <answer 6 name 2>
```

### Verify connectivity

```bash
jsm-cli me
```

Expect a JSON object with `name`, `displayName`, `emailAddress`. If this errors, fix before continuing — use the [Troubleshooting](#troubleshooting) table below to map error → cause.

### What the agent persists in its own memory

Separate from the config file, the agent should remember:

- **Project key** — so user phrases like "my queue" resolve to the right project without re-asking.
- **Display name + username** — so the agent recognizes self-references in tickets ("assigned to me", "I commented on", etc).
- **Favorite queue names** — for default listings when no queue is specified.
- **Workflow status names** that are meaningful in this user's deployment — e.g. "Waiting for support" → initial, "Resolved" → terminal. Build this knowledge as the agent uses the tool.

These live in the agent's memory layer, not in the repo. The repo skill stays generic; per-user knowledge stays per-user.

### Re-bootstrap

If the user later moves Jira instances, rotates a PAT, or changes default project, re-run the flow and overwrite `~/.config/jsm-tui/config.yaml`. The CLI does not cache anything outside that file, so the rewrite is sufficient. Update the agent's memory layer to match.

## Install

### Via Go

```bash
go install github.com/arch-err/jsm-tui/cmd/jsm-cli@latest
```

Binary lands at `$GOPATH/bin/jsm-cli` (default `~/go/bin/jsm-cli`). Make sure that's on your `PATH`.

### Pre-built binary

Download the latest release archive from the [releases page](https://github.com/arch-err/jsm-tui/releases). Each archive contains both `jsm-tui` and `jsm-cli`.

### From source

```bash
git clone https://github.com/arch-err/jsm-tui.git
cd jsm-tui
go build -o jsm-cli ./cmd/jsm-cli
```

## Configure

`jsm-cli` reads `~/.config/jsm-tui/config.yaml` — the same file `jsm-tui` uses. If you've already configured `jsm-tui`, there is nothing to do.

Minimal config:

```yaml
url: https://your-jira-instance.com
auth:
  type: pat            # or 'basic'
  token: <your-PAT>
project: HELP          # service desk project key
```

For basic auth, replace `token` with `username` and `password`. See [docs/configuration.md](configuration.md) for the full schema.

## Verify

```bash
jsm-cli me
```

Should print a JSON object describing the authenticated user. If it errors, fix auth/connectivity before continuing.

## Commands

All commands print JSON to stdout. Write commands are silent on success and exit non-zero on failure (errors to stderr).

| Command | Reads | Writes | Description |
|---|---|---|---|
| `me` | yes | no | Current authenticated user |
| `queues` | yes | no | List all service desk queues for the configured project |
| `queue <name>` | yes | no | List issues in a queue (case-insensitive name match) |
| `issue <KEY>` | yes | no | Full issue details (comments embedded in `fields.comment`) |
| `transitions <KEY>` | yes | no | Available workflow transitions for an issue |
| `comment <KEY> <body>` | no | yes | Add a comment (`--internal` for internal-only) |
| `transition <KEY> <status>` | no | yes | Transition issue to target status by name |
| `assign <KEY> <user>` | no | yes | Assign to user (`--unassign` to clear) |

### Flags

- `comment --internal` — post an internal-only comment (not visible to the requester on the customer portal).
- `queue --start N --limit M` — paginate. Defaults: `start=0 limit=50`.
- `assign --unassign` — clear the assignee. Username arg is then optional.

## Patterns for agents

### Read-before-write

Workflows are state-dependent. Always list available transitions before transitioning:

```bash
jsm-cli transitions HELP-1234
# inspect the JSON, pick a "to.name" target
jsm-cli transition HELP-1234 "In Progress"
```

The `transition` command matches by **target status name** (the `to.name` field in `transitions` output), not by the transition's own name.

### Composing with jq

Output is consistent JSON, pipe freely:

```bash
# All open issue keys in the "Main" queue
jsm-cli queue "Main" --limit 50 | jq -r '.[].key'

# Just the status of one issue
jsm-cli issue HELP-1234 | jq -r '.fields.status.name'

# All transition target names available right now
jsm-cli transitions HELP-1234 | jq -r '.[].to.name'
```

### Error handling

Check exit code on writes:

```bash
if jsm-cli comment HELP-1234 "investigating" 2>/tmp/err; then
  echo ok
else
  echo "failed: $(cat /tmp/err)"
fi
```

Errors are formatted as `error: <message>` on stderr. HTTP errors include status code and response body.

### Internal vs external comments

```bash
# Visible to the customer (default)
jsm-cli comment HELP-1234 "We're looking into it."

# Internal note — only visible to staff
jsm-cli comment HELP-1234 --internal "Likely a duplicate of HELP-1198."
```

When acting as an agent, prefer `--internal` for thinking-out-loud notes meant for human staff. Reserve customer-visible comments for content the user has explicitly asked to send.

## Conventions an agent should follow

- **Issue keys are project-prefixed**: `HELP-1234`, `FKS-2718`. Keys are case-sensitive in some Jira deployments — pass them exactly as the user gives them.
- **Stay defensive on writes**: read first, confirm the issue exists and is in the expected state, then write.
- **Don't bulk-read by default**: `queue` and `queues` can be expensive on busy projects. Prefer `issue <KEY>` when you have a key.
- **Don't transition without listing**: a transition that worked yesterday may not be available today if the issue moved states.
- **Idempotency is not guaranteed**: `comment` will create a new comment every time you call it. Don't retry blindly on uncertain failures.

## Limitations

- **Single-project**: the configured `project` is global. To work across projects, swap configs (point `XDG_CONFIG_HOME` at a different dir) or run with a different config file.
- **No JQL**: there is no arbitrary JQL search command yet. Use queue-by-name.
- **No bulk operations**: one issue per command.
- **No MCP server**: this is a plain CLI. Wrap it behind your agent's bash/shell tool, or write a thin MCP layer if you need structured tool-calling.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `error: failed to read config file` | No config at `~/.config/jsm-tui/config.yaml` — see [Configure](#configure) |
| `error: HTTP 401` | Token expired, wrong, or wrong auth type for your Jira |
| `error: HTTP 403` | Token valid but lacks permission on this project/issue |
| `error: HTTP 404 ... issue` | Issue key wrong, or you don't have read access |
| `error: queue not found` | Queue name typo — run `jsm-cli queues` to see exact names |
| `error: no transition to status "X" available` | Workflow doesn't allow that transition from current state — run `transitions <KEY>` to see valid targets |

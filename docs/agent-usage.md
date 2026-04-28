# jsm-cli — agent install guide

> **You want `jsm-cli`, not `jsm-tui`.** The repo is named `jsm-tui` because the human-facing TUI shipped first. As an LLM agent, you should install and use **`jsm-cli`** — the non-interactive JSON CLI. Never run `jsm-tui` from an agent context: it opens a Bubbletea TUI that needs interactive keyboard input you can't provide.

This guide is for **AI coding agents** (Claude Code, Cursor, Codex, Aider, pi, custom harnesses) installing `jsm-cli` on behalf of a user for the first time.

If you're a human, see [README.md](../README.md).

## What this is

`jsm-cli` is a non-interactive JSON CLI for Jira Service Management. The agent shells out to it. It outputs JSON to stdout, errors to stderr, exits non-zero on failure.

Sister binary to `jsm-tui` (interactive TUI for humans — DO NOT install or run as an agent). Same config, same API client, completely different invocation model.

## Install flow (run once per system)

The first time an agent encounters this tool on a system, walk the user through these six steps. Subsequent sessions skip to "Verify" and use the installed skill.

### Step 1 — install the binary

```bash
go install github.com/arch-err/jsm-tui/cmd/jsm-cli@latest
```

Confirm:

```bash
command -v jsm-cli
```

If `go` isn't installed, point the user at the [releases page](https://github.com/arch-err/jsm-tui/releases) for prebuilt binaries.

### Step 2 — ask the user the bootstrap questions

Collect every value below. Some are written to the config file, some are substituted into the skill, some get persisted in the agent's own memory.

| Variable | Question to ask | Used in | Example |
|---|---|---|---|
| `__JIRA_URL__` | "What's your Jira URL?" | config + skill | `https://jira.example.com` |
| auth type | "Personal Access Token (PAT) or basic auth?" — recommend PAT | config | `pat` |
| `__JIRA_TOKEN__` | (PAT only) "What's your token?" | config only | `NjE0MDE1...` |
| basic username | (basic only) "Your Jira login username?" | config only | `jdoe` |
| basic password | (basic only) "Your Jira login password?" | config only | (sensitive) |
| `__PROJECT_KEY__` | "Service desk project key? (the prefix in any issue key — `HELP-1234` → key is `HELP`)" | config + skill | `HELP` |
| `__JIRA_USERNAME__` | "Your Jira username or account ID? (used to identify you in tickets)" | config + skill | `jdoe` (Server) or `5b10ac8d82e05b22cc7d4ef5` (Cloud account ID) |
| `__USER_DISPLAY_NAME__` | "Your full name as it appears on Jira tickets?" | skill | `Jane Doe` |
| favorite queues | "Any favorite queue names? (optional, comma-separated)" | config | `Main, Assigned to me` |

**Never substitute the token or password into the skill.** Secrets stay in the config file only.

### Step 3 — write the config file

Write `~/.config/jsm-tui/config.yaml` with mode `0600` (file holds the token):

For PAT auth:

```yaml
url: <__JIRA_URL__>
auth:
  type: pat
  token: <__JIRA_TOKEN__>
project: <__PROJECT_KEY__>
username: <__JIRA_USERNAME__>
queues:
  favorites:
    - <queue 1>
    - <queue 2>
```

For basic auth:

```yaml
url: <__JIRA_URL__>
auth:
  type: basic
  username: <basic username>
  password: <basic password>
project: <__PROJECT_KEY__>
username: <__JIRA_USERNAME__>
queues:
  favorites:
    - <queue 1>
```

After writing:

```bash
chmod 600 ~/.config/jsm-tui/config.yaml
```

### Step 4 — install the skill (template substitution)

Download the skill template:

```bash
curl -fsSL https://raw.githubusercontent.com/arch-err/jsm-tui/main/skill/SKILL.md -o /tmp/jsm-skill-template.md
```

Or use `skill/SKILL.md` directly if you have a local clone of the repo.

Substitute the placeholders with the user's values:

```bash
sed -i \
  -e "s|__JIRA_URL__|<__JIRA_URL__>|g" \
  -e "s|__PROJECT_KEY__|<__PROJECT_KEY__>|g" \
  -e "s|__JIRA_USERNAME__|<__JIRA_USERNAME__>|g" \
  -e "s|__USER_DISPLAY_NAME__|<__USER_DISPLAY_NAME__>|g" \
  /tmp/jsm-skill-template.md
```

Verify no placeholders remain:

```bash
grep -E '__[A-Z_]+__' /tmp/jsm-skill-template.md && echo "MISSING SUBSTITUTIONS" || echo "ok"
```

Save the customized skill to wherever your harness loads instruction files from:

| Harness | Path |
|---|---|
| Claude Code | `~/.claude/skills/jsm/SKILL.md` |
| Claude Code (workspace) | `.claude/skills/jsm/SKILL.md` in the project |
| Cursor | `.cursor/rules/jsm.mdc` in the project |
| Codex | `~/.codex/skills/jsm/SKILL.md` |
| Aider | `~/.config/aider/jsm-skill.md` (manual reference) |
| Other | whatever your platform reads instruction files from |

If your harness has no skill system, save the customized file somewhere durable and reference it manually each session.

### Step 5 — verify

```bash
jsm-cli me
```

Expect a JSON object with `name`, `displayName`, `emailAddress`. If it errors, see the troubleshooting table below before continuing. Do not proceed with anything else until this works.

### Step 6 — persist shorthand in agent memory

Beyond the config and the customized skill, write these into the agent's own memory layer (if it has one):

- **Project key** — so "my queue" resolves to the right project across sessions.
- **Display name + username** — so the agent recognizes self-references.
- **Favorite queue names** — for default listings.
- **Workflow status names** that are meaningful for this deployment — e.g. "Waiting for support" → initial, "Resolved" → terminal. Build this knowledge as the tool gets used.

Memory persists what the skill doesn't — running notes, edge cases, workflow specifics learned over time.

## Re-bootstrap

If the user moves Jira instances, rotates a PAT, or changes default project:

1. Re-run **steps 2 and 3** to overwrite `~/.config/jsm-tui/config.yaml`.
2. Re-run **step 4** to update the installed skill with new placeholders.
3. Re-run **step 5** to verify.

The CLI does not cache anything outside the config file, so the rewrite is sufficient.

## After install

Use the customized skill (loaded by the harness) for ongoing work. It contains the substituted user values and behavior rules ("when user says 'my', they mean assignee = X").

This guide (`docs/agent-usage.md`) is only relevant during install or re-bootstrap.

## Troubleshooting install

| Symptom | Likely cause | Fix |
|---|---|---|
| `go: command not found` | Go not installed | Install Go: `yay -S go` (Arch), `brew install go` (Mac), or [go.dev/dl](https://go.dev/dl/) |
| `command not found: jsm-cli` after install | `~/go/bin` not in `PATH` | `export PATH="$HOME/go/bin:$PATH"` (add to shell rc) |
| `jsm-cli me` errors with TLS / cert | Corporate CA not trusted | Ask user to install corp CA certs system-wide |
| `jsm-cli me` errors with proxy / connection refused | Corporate proxy intercepting | Ensure Jira host is in `NO_PROXY` env var |
| `HTTP 401 Unauthorized` | Wrong or expired token | Re-collect step 2, rewrite step 3 |
| `HTTP 403 Forbidden` | Token valid but lacks project permission | User issue — escalate to user |
| Skill placeholders still visible (`__USER_DISPLAY_NAME__` etc) in agent context | Step 4 substitution incomplete | Re-run step 4 — verify with the `grep` check |

## See also

- [README.md](../README.md) — human install + usage
- [skill/SKILL.md](../skill/SKILL.md) — the templated skill (source of truth for ongoing usage)
- [llms.txt](../llms.txt) — repo entry-point for LLM-driven discovery

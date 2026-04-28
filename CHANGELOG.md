# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-04-28

### Added
- New `jsm-cli` binary — non-interactive JSON CLI cousin of `jsm-tui` for use from agents and scripts. Shares config and API client with the TUI.
- `cmd/jsm-cli/main.go` exposing 8 subcommands: `me`, `queues`, `queue`, `issue`, `transitions`, `comment`, `transition`, `assign`. JSON output to stdout, errors to stderr, non-zero exit on failure.
- `llms.txt` at the repo root following the [llms.txt](https://llmstxt.org/) standard for LLM-discoverable project entry-point.
- `docs/agent-usage.md` — agent install guide. Six-step flow: install binary, ask user the bootstrap questions, write config, customize the skill template (placeholder substitution), verify, persist shorthand in agent memory. Harness-agnostic (Claude, Cursor, Codex, Aider, pi).
- `skill/SKILL.md` — templated agent skill with `__PLACEHOLDER__` markers (`__JIRA_URL__`, `__PROJECT_KEY__`, `__JIRA_USERNAME__`, `__USER_DISPLAY_NAME__`). Customized per-user at install time and saved to wherever the harness loads instruction files from. Day-to-day reference for the agent — generic on disk in this repo, personalized once installed.
- `.goreleaser.yaml`: second build target for `jsm-cli`; release archives now bundle both binaries.

## [0.2.0] - 2026-01-28

### Added
- Color-coded statuses (Escalated=red, In Progress=yellow, Waiting for support=orange, Pending=blue, Waiting for customer=green)
- Assignee coloring (teal=me, white=unassigned, gray=others)
- Dimmed row text for issues assigned to others
- Scrollable issues list with scroll indicators
- `G` keybind to go to bottom of lists
- `gg` keybind (double-tap) to go to top of lists
- `h` keybind to toggle hiding non-favorite queues
- Persistent `hide_non_favorites` setting in config

### Changed
- Restructured config: `favorite_queues` moved to `queues.favorites`
- Favorite queues now display in config-defined order
- Improved column alignment with proper ANSI code handling
- Condensed help text navigation hints
- Hide status column in "All" queue view
- Issues view now receives window dimensions on creation

## [0.1.0] - 2026-01-28

### Added
- Initial release of jsm-tui
- Queue browsing for Service Desk projects
- Issue list view with key, summary, status, priority, and assignee
- Issue detail view with full description, comments, and metadata
- Workflow transition support
- Comment creation on issues
- Vim-style keyboard navigation (j/k)
- Personal Access Token (PAT) authentication support
- Basic authentication support
- Configuration via YAML file (`~/.config/jsm-tui/config.yaml`)
- Color-coded status and priority indicators
- Scrollable viewport for long issue descriptions
- Refresh capability for all views
- Loading spinners for API requests
- Error handling and display
- Help text in status bar
- Cross-platform support (Linux, macOS, Windows)
- Comprehensive documentation with MkDocs
- CI/CD pipeline with GitHub Actions
- Automated releases with GoReleaser

[Unreleased]: https://github.com/arch-err/jsm-tui/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/arch-err/jsm-tui/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/arch-err/jsm-tui/releases/tag/v0.1.0

# jsm-tui

> Jira Service Management TUI for browsing queues, viewing tickets, and taking actions from the terminal

[![Build Status](https://github.com/arch-err/jsm-tui/workflows/Build/badge.svg)](https://github.com/arch-err/jsm-tui/actions)
[![Release](https://github.com/arch-err/jsm-tui/workflows/Release/badge.svg)](https://github.com/arch-err/jsm-tui/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/arch-err/jsm-tui)](https://goreportcard.com/report/github.com/arch-err/jsm-tui)
[![License](https://img.shields.io/github/license/arch-err/jsm-tui)](LICENSE)

A fast, keyboard-driven terminal user interface for Jira Service Management built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Features

- 📋 **Queue Browsing** - View all Service Desk queues
- 🎫 **Issue Management** - Browse, view, and manage tickets
- 🔄 **Workflow Transitions** - Move tickets through workflow states
- 💬 **Comments** - Add comments to issues
- ⌨️ **Keyboard Navigation** - Vim-style keybindings (j/k) and arrow keys
- 🎨 **Color-coded UI** - Status and priority indicators
- ⚡ **Fast & Lightweight** - Compiled binary with minimal dependencies
- 🔐 **Secure** - PAT or basic authentication support

## Installation

### Download Pre-built Binary

Download the latest release from the [releases page](https://github.com/arch-err/jsm-tui/releases).

### Using Go

```bash
go install github.com/arch-err/jsm-tui@latest
```

### Build from Source

```bash
git clone https://github.com/arch-err/jsm-tui.git
cd jsm-tui
go build -o jsm-tui .
```

## Configuration

Create a configuration file at `~/.config/jsm-tui/config.yaml`:

```yaml
url: https://your-jira-instance.com
auth:
  type: pat  # or 'basic'
  token: your-personal-access-token
  # For basic auth, use:
  # username: your-username
  # password: your-password
project: YOUR-PROJECT-KEY
```

See the [configuration documentation](https://arch-err.github.io/jsm-tui/configuration/) for more details.

## Usage

Start the application:

```bash
jsm-tui
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| <kbd>j</kbd> / <kbd>k</kbd> or <kbd>↑</kbd> / <kbd>↓</kbd> | Navigate |
| <kbd>Enter</kbd> | Select / Open |
| <kbd>Esc</kbd> | Go back |
| <kbd>r</kbd> | Refresh |
| <kbd>t</kbd> | Transition (in detail view) |
| <kbd>c</kbd> | Add comment (in detail view) |
| <kbd>q</kbd> | Quit (from queue list) |
| <kbd>Ctrl+C</kbd> | Force quit |

See the [usage documentation](https://arch-err.github.io/jsm-tui/usage/) for detailed information.

## Documentation

Full documentation is available at [https://arch-err.github.io/jsm-tui/](https://arch-err.github.io/jsm-tui/)

- [Installation Guide](https://arch-err.github.io/jsm-tui/installation/)
- [Configuration Reference](https://arch-err.github.io/jsm-tui/configuration/)
- [Usage Guide](https://arch-err.github.io/jsm-tui/usage/)

## Requirements

- Jira Data Center instance
- Personal Access Token (PAT) or basic auth credentials
- Service Desk project key

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **API**: Jira Data Center REST API v2

## Development

### Prerequisites

- Go 1.21 or higher
- Git

### Building

```bash
# Clone the repository
git clone https://github.com/arch-err/jsm-tui.git
cd jsm-tui

# Download dependencies
go mod download

# Build
go build -o jsm-tui .

# Run
./jsm-tui
```

### Testing

```bash
go test ./...
```

### Linting

```bash
go vet ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/) (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

### Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `chore:` - Maintenance tasks
- `refactor:` - Code refactoring
- `test:` - Test changes
- `ci:` - CI/CD changes

## Versioning

This project adheres to [Semantic Versioning](https://semver.org/). For available versions, see the [releases page](https://github.com/arch-err/jsm-tui/releases).

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework
- Inspired by other TUI tools in the Go ecosystem
- Thanks to the Charm community for excellent TUI libraries

## Support

- 📚 [Documentation](https://arch-err.github.io/jsm-tui/)
- 🐛 [Issue Tracker](https://github.com/arch-err/jsm-tui/issues)
- 💬 [Discussions](https://github.com/arch-err/jsm-tui/discussions)

---

**Note**: This tool is designed for Jira Data Center. Jira Cloud support may be added in future releases.

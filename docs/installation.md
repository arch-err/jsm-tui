# Installation

## Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/arch-err/jsm-tui/releases) page.

### Linux

```bash
# Download the latest release (replace VERSION with actual version)
wget https://github.com/arch-err/jsm-tui/releases/download/vVERSION/jsm-tui_VERSION_Linux_x86_64.tar.gz

# Extract
tar -xzf jsm-tui_VERSION_Linux_x86_64.tar.gz

# Move to PATH
sudo mv jsm-tui /usr/local/bin/

# Verify installation
jsm-tui --version
```

### macOS

```bash
# Download the latest release (replace VERSION with actual version)
wget https://github.com/arch-err/jsm-tui/releases/download/vVERSION/jsm-tui_VERSION_Darwin_x86_64.tar.gz

# Extract
tar -xzf jsm-tui_VERSION_Darwin_x86_64.tar.gz

# Move to PATH
sudo mv jsm-tui /usr/local/bin/

# Verify installation
jsm-tui --version
```

### Windows

1. Download the latest `.zip` file from [GitHub Releases](https://github.com/arch-err/jsm-tui/releases)
2. Extract the archive
3. Add the extracted directory to your PATH
4. Run `jsm-tui.exe` from Command Prompt or PowerShell

## Go Install

If you have Go installed (1.21+), you can install directly:

```bash
go install github.com/arch-err/jsm-tui@latest
```

## Building from Source

### Requirements

- Go 1.21 or higher
- Git

### Steps

```bash
# Clone the repository
git clone https://github.com/arch-err/jsm-tui.git
cd jsm-tui

# Download dependencies
go mod download

# Build
go build -o jsm-tui .

# Install (optional)
sudo mv jsm-tui /usr/local/bin/
```

## Verify Installation

After installation, verify that jsm-tui is available:

```bash
jsm-tui --version
```

If the command is not found, ensure the installation directory is in your PATH.

## Next Steps

After installation, you need to [configure](configuration.md) jsm-tui with your Jira credentials and project information.

package tui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) error {
	// Detect Wayland vs X11
	isWayland := os.Getenv("WAYLAND_DISPLAY") != ""

	// Order commands based on display server
	var commands []struct {
		name string
		args []string
	}

	if isWayland {
		// Wayland: prefer wl-copy
		commands = []struct {
			name string
			args []string
		}{
			{"wl-copy", []string{}},
			{"xclip", []string{"-selection", "clipboard"}}, // XWayland fallback
			{"xsel", []string{"--clipboard", "--input"}},
			{"pbcopy", []string{}},
		}
	} else {
		// X11 or macOS
		commands = []struct {
			name string
			args []string
		}{
			{"xclip", []string{"-selection", "clipboard"}},
			{"xsel", []string{"--clipboard", "--input"}},
			{"wl-copy", []string{}},
			{"pbcopy", []string{}},
		}
	}

	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd.name); err == nil {
			c := exec.Command(cmd.name, cmd.args...)
			c.Stdin = strings.NewReader(text)
			return c.Run()
		}
	}

	return nil // Silently fail if no clipboard tool found
}

// OpenInBrowser opens a URL in the default browser
func OpenInBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // Linux and others
		// Try xdg-open first, then common browsers
		browsers := []string{"xdg-open", "x-www-browser", "firefox", "chromium", "google-chrome"}
		for _, browser := range browsers {
			if _, err := exec.LookPath(browser); err == nil {
				cmd = exec.Command(browser, url)
				break
			}
		}
	}

	if cmd == nil {
		return nil // No browser found
	}

	return cmd.Start() // Don't wait for browser to close
}

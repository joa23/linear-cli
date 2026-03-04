package xdg

import (
	"os"
	"path/filepath"
)

// LinearConfigDir returns the Linear config directory.
// Respects $XDG_CONFIG_HOME per the XDG Base Directory Specification.
// Falls back to ~/.config/linear when $XDG_CONFIG_HOME is unset.
func LinearConfigDir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "linear")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "linear")
}

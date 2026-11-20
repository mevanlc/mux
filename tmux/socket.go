package tmux

import (
	"fmt"
	"os"
	"strings"
)

var socketName string

// SetSocket selects a named tmux server. Call before starting the TUI or running
// any commands. An empty name preserves tmux's normal server selection.
func SetSocket(name string) {
	socketName = name
}

// SocketArgs prepends the selected socket option to tmux command arguments.
func SocketArgs(args ...string) []string {
	if socketName == "" {
		return args
	}
	return append([]string{"-L", socketName}, args...)
}

// IsCurrentServer reports whether the selected server owns the current client.
// A client can switch sessions only within its own server.
func IsCurrentServer() (bool, error) {
	current := os.Getenv("TMUX")
	if current == "" {
		return false, nil
	}
	if socketName == "" {
		return true, nil
	}

	// TMUX contains socket-path,server-pid,session-id. Strip the numeric fields
	// from the right because the path itself may contain commas.
	for range 2 {
		idx := strings.LastIndexByte(current, ',')
		if idx < 0 {
			return false, fmt.Errorf("invalid TMUX environment variable")
		}
		current = current[:idx]
	}
	out, err := output("display-message", "-p", "#{socket_path}")
	if err != nil {
		return false, fmt.Errorf("resolve tmux socket: %w", err)
	}
	selected := strings.TrimSuffix(string(out), "\n")
	if current == selected {
		return true, nil
	}

	// The same socket can be spelled differently through symlinked directories
	// (for example /tmp and /private/tmp on macOS).
	currentInfo, currentErr := os.Stat(current)
	selectedInfo, selectedErr := os.Stat(selected)
	return currentErr == nil && selectedErr == nil && os.SameFile(currentInfo, selectedInfo), nil
}

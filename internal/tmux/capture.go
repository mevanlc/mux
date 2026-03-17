package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

func CapturePane(sessionName string) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e").Output()
	if err != nil {
		return "", fmt.Errorf("capture pane %s: %w", sessionName, err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const listFormat = "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_path}"

func ListSessions() ([]Session, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", listFormat).Output()
	if err != nil {
		// tmux returns error when no server is running
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		s, err := parseLine(line)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func parseLine(line string) (Session, error) {
	parts := strings.SplitN(line, "|", 5)
	if len(parts) < 5 {
		return Session{}, fmt.Errorf("unexpected format: %s", line)
	}

	windows, _ := strconv.Atoi(parts[1])
	createdUnix, _ := strconv.ParseInt(parts[2], 10, 64)
	attached, _ := strconv.Atoi(parts[3])

	return Session{
		Name:      parts[0],
		Windows:   windows,
		Created:   time.Unix(createdUnix, 0),
		Attached:  attached > 0,
		Directory: parts[4],
	}, nil
}

func CreateSession(name string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name).Run()
}

func CreateSessionWithDir(name, dir string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir).Run()
}

func KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func RenameSession(oldName, newName string) error {
	return exec.Command("tmux", "rename-session", "-t", oldName, newName).Run()
}

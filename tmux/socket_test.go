package tmux

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSocketSelectionForOperations(t *testing.T) {
	old := socketName
	t.Cleanup(func() { SetSocket(old) })
	SetSocket("work")
	withMock(t, func(m *mockRunner) {
		m.OnOutput(nil, nil, "tmux", "-L", "work", "list-sessions", "-F", listFormat)
		m.OnOutput(nil, nil, "tmux", "-L", "work", "list-windows", "-t", "dev", "-F", windowListFormat)
		m.OnOutput(nil, nil, "tmux", "-L", "work", "list-panes", "-t", "dev:1", "-F", paneListFormat)
		m.OnOutput([]byte("preview\n"), nil, "tmux", "-L", "work", "capture-pane", "-t", "dev:1.2", "-p", "-e")
		if _, err := ListSessions(); err != nil {
			t.Fatal(err)
		}
		if _, err := ListWindows("dev"); err != nil {
			t.Fatal(err)
		}
		if _, err := ListPanes("dev", 1); err != nil {
			t.Fatal(err)
		}
		if got, err := CapturePaneTarget("dev:1.2"); err != nil || got != "preview" {
			t.Fatalf("capture = %q, %v", got, err)
		}
		for _, err := range []error{
			CreateSession("dev"),
			CreateSessionWithDir("docs", "/tmp/docs"),
			RenameSession("dev", "new"),
			KillSession("new"),
		} {
			if err != nil {
				t.Fatal(err)
			}
		}
		want := []string{
			"tmux -L work new-session -d -s dev",
			"tmux -L work new-session -d -s docs -c /tmp/docs",
			"tmux -L work rename-session -t dev new",
			"tmux -L work kill-session -t new",
		}
		if !slices.Equal(m.runs, want) {
			t.Fatalf("commands = %q, want %q", m.runs, want)
		}
	})
}

func TestIsCurrentServer(t *testing.T) {
	old := socketName
	t.Cleanup(func() { SetSocket(old) })
	for _, tt := range []struct {
		name, socket, current, selected string
		want                            bool
	}{
		{name: "outside tmux", socket: "work"},
		{name: "inherited", current: "/tmp/current,123,0", want: true},
		{name: "same server", socket: "work", current: "/tmp/work,123,0", selected: "/tmp/work", want: true},
		{name: "trailing space", socket: "work ", current: "/tmp/work ,123,0", selected: "/tmp/work ", want: true},
		{name: "other server", socket: "work", current: "/tmp/other,123,0", selected: "/tmp/work"},
		{name: "comma in path", socket: "work", current: "/tmp/a,b,123,0", selected: "/tmp/a,b", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX", tt.current)
			SetSocket(tt.socket)
			withMock(t, func(m *mockRunner) {
				m.OnOutput([]byte(tt.selected+"\n"), nil, "tmux", "-L", tt.socket, "display-message", "-p", "#{socket_path}")
				got, err := IsCurrentServer()
				if err != nil || got != tt.want {
					t.Fatalf("IsCurrentServer() = %v, %v; want %v", got, err, tt.want)
				}
			})
		})
	}
}

func TestIsCurrentServerThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "socket")
	alias := filepath.Join(dir, "alias")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	old := socketName
	t.Cleanup(func() { SetSocket(old) })
	SetSocket("work")
	t.Setenv("TMUX", alias+",123,0")
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte(path+"\n"), nil, "tmux", "-L", "work", "display-message", "-p", "#{socket_path}")
		if got, err := IsCurrentServer(); err != nil || !got {
			t.Fatalf("IsCurrentServer() = %v, %v; want true", got, err)
		}
	})
}

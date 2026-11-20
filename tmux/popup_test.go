package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const sampleOhMyTmuxLocal = `# -- general -------------------------------------------------------------------

tmux_conf_24b_colour=true

# -- custom variables ----------------------------------------------------------

# EOF

# "$@"
`

func TestMuxShellCommandPreservesArguments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mux with 'quotes' #$name")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\000' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	old := socketName
	t.Cleanup(func() { SetSocket(old) })
	for _, socket := range []string{"", "work", "work with 'quotes' #${name}; $(false)"} {
		t.Run(socket, func(t *testing.T) {
			SetSocket(socket)
			out, err := exec.Command("sh", "-c", muxShellCommand(path, "vertical")).Output()
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"--layout", "vertical"}
			if socket != "" {
				want = append([]string{"--socket", socket}, want...)
			}
			got := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
			if !slices.Equal(got, want) {
				t.Fatalf("shell arguments = %q, want %q", got, want)
			}
		})
	}
}

func TestFindTmuxConfLocal(t *testing.T) {
	tests := []struct {
		conf string
		want string
	}{
		{conf: "/home/u/.tmux.conf", want: "/home/u/.tmux.conf.local"},
		{conf: "/home/u/.config/tmux/tmux.conf", want: "/home/u/.config/tmux/tmux.conf.local"},
		{conf: "/etc/odd/path", want: "/etc/odd/path.local"},
	}
	for _, tt := range tests {
		if got := findTmuxConfLocal(tt.conf); got != tt.want {
			t.Errorf("findTmuxConfLocal(%q) = %q, want %q", tt.conf, got, tt.want)
		}
	}
}

func TestIsOhMyTmux_Signature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte(ohMyTmuxSignature+"\n# rest of file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isOhMyTmux(path) {
		t.Error("expected oh-my-tmux to be detected via signature")
	}
}

func TestIsOhMyTmux_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".tmux", ".tmux.conf")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# anything\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".tmux.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if !isOhMyTmux(link) {
		t.Error("expected oh-my-tmux to be detected via symlink target")
	}
}

func TestIsOhMyTmux_PlainConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if isOhMyTmux(path) {
		t.Error("plain config should not be detected as oh-my-tmux")
	}
}

func TestIsOhMyTmux_Missing(t *testing.T) {
	if isOhMyTmux(filepath.Join(t.TempDir(), "does-not-exist")) {
		t.Error("missing file should not be detected as oh-my-tmux")
	}
}

func TestUpsertBindLine_AppendsToPlainConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/mux"`, true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "set -g mouse on") {
		t.Error("existing config dropped")
	}
	if !strings.Contains(string(got), muxKeybindMarker) {
		t.Error("marker not written")
	}
}

func TestUpsertBindLine_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/old"`, true); err != nil {
		t.Fatal(err)
	}
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/new"`, true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "/bin/old") {
		t.Error("old bind should have been replaced")
	}
	if !strings.Contains(string(got), "/bin/new") {
		t.Error("new bind missing")
	}
	if n := strings.Count(string(got), muxKeybindMarker); n != 1 {
		t.Errorf("expected exactly one marker line, got %d", n)
	}
}

func TestWriteBindToLocal_InsertsBeforeSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte(sampleOhMyTmuxLocal), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/mux"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)

	bindIdx := strings.Index(s, "/bin/mux")
	sentIdx := strings.Index(s, ohMyTmuxSentinel)
	if bindIdx == -1 {
		t.Fatal("bind line missing")
	}
	if sentIdx == -1 {
		t.Fatal("sentinel was removed")
	}
	if bindIdx >= sentIdx {
		t.Errorf("bind line should appear before sentinel (bind=%d, sentinel=%d)", bindIdx, sentIdx)
	}
	// The cut -c3- shell extraction would treat any non-`# `-prefixed line as
	// shell input. Sanity check: the tagged line still starts with `bind`,
	// not a stripped `nd m display-popup …`.
	if !strings.Contains(s, "bind m display-popup") {
		t.Error("bind line should not have been mangled")
	}
}

func TestWriteBindToLocal_NoSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/mux"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "/bin/mux") {
		t.Error("bind line missing when sentinel absent")
	}
}

func TestWriteBindToLocal_CreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/mux"`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "/bin/mux") {
		t.Error("bind line missing in newly created file")
	}
}

func TestWriteBindToLocal_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte(sampleOhMyTmuxLocal), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/old"`); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/new"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "/bin/old") {
		t.Error("old bind should have been replaced on second write")
	}
	if n := strings.Count(s, muxKeybindMarker); n != 1 {
		t.Errorf("expected exactly one marker line after re-run, got %d", n)
	}
	// Sentinel must still be the last meaningful line.
	if !strings.Contains(s, ohMyTmuxSentinel) {
		t.Error("sentinel lost on rewrite")
	}
}

func TestStripMarkerLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	original := "set -g mouse on\nbind m display-popup -E \"/bin/old\"  " + muxKeybindMarker + "\nset -g status on\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	removed, err := stripMarkerLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Error("expected removed=true")
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, muxKeybindMarker) {
		t.Error("marker line should be gone")
	}
	if !strings.Contains(s, "set -g mouse on") || !strings.Contains(s, "set -g status on") {
		t.Error("non-marker lines should be preserved")
	}
}

func TestStripMarkerLines_Missing(t *testing.T) {
	removed, err := stripMarkerLines(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if removed {
		t.Error("expected removed=false for missing file")
	}
}

func TestStripMarkerLines_NoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	removed, err := stripMarkerLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Error("expected removed=false when no marker present")
	}
}

// Pre-fix install.sh appended an untagged bind line to ~/.tmux.conf. For
// users corrupted via the installer (not via `mux setup-keybind`), our
// cleanup must strip the legacy line too — otherwise the oh-my-tmux
// `cut -c3- | sh` reload still fails after the fix.
func TestStripMarkerLines_RemovesLegacyInstallerBind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	original := "# : << 'EOF'\n# (oh-my-tmux body)\nset -g mouse on\n" +
		`bind-key m display-popup -E -w 80% -h 80% "mux"` + "\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	removed, err := stripMarkerLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected legacy installer bind to be removed")
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), legacyInstallerBindFragment) {
		t.Error("legacy installer bind should be gone")
	}
	if !strings.Contains(string(got), "# : << 'EOF'") {
		t.Error("oh-my-tmux signature should be preserved")
	}
}

func TestStripMarkerLines_PreservesUserCustomBind(t *testing.T) {
	// A user-authored binding that mentions mux but doesn't match the exact
	// legacy installer pattern should be preserved. We only nuke our own
	// known shapes.
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	original := `bind X display-popup -E "/usr/local/bin/mux popup"` + "\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	removed, err := stripMarkerLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Error("user-authored bind should not be touched")
	}
}

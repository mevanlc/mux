package tmux

import (
	"testing"
	"time"
)

func TestIsAICommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"claude", true},
		{"codex", true},
		{"aider", true},
		{"gemini", true},
		{"bash", false},
		{"vim", false},
		{"", false},
		{"Claude", false}, // case-sensitive
	}

	for _, tt := range tests {
		if got := IsAICommand(tt.cmd); got != tt.want {
			t.Errorf("IsAICommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestResolveCommandCache(t *testing.T) {
	// Clear cache before test
	cmdCacheMu.Lock()
	cmdCache = make(map[int]cachedCommand)
	cmdCacheMu.Unlock()

	// Pre-populate cache with a known result
	cmdCacheMu.Lock()
	cmdCache[99999] = cachedCommand{
		command:   "claude",
		expiresAt: time.Now().Add(10 * time.Second),
	}
	cmdCacheMu.Unlock()

	// Should return cached value without calling pgrep/ps
	result := resolveCommand(99999, "bash")
	if result != "claude" {
		t.Errorf("expected cached 'claude', got %q", result)
	}

	// Expire the cache entry
	cmdCacheMu.Lock()
	cmdCache[99999] = cachedCommand{
		command:   "claude",
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	cmdCacheMu.Unlock()

	// Expired cache should fall through (pgrep will fail for fake PID, returning rawCmd)
	result = resolveCommand(99999, "bash")
	if result != "bash" {
		t.Errorf("expected fallback 'bash' after cache expiry, got %q", result)
	}
}

func TestResolveCommandDirectAI(t *testing.T) {
	// If rawCmd is already an AI command, should return immediately (no cache needed)
	result := resolveCommand(12345, "claude")
	if result != "claude" {
		t.Errorf("expected 'claude', got %q", result)
	}
}

func TestResolveCommandDetectsAIFromPanePID(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte("claude\n"), nil, "ps", "-o", "comm=", "-p", "80346")

		result := resolveCommand(80346, "2.1.215")
		if result != "claude" {
			t.Errorf("expected 'claude', got %q", result)
		}
	})
}

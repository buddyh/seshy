package retention

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReportFixture(t *testing.T) {
	home := newFixtureHome(t)
	rows := Report(home)

	if len(rows) != len(Sources) {
		t.Fatalf("rows = %d, want %d", len(rows), len(Sources))
	}

	// (sessions, oldest) per agent, from the pinned fixture instants.
	want := []struct {
		key      string
		sessions int
		oldest   string // RFC3339, "" = zero
	}{
		{"claude", 2, "2025-08-01T12:00:00Z"}, // agent-* transcript excluded
		{"codex", 1, "2026-01-15T09:30:00Z"},
		{"gemini", 1, "2026-01-15T09:30:00Z"},
		{"grok", 1, "2025-08-01T12:00:00Z"},
		{"pi", 1, "2026-01-15T09:30:00Z"},
		{"opencode", 2, "2025-08-01T12:00:00Z"}, // from time_created, not file mtime
		{"agy", 1, "2026-06-20T18:45:00Z"},
		{"droid", 1, "2026-01-15T09:30:00Z"},
		{"cursor", 2, "2025-08-01T12:00:00Z"},  // chat dir + project transcript
		{"copilot", 2, "2025-08-01T12:00:00Z"}, // current + legacy store
	}
	for _, w := range want {
		r := rowFor(t, rows, w.key)
		if !r.Installed {
			t.Errorf("%s: not installed, want installed", w.key)
			continue
		}
		if r.Sessions != w.sessions {
			t.Errorf("%s: sessions = %d, want %d", w.key, r.Sessions, w.sessions)
		}
		got := ""
		if !r.Oldest.IsZero() {
			got = r.Oldest.UTC().Format("2006-01-02T15:04:05Z")
		}
		if got != w.oldest {
			t.Errorf("%s: oldest = %s, want %s", w.key, got, w.oldest)
		}
	}
}

func TestReportBytes(t *testing.T) {
	home := newFixtureHome(t)
	rows := Report(home)

	// claude: 100 + 200 + 50 (agent-* counts toward disk even though it is
	// not a session).
	if r := rowFor(t, rows, "claude"); r.Bytes != 350 {
		t.Errorf("claude bytes = %d, want 350", r.Bytes)
	}
	// copilot: both stores.
	if r := rowFor(t, rows, "copilot"); r.Bytes != 100 {
		t.Errorf("copilot bytes = %d, want 100", r.Bytes)
	}
	// opencode: whatever sqlite wrote — assert it matches the file itself
	// rather than a hardcoded size (page allocation is a driver detail).
	fi, err := os.Stat(filepath.Join(home, ".local/share/opencode/opencode.db"))
	if err != nil {
		t.Fatal(err)
	}
	if r := rowFor(t, rows, "opencode"); r.Bytes != fi.Size() {
		t.Errorf("opencode bytes = %d, want %d", r.Bytes, fi.Size())
	}
}

func TestReportNotInstalled(t *testing.T) {
	neutralEnv(t)
	rows := Report(t.TempDir()) // empty home: nothing installed
	for _, r := range rows {
		if r.Installed {
			t.Errorf("%s: installed in empty home", r.Key)
		}
		if r.Bytes != 0 || r.Sessions != 0 || !r.Oldest.IsZero() {
			t.Errorf("%s: non-zero usage in empty home: %+v", r.Key, r)
		}
		if r.Policy.Kind == "" {
			t.Errorf("%s: missing policy", r.Key)
		}
	}
}

func TestEnvOverrideRoots(t *testing.T) {
	home := t.TempDir()
	alt := t.TempDir()
	t.Setenv("GROK_HOME", filepath.Join(alt, "grok-home"))
	writeAt(t, filepath.Join(alt, "grok-home"), "sessions/%2Ftmp%2Fproject/uuid-9/summary.json", 30, tMid)

	r := rowFor(t, Report(home), "grok")
	if !r.Installed {
		t.Fatal("grok: GROK_HOME store not picked up")
	}
	if r.Sessions != 1 || r.Bytes != 30 {
		t.Errorf("grok via GROK_HOME: sessions=%d bytes=%d, want 1/30", r.Sessions, r.Bytes)
	}
}

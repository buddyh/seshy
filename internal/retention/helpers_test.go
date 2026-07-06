package retention

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// Fixture instants, all pinned so counts, ages, and golden output are
// deterministic. Files are aged with os.Chtimes after writing.
var (
	tOld = time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	tMid = time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC)
	tNew = time.Date(2026, 6, 20, 18, 45, 0, 0, time.UTC)
)

// writeAt writes a file of n bytes under home, creating parents, and pins
// its mtime.
func writeAt(t *testing.T, home, rel string, n int, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
	chtimes(t, path, mtime)
	return path
}

// writeFile writes literal content under home and pins its mtime.
func writeFile(t *testing.T, home, rel, content string, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	chtimes(t, path, mtime)
	return path
}

func chtimes(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// newOpencodeDB creates a real opencode.db with the session table and the
// given (id, time_created unix-millis) rows.
func newOpencodeDB(t *testing.T, path string, created ...time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE session (
		id TEXT, directory TEXT, title TEXT,
		time_created INTEGER, time_updated INTEGER, model TEXT, cost REAL)`); err != nil {
		t.Fatal(err)
	}
	for i, c := range created {
		if _, err := db.Exec(
			"INSERT INTO session VALUES (?,?,?,?,?,?,?)",
			// Directory values stay generic: this repo is public.
			"ses_"+string(rune('a'+i)), "/tmp/project", "fixture session",
			c.UnixMilli(), c.UnixMilli(), "model-x", 0.0,
		); err != nil {
			t.Fatal(err)
		}
	}
}

// neutralEnv clears the store-relocating env vars so fixture homes are
// hermetic regardless of the host machine's environment. envOr treats an
// empty value as unset.
func neutralEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GROK_HOME", "COPILOT_HOME", "XDG_DATA_HOME"} {
		t.Setenv(key, "")
	}
}

// newFixtureHome builds a synthetic home covering all ten agents:
// two Claude transcripts (plus an excluded agent-* subagent file), one
// session per remaining store, and no settings files (default policies).
func newFixtureHome(t *testing.T) string {
	t.Helper()
	neutralEnv(t)
	home := t.TempDir()

	// claude: 2 counted transcripts + 1 excluded subagent transcript
	writeAt(t, home, ".claude/projects/-tmp-project/aaaa.jsonl", 100, tOld)
	writeAt(t, home, ".claude/projects/-tmp-project/bbbb.jsonl", 200, tNew)
	writeAt(t, home, ".claude/projects/-tmp-project/agent-cccc.jsonl", 50, tMid)

	// codex: date-sharded rollout
	writeAt(t, home, ".codex/sessions/2026/01/15/rollout-2026-01-15-uuid1.jsonl", 300, tMid)

	// gemini: one chat in one project-hash dir
	writeAt(t, home, ".gemini/tmp/hash1/chats/session-1.json", 150, tMid)

	// grok: one session dir with summary.json
	writeAt(t, home, ".grok/sessions/%2Ftmp%2Fproject/uuid-1/summary.json", 80, tOld)

	// pi: one session
	writeAt(t, home, ".pi/agent/sessions/--tmp-project--/2026-01-15T09-30-00-000Z_uuid.jsonl", 120, tMid)

	// opencode: real sqlite db, two sessions
	newOpencodeDB(t, filepath.Join(home, ".local/share/opencode/opencode.db"), tOld, tNew)

	// agy: one conversation db
	writeAt(t, home, ".gemini/antigravity-cli/conversations/conv-1.db", 90, tNew)

	// droid: one session
	writeAt(t, home, ".factory/sessions/-tmp-project/uuid-1.jsonl", 110, tMid)

	// cursor: one chat dir
	writeAt(t, home, ".cursor/chats/hashA/uuid-1/store.db", 70, tOld)

	// copilot: one current + one legacy session dir
	writeAt(t, home, ".copilot/session-state/uuid-1/events.jsonl", 60, tNew)
	writeAt(t, home, ".copilot/history-session-state/uuid-0/events.jsonl", 40, tOld)

	return home
}

// rowFor pulls one agent's row out of a Report result.
func rowFor(t *testing.T, rows []Row, key string) Row {
	t.Helper()
	for _, r := range rows {
		if r.Key == key {
			return r
		}
	}
	t.Fatalf("no row for %q", key)
	return Row{}
}

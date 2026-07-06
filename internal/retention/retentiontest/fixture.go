// Package retentiontest builds synthetic agent-store homes for tests of the
// retention feature (shared by the retention package and render's golden
// test). It writes fixture files only — it does not import retention.
package retentiontest

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite" // driver for the fixture opencode db
)

// Fixture instants, pinned so counts, ages, and golden output are
// deterministic. Files are aged with os.Chtimes after writing.
var (
	TOld = time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	TMid = time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC)
	TNew = time.Date(2026, 6, 20, 18, 45, 0, 0, time.UTC)
)

// WriteAt writes a file of n zero bytes under home, creating parents, and
// pins its mtime.
func WriteAt(t *testing.T, home, rel string, n int, mtime time.Time) string {
	t.Helper()
	return write(t, home, rel, make([]byte, n), mtime)
}

// WriteFile writes literal content under home and pins its mtime.
func WriteFile(t *testing.T, home, rel, content string, mtime time.Time) string {
	t.Helper()
	return write(t, home, rel, []byte(content), mtime)
}

func write(t *testing.T, home, rel string, b []byte, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	return path
}

// NeutralEnv clears the store-relocating env vars so fixture homes are
// hermetic regardless of the host machine's environment (retention's envOr
// treats an empty value as unset).
func NeutralEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GROK_HOME", "COPILOT_HOME", "XDG_DATA_HOME"} {
		t.Setenv(key, "")
	}
}

// NewOpencodeDB creates a real opencode.db with the session table and one
// row per created instant (time_created/time_updated in unix millis).
func NewOpencodeDB(t *testing.T, path string, created ...time.Time) {
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
			// Values stay generic: this repo is public.
			"ses_"+string(rune('a'+i)), "/tmp/project", "fixture session",
			c.UnixMilli(), c.UnixMilli(), "model-x", 0.0,
		); err != nil {
			t.Fatal(err)
		}
	}
}

// FixtureHome builds a synthetic home covering all ten agents: two Claude
// transcripts (plus an excluded agent-* subagent file), one session per
// remaining store (two for opencode and copilot), and no settings files
// (default policies everywhere).
func FixtureHome(t *testing.T) string {
	t.Helper()
	NeutralEnv(t)
	home := t.TempDir()

	// claude: 2 counted transcripts + 1 excluded subagent transcript
	WriteAt(t, home, ".claude/projects/-tmp-project/aaaa.jsonl", 100, TOld)
	WriteAt(t, home, ".claude/projects/-tmp-project/bbbb.jsonl", 200, TNew)
	WriteAt(t, home, ".claude/projects/-tmp-project/agent-cccc.jsonl", 50, TMid)

	// codex: date-sharded rollout
	WriteAt(t, home, ".codex/sessions/2026/01/15/rollout-2026-01-15-uuid1.jsonl", 300, TMid)

	// gemini: one chat in one project-hash dir
	WriteAt(t, home, ".gemini/tmp/hash1/chats/session-1.json", 150, TMid)

	// grok: one session dir with summary.json
	WriteAt(t, home, ".grok/sessions/%2Ftmp%2Fproject/uuid-1/summary.json", 80, TOld)

	// pi: one session
	WriteAt(t, home, ".pi/agent/sessions/--tmp-project--/2026-01-15T09-30-00-000Z_uuid.jsonl", 120, TMid)

	// opencode: real sqlite db, two sessions
	NewOpencodeDB(t, filepath.Join(home, ".local/share/opencode/opencode.db"), TOld, TNew)

	// agy: one conversation db
	WriteAt(t, home, ".gemini/antigravity-cli/conversations/conv-1.db", 90, TNew)

	// droid: one session
	WriteAt(t, home, ".factory/sessions/-tmp-project/uuid-1.jsonl", 110, TMid)

	// cursor: one chat dir + one project agent transcript
	WriteAt(t, home, ".cursor/chats/hashA/uuid-1/store.db", 70, TOld)
	WriteAt(t, home, ".cursor/projects/proj-a/agent-transcripts/uuid-2.jsonl", 35, TMid)

	// copilot: one current + one legacy session dir
	WriteAt(t, home, ".copilot/session-state/uuid-1/events.jsonl", 60, TNew)
	WriteAt(t, home, ".copilot/history-session-state/uuid-0/events.jsonl", 40, TOld)

	return home
}

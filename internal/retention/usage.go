package retention

import (
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buddyh/seshy/internal/store"
)

// dirBytes sums the sizes of every regular file under root (root itself when
// it is a file). Unreadable entries are skipped, not fatal.
func dirBytes(root string) int64 {
	var total int64
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			total += fi.Size()
		}
		return nil
	})
	return total
}

// countFiles counts files matching glob and returns the minimum mtime.
// keep filters candidates by base name (nil keeps everything).
func countFiles(pattern string, keep func(base string) bool) (int, time.Time) {
	var n int
	var oldest time.Time
	for _, p := range store.Glob(pattern) {
		if keep != nil && !keep(filepath.Base(p)) {
			continue
		}
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		n++
		if oldest.IsZero() || fi.ModTime().Before(oldest) {
			oldest = fi.ModTime()
		}
	}
	return n, oldest
}

// countClaude counts transcripts the way the collector does: *.jsonl per
// project folder, excluding agent-* subagent files.
func countClaude(home string) (int, time.Time) {
	return countFiles(filepath.Join(home, ".claude", "projects", "*", "*.jsonl"),
		func(base string) bool { return !strings.HasPrefix(base, "agent-") })
}

// countCodex walks the date-sharded rollout tree.
func countCodex(home string) (int, time.Time) {
	var n int
	var oldest time.Time
	root := filepath.Join(home, ".codex", "sessions")
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), "rollout-") || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			n++
			if oldest.IsZero() || fi.ModTime().Before(oldest) {
				oldest = fi.ModTime()
			}
		}
		return nil
	})
	return n, oldest
}

func countGemini(home string) (int, time.Time) {
	return countFiles(filepath.Join(home, ".gemini", "tmp", "*", "chats", "*"), nil)
}

// countGrok counts session dirs under each url-escaped project dir, aging
// them by their summary.json (what the collector stats).
func countGrok(home string) (int, time.Time) {
	return countFiles(filepath.Join(grokHome(home), "sessions", "%*", "*", "summary.json"), nil)
}

func countPi(home string) (int, time.Time) {
	return countFiles(filepath.Join(home, ".pi", "agent", "sessions", "*", "*.jsonl"), nil)
}

// countOpenCode reads the session table of every channel-suffixed database
// (opencode.db, opencode-<channel>.db); time_created is unix millis. A DB
// that fails to open falls back to its file mtime.
func countOpenCode(home string) (int, time.Time) {
	var n int
	var oldest time.Time
	for _, dbPath := range store.Glob(filepath.Join(opencodeData(home), "opencode*.db")) {
		count, min, err := opencodeSessions(dbPath)
		if err != nil {
			if fi, statErr := os.Stat(dbPath); statErr == nil {
				if oldest.IsZero() || fi.ModTime().Before(oldest) {
					oldest = fi.ModTime()
				}
			}
			continue
		}
		n += count
		if count > 0 && (oldest.IsZero() || min.Before(oldest)) {
			oldest = min
		}
	}
	return n, oldest
}

func opencodeSessions(dbPath string) (int, time.Time, error) {
	db, err := store.OpenRO(dbPath)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer db.Close()
	var count int
	var min sql.NullInt64
	row := db.QueryRow("SELECT count(*), min(time_created) FROM session")
	if err := row.Scan(&count, &min); err != nil {
		return 0, time.Time{}, err
	}
	if !min.Valid {
		return count, time.Time{}, nil
	}
	return count, time.UnixMilli(min.Int64), nil
}

func countAgy(home string) (int, time.Time) {
	return countFiles(filepath.Join(home, ".gemini", "antigravity-cli", "conversations", "*.db"), nil)
}

// countCursor counts chat dirs (~/.cursor/chats/<workspace-hash>/<uuid>),
// aging them by the files inside.
func countCursor(home string) (int, time.Time) {
	var n int
	var oldest time.Time
	for _, dir := range store.Glob(filepath.Join(home, ".cursor", "chats", "*", "*")) {
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			continue
		}
		n++
		if _, min := countFiles(filepath.Join(dir, "*"), nil); !min.IsZero() {
			if oldest.IsZero() || min.Before(oldest) {
				oldest = min
			}
		}
	}
	return n, oldest
}

func countDroid(home string) (int, time.Time) {
	return countFiles(filepath.Join(home, ".factory", "sessions", "*", "*.jsonl"), nil)
}

// countCopilot counts session dirs in both the current and legacy stores,
// aging them by the files inside.
func countCopilot(home string) (int, time.Time) {
	var n int
	var oldest time.Time
	ch := copilotHome(home)
	for _, root := range []string{"session-state", "history-session-state"} {
		for _, dir := range store.Glob(filepath.Join(ch, root, "*")) {
			fi, err := os.Stat(dir)
			if err != nil || !fi.IsDir() {
				continue
			}
			n++
			if _, min := countFiles(filepath.Join(dir, "*"), nil); !min.IsZero() {
				if oldest.IsZero() || min.Before(oldest) {
					oldest = min
				}
			}
		}
	}
	return n, oldest
}

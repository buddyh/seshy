package plugin

import (
	"encoding/json"
	"github.com/buddyh/seshy/internal/agents"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, contents ...string) (*Service, []agents.Session) {
	t.Helper()
	home := t.TempDir()
	var rows []agents.Session
	for i, text := range contents {
		path := filepath.Join(home, ".codex", "sessions", string(rune('a'+i))+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, agents.Session{Tool: "codex", ID: string(rune('a' + i)), Path: path, Dir: "/project", Mtime: time.Unix(1000+int64(i), 0)})
	}
	return &Service{home: home, collect: func() []agents.Session { return rows }}, rows
}
func line(text string) string {
	b, _ := json.Marshal(map[string]any{"role": "user", "content": text})
	return string(b) + "\n"
}

func TestServiceScopedPaginationAndStaleSearchCursor(t *testing.T) {
	s, _ := fixture(t, line("first"), line("second"), line("third"))
	q := Query{Directory: "/project", Limit: 1}
	p, err := s.List(q, false)
	if err != nil || len(p.Sessions) != 1 || p.NextCursor == "" {
		t.Fatalf("page: %+v %v", p, err)
	}
	q.Cursor = p.NextCursor
	next, err := s.List(q, false)
	if err != nil || next.Sessions[0].Key == p.Sessions[0].Key {
		t.Fatalf("continuation: %+v %v", next, err)
	}
	q.Directory = "/another"
	if _, err = s.List(q, false); err == nil {
		t.Fatal("accepted cursor with changed scope")
	}
	q.Directory = "/project"
	s.mu.Lock()
	s.loaded = time.Time{}
	s.collect = func() []agents.Session { return nil }
	s.mu.Unlock()
	if _, err = s.List(q, false); err == nil {
		t.Fatal("accepted stale snapshot")
	}
}

func TestServiceSearchTraversesLargeSessionAndLateText(t *testing.T) {
	// More than one request's 16 MiB budget; the only match is near the end.
	text := strings.Repeat(line(strings.Repeat("ordinary ", 10000)), 220) + line("unique-late-decision")
	s, _ := fixture(t, text)
	q := Query{Query: "unique-late-decision", Directory: "/project"}
	p, err := s.List(q, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sessions) > 0 || p.NextCursor == "" || p.CoverageComplete {
		t.Fatalf("first scan must continue: %+v", p)
	}
	for i := 0; i < 10 && len(p.Sessions) == 0 && p.NextCursor != ""; i++ {
		q.Cursor = p.NextCursor
		p, err = s.List(q, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err != nil || len(p.Sessions) != 1 || !strings.Contains(p.Snippets["codex:a"], q.Query) {
		t.Fatalf("missed late match: %+v %v", p, err)
	}
}

func TestServiceRejectsMissingKeysSymlinkEscapeAndAmbiguity(t *testing.T) {
	s, rows := fixture(t, line("selected"))
	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	os.WriteFile(outside, []byte(line("secret")), 0600)
	link := filepath.Join(s.home, ".codex", "sessions", "link.jsonl")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	escaped := rows[0]
	escaped.ID = "escape"
	escaped.Path = link
	s.collect = func() []agents.Session { return append(rows, escaped) }
	p, err := s.List(Query{}, false)
	if err != nil || len(p.Sessions) != 1 || p.CoverageComplete || len(p.Warnings) == 0 {
		t.Fatalf("escape was not excluded visibly: %+v %v", p, err)
	}
	if _, err = s.Read(ReadRequest{SessionKey: "codex:escape"}); err == nil {
		t.Fatal("read escaped source")
	}
	if _, err = s.Read(ReadRequest{SessionKey: outside}); err == nil {
		t.Fatal("accepted arbitrary caller path")
	}
	s.loaded = time.Time{}
	s.collect = func() []agents.Session { return append(rows, rows[0]) }
	if _, err = s.Read(ReadRequest{SessionKey: "codex:a"}); err == nil {
		t.Fatal("accepted ambiguous native identity")
	}
}

func TestSourceChangeInvalidatesReadAndHandoffNeverExecutes(t *testing.T) {
	s, rows := fixture(t, line("original goal")+line("verified decision"))
	p, err := s.Read(ReadRequest{SessionKey: "codex:a", Limit: 1})
	if err != nil || p.NextCursor == "" {
		t.Fatal(err)
	}
	if err = os.WriteFile(rows[0].Path, []byte(line("source changed substantially")), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Read(ReadRequest{SessionKey: "codex:a", Cursor: p.NextCursor}); err == nil {
		t.Fatal("accepted old source cursor")
	}
	h, err := s.Handoff("codex:a")
	if err != nil {
		t.Fatal(err)
	}
	if h["executed"] != false || !strings.Contains(h["prompt"].(string), "untrusted historical data") {
		t.Fatalf("unsafe handoff: %+v", h)
	}
}

func TestDirectoryRootAndInvalidQueries(t *testing.T) {
	s, _ := fixture(t, line("test"))
	p, err := s.List(Query{Directory: "/", IncludeSubdirectories: true}, false)
	if err != nil || len(p.Sessions) != 1 {
		t.Fatalf("root scope: %+v %v", p, err)
	}
	for _, q := range []Query{{Directory: "relative"}, {Limit: 51}, {Agent: "unknown"}, {After: "yesterday"}, {After: "2026-01-02T00:00:00Z", Before: "2026-01-01T00:00:00Z"}} {
		if _, err = s.List(q, false); err == nil {
			t.Fatalf("accepted invalid query: %+v", q)
		}
	}
}

func TestGrokCursorTracksTranscriptRatherThanSummary(t *testing.T) {
	s, rows := fixture(t, line("metadata"))
	root := filepath.Join(s.home, ".grok", "sessions", "project", "session")
	os.MkdirAll(root, 0700)
	summary := filepath.Join(root, "summary.json")
	history := filepath.Join(root, "chat_history.jsonl")
	os.WriteFile(summary, []byte("{}"), 0600)
	os.WriteFile(history, []byte(line("one")+line("two")), 0600)
	rows[0].Tool = "grok"
	rows[0].Path = summary
	p, err := s.Read(ReadRequest{SessionKey: "grok:a", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if p.Session.Source != history {
		t.Fatal("byte citation points at metadata instead of actual history")
	}
	os.WriteFile(history, []byte(line("changed substantially")), 0600)
	if _, err = s.Read(ReadRequest{SessionKey: "grok:a", Cursor: p.NextCursor}); err == nil {
		t.Fatal("accepted stale Grok source")
	}
}

func TestAntigravityTaskClippingIsExplicit(t *testing.T) {
	s, rows := fixture(t, line("metadata"))
	root := filepath.Join(s.home, ".gemini", "antigravity-cli", "brain", "a")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "task.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 2000)), 0600); err != nil {
		t.Fatal(err)
	}
	rows[0].Tool = "agy"
	rows[0].Path = path
	p, err := s.Read(ReadRequest{SessionKey: "agy:a", MaxChars: 1000})
	if err != nil || len(p.Page.Messages) != 1 || !p.Page.Messages[0].Truncated || p.Page.Complete || p.Page.SourceBytes != 2000 || p.Session.Source != path {
		t.Fatalf("task clipping contract: %+v, %v", p, err)
	}
}

package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func transcriptFile(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "history.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func bounds() TranscriptOptions {
	return TranscriptOptions{Limit: 100, MaxChars: 64000, MaxBytes: 8 << 20}
}
func textRecord(role, text string) string {
	b, _ := json.Marshal(map[string]any{"role": role, "content": text})
	return string(b)
}

func TestLiteralExcerptUsesUnicodePositions(t *testing.T) {
	text := strings.Repeat("İ", 200) + "MATCH" + strings.Repeat("世界", 300)
	snippet, found := LiteralExcerpt(text, "match")
	if !found || !strings.Contains(snippet, "MATCH") || strings.Contains(snippet, "�") {
		t.Fatalf("invalid Unicode excerpt: %q", snippet)
	}
	path := transcriptFile(t, textRecord("user", text))
	options := bounds()
	options.LiteralQuery = "match"
	page, err := ReadJSONLTranscript(path, options)
	if err != nil || len(page.Messages) != 1 || !strings.Contains(page.Messages[0].Text, "MATCH") {
		t.Fatalf("Unicode transcript search failed: %+v, %v", page, err)
	}
}

func TestTranscriptShapesAndSourceLocations(t *testing.T) {
	lines := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Claude question"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Codex answer"}]}}`,
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"pi question"}]}}`,
		`{"type":"message","role":"assistant","content":"Droid answer"}`,
		`{"type":"user","content":"Grok question"}`,
	}
	p, err := ReadJSONLTranscript(transcriptFile(t, lines...), bounds())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Messages) != 5 || !p.Complete {
		t.Fatalf("want all messages and complete: %+v", p)
	}
	offset := int64(0)
	for i, m := range p.Messages {
		if m.ByteOffset == nil || *m.ByteOffset != offset {
			t.Fatalf("source %d: %+v", i, m)
		}
		offset += int64(len(lines[i]) + 1)
	}
}

func TestTranscriptOmitsDuplicateEventsReasoningAndInstructions(t *testing.T) {
	p, err := ReadJSONLTranscript(transcriptFile(t,
		`{"type":"event_msg","payload":{"type":"user_message","message":"duplicate"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","channel":"analysis","content":"private reasoning"}}`,
		`{"role":"assistant","channel":"analysis","content":"private reasoning"}`,
		textRecord("user", "# AGENTS.md instructions\nsetup"), textRecord("system", "setup"),
		`{"role":"assistant","content":[{"type":"thinking","thinking":"private reasoning"},{"type":"text","text":"Public answer"},{"type":"tool_use","name":"bash"}]}`,
	), bounds())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Messages) != 1 || p.Messages[0].Text != "Public answer" {
		t.Fatalf("unexpected records: %+v", p.Messages)
	}
}

func TestTranscriptPaginationAndInvalidBoundaries(t *testing.T) {
	path := transcriptFile(t, textRecord("user", "one"), textRecord("assistant", "two"), textRecord("user", "three"))
	o := bounds()
	o.Limit = 1
	p, err := ReadJSONLTranscript(path, o)
	if err != nil {
		t.Fatal(err)
	}
	if p.NextOffset == nil || p.Complete {
		t.Fatal("first page must continue")
	}
	o.Offset = *p.NextOffset
	p, err = ReadJSONLTranscript(path, o)
	if err != nil || p.Messages[0].Text != "two" {
		t.Fatalf("second page: %+v %v", p, err)
	}
	o.Offset = 1
	if _, err = ReadJSONLTranscript(path, o); err == nil {
		t.Fatal("accepted arbitrary byte offset")
	}
	o.Offset = 1 << 40
	if _, err = ReadJSONLTranscript(path, o); err == nil {
		t.Fatal("accepted out of range offset")
	}
}

func TestTranscriptMalformedCompactedAndUnicodeBudgets(t *testing.T) {
	path := transcriptFile(t, "broken", `{"type":"compacted"}`, textRecord("user", strings.Repeat("λ", 300)))
	o := bounds()
	o.MaxChars = 256
	p, err := ReadJSONLTranscript(path, o)
	if err != nil {
		t.Fatal(err)
	}
	if p.Complete || p.MalformedRecords != 1 || !p.Messages[0].Truncated || len([]rune(p.Messages[0].Text)) != 256 {
		t.Fatalf("bad coverage/budget: %+v", p)
	}
	if len(p.Warnings) < 3 {
		t.Fatalf("missing warnings: %v", p.Warnings)
	}
}

func TestLiteralSearchFindsTextBeyondOutputBudget(t *testing.T) {
	path := transcriptFile(t, textRecord("user", strings.Repeat("prefix ", 10000)+"UNIQUE decision"))
	o := bounds()
	o.MaxChars = 1000
	o.Limit = 1
	o.LiteralQuery = "unique decision"
	p, err := ReadJSONLTranscript(path, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Messages) != 1 || !strings.Contains(p.Messages[0].Text, "UNIQUE decision") {
		t.Fatal("search missed late message content")
	}
}

func TestRecentAndTornScanBudgets(t *testing.T) {
	path := transcriptFile(t, textRecord("user", "one"), textRecord("assistant", strings.Repeat("x", 400)), textRecord("user", "last"))
	o := bounds()
	o.Tail = true
	o.Limit = 1
	p, err := ReadJSONLTranscript(path, o)
	if err != nil || len(p.Messages) != 1 || p.Messages[0].Text != "last" {
		t.Fatalf("recent: %+v %v", p, err)
	}
	o.Tail = false
	o.MaxBytes = 100
	p, err = ReadJSONLTranscript(path, o)
	if err != nil || p.NextOffset == nil || *p.NextOffset == 0 {
		t.Fatalf("torn scan: %+v %v", p, err)
	}
	o.Offset = *p.NextOffset
	if _, err = ReadJSONLTranscript(path, o); err == nil {
		t.Fatal("nonprogressing source budget must fail clearly")
	}
}

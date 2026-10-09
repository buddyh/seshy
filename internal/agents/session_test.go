package agents

import (
	"strings"
	"testing"
	"time"
)

func TestResumeArgv(t *testing.T) {
	s := Session{ID: "session-id", Path: "/tmp/session.jsonl"}
	tests := map[string][]string{
		"claude":   {"claude", "--resume", "session-id"},
		"codex":    {"codex", "resume", "session-id"},
		"grok":     {"grok", "--resume", "session-id"},
		"pi":       {"pi", "--session", "/tmp/session.jsonl"},
		"opencode": {"opencode", "-s", "session-id"},
		"agy":      {"agy", "--conversation=session-id"},
		"droid":    {"droid", "--resume", "session-id"},
	}
	for tool, want := range tests {
		s.Tool = tool
		got := ResumeArgv(s)
		if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Errorf("ResumeArgv(%q) = %#v, want %#v", tool, got, want)
		}
	}
}

func TestHandoffPromptContainsSessionContext(t *testing.T) {
	s := Session{
		Tool:  "claude",
		ID:    "6a3654a9-ec29-4da7-982c-5708ab505bca",
		Path:  "/Users/buddy/.claude/projects/demo/session.jsonl",
		Dir:   "/Users/buddy/repos/demo",
		Mtime: time.Unix(0, 0),
	}

	got := HandoffPrompt(s)
	for _, want := range []string{
		"Get up to speed on this Claude session and continue its work.",
		s.ID,
		s.Path,
		s.Dir,
		"original goal",
		"source of truth",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("HandoffPrompt() missing %q:\n%s", want, got)
		}
	}
}

func TestHandoffArgv(t *testing.T) {
	s := Session{Tool: "claude", ID: "session-id", Path: "/tmp/session.jsonl", Dir: "/tmp/project"}
	for tool, want := range map[string][]string{
		"claude":   {"claude"},
		"codex":    {"codex"},
		"grok":     {"grok"},
		"pi":       {"pi"},
		"droid":    {"droid"},
		"opencode": {"opencode", "--prompt"},
		"agy":      {"agy", "--prompt-interactive"},
	} {
		got := HandoffArgv(tool, s)
		if len(got) != len(want)+1 {
			t.Fatalf("HandoffArgv(%q) length = %d, want %d", tool, len(got), len(want)+1)
		}
		for i, arg := range want {
			if got[i] != arg {
				t.Errorf("HandoffArgv(%q)[%d] = %q, want %q", tool, i, got[i], arg)
			}
		}
		if got[len(got)-1] != HandoffPrompt(s) {
			t.Errorf("HandoffArgv(%q) did not append HandoffPrompt", tool)
		}
	}
	if got := HandoffArgv("unknown", s); got != nil {
		t.Errorf("HandoffArgv(unknown) = %#v, want nil", got)
	}
}

// Package agents discovers and resumes AI coding-agent sessions across tools.
package agents

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Session is one discovered agent session for a directory.
type Session struct {
	Tool    string    `json:"agent"`
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	Dir     string    `json:"dir"`
	Mtime   time.Time `json:"mtime"`
	Preview string    `json:"preview,omitempty"`

	// Optional metadata (filled when cheaply available).
	Model string  `json:"model,omitempty"`
	Cost  float64 `json:"cost,omitempty"`
	Msgs  int     `json:"msgs,omitempty"`

	// preFilled marks a preview already resolved by the collector (grok/opencode/agy).
	preFilled bool
}

// Meta describes a supported agent: display label and brand color (hex).
type Meta struct {
	Label string
	Hex   string // 24-bit hex, e.g. "#DE7356"
}

// Registry of agent display metadata. Order here is the canonical agent order.
// gemini, cursor, and copilot have no session collectors yet — they appear on
// retention-surface commands (store locations, disk usage, cleanup policies).
var Metas = map[string]Meta{
	"claude":   {"Claude", "#DE7356"},
	"codex":    {"Codex", "#10A37F"},
	"gemini":   {"Gemini", "#4796E3"},
	"grok":     {"Grok", "#FFFFFF"},
	"pi":       {"pi", "#BA3C3C"},
	"opencode": {"OpenCode", "#ABA198"},
	"agy":      {"agy", "#4285F4"},
	"droid":    {"Droid", "#9766F0"},
	"cursor":   {"Cursor", "#E5E5E5"},
	"copilot":  {"Copilot", "#79C0FF"},
}

// Label returns the display label for a tool key (falls back to the key).
func Label(tool string) string {
	if m, ok := Metas[tool]; ok {
		return m.Label
	}
	return tool
}

// ResumeArgv returns the command to resume a session, run from its Dir.
func ResumeArgv(s Session) []string {
	switch s.Tool {
	case "claude":
		return []string{"claude", "--resume", s.ID}
	case "codex":
		return []string{"codex", "resume", s.ID}
	case "grok":
		return []string{"grok", "--resume", s.ID}
	case "pi":
		return []string{"pi", "--session", s.Path}
	case "opencode":
		return []string{"opencode", "-s", s.ID}
	case "agy":
		return []string{"agy", "--conversation=" + s.ID}
	case "droid":
		return []string{"droid", "--resume", s.ID}
	}
	return nil
}

// HandoffPrompt returns a ready-to-paste prompt for continuing this session
// in a different coding agent. It names the source transcript, but tells the
// receiving agent to verify the workspace instead of trusting the transcript.
func HandoffPrompt(s Session) string {
	return strings.TrimSpace(fmt.Sprintf(`Get up to speed on this %s session and continue its work.

Session ID: %s
Transcript: %s
Working directory: %s

First recover the prior session's original goal, decisions, what actually landed, and the unresolved next step from the transcript. Then inspect the current workspace and treat it as the source of truth. Continue the unresolved work and verify the result.`, Label(s.Tool), s.ID, s.Path, s.Dir))
}

// HandoffArgv returns the command to start another coding agent with the
// handoff prompt as its first interactive message.
func HandoffArgv(tool string, s Session) []string {
	prompt := HandoffPrompt(s)
	switch tool {
	case "claude", "codex", "grok", "pi", "droid":
		return []string{tool, prompt}
	case "opencode":
		return []string{"opencode", "--prompt", prompt}
	case "agy":
		return []string{"agy", "--prompt-interactive", prompt}
	}
	return nil
}

// AvailableHandoffTargets returns installed supported agents other than the
// source agent, in the same stable order used by the session registry.
func AvailableHandoffTargets(source string) []string {
	var out []string
	for _, tool := range []string{"claude", "codex", "grok", "pi", "opencode", "agy", "droid"} {
		if tool == source {
			continue
		}
		if _, err := exec.LookPath(tool); err == nil {
			out = append(out, tool)
		}
	}
	return out
}

// Handoff starts another coding agent in the source session's directory with
// a prompt that reconstructs the source session before continuing its work.
func Handoff(s Session, target string) error {
	argv := HandoffArgv(target, s)
	if len(argv) == 0 {
		return fmt.Errorf("no handoff command for %s", target)
	}
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("%s not found on PATH", argv[0])
	}
	if s.Dir != "" {
		_ = os.Chdir(s.Dir)
	}
	return syscall.Exec(bin, argv, os.Environ())
}

// Resume replaces the current process with the agent's native resume command
// for s, run from its directory. It does not return on success.
func Resume(s Session) error {
	argv := ResumeArgv(s)
	if len(argv) == 0 {
		return fmt.Errorf("no resume command for %s", s.Tool)
	}
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("%s not found on PATH", argv[0])
	}
	if s.Dir != "" {
		_ = os.Chdir(s.Dir)
	}
	return syscall.Exec(bin, argv, os.Environ())
}

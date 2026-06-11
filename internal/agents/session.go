// Package agents discovers and resumes AI coding-agent sessions across tools.
package agents

import (
	"fmt"
	"os"
	"os/exec"
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
var Metas = map[string]Meta{
	"claude":   {"Claude", "#DE7356"},
	"codex":    {"Codex", "#10A37F"},
	"grok":     {"Grok", "#FFFFFF"},
	"pi":       {"pi", "#BA3C3C"},
	"opencode": {"OpenCode", "#ABA198"},
	"agy":      {"agy", "#4285F4"},
	"droid":    {"Droid", "#9766F0"},
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

// Package retention reports and manages each agent's local session
// retention: where sessions live, how much disk they use, and whether the
// agent deletes them on a timer. Two agents auto-delete by default (Claude
// Code and Gemini CLI, both after 30 days); the rest keep sessions forever.
//
// Every function takes home explicitly so tests can inject a fixture home;
// the CLI passes store.Home (which honors SESHY_HOME).
package retention

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Kind classifies what an agent does with old local sessions.
type Kind string

const (
	// AutoDelete: the agent deletes local sessions on a timer.
	AutoDelete Kind = "auto-delete"
	// KeepForever: the agent never deletes local sessions.
	KeepForever Kind = "keep-forever"
	// CloudManaged: local files are kept; a retention setting governs only
	// cloud-synced copies.
	CloudManaged Kind = "cloud-managed"
)

// Policy is one agent's retention policy as resolved from disk right now.
type Policy struct {
	Kind      Kind
	Default   string // behavior with no user config, e.g. "30d", "forever"
	Effective string // behavior in effect, e.g. "30d", "365d", "off"
	Source    string // "default" | "configured" | "disabled"
	AtRisk    bool   // auto-deletion is active with a finite horizon
	Note      string // one-line nuance (manual cleanup command, caveats)
	Err       error  // config file exists but is unreadable — never guess
}

// Source describes one agent's session store and retention knobs. Paths are
// funcs of home so env overrides (GROK_HOME, COPILOT_HOME, XDG_DATA_HOME)
// resolve at call time.
type Source struct {
	Key     string                     // agents.Metas key ("claude", "cursor", ...)
	Display func(home string) string   // primary store path for display
	Roots   func(home string) []string // store dirs/files to measure
	Config  func(home string) string   // retention config file ("" when none)
	Inspect func(home string) Policy
	Count   func(home string) (sessions int, oldest time.Time)
	Set     func(home, value string) (Change, error) // nil when not settable
}

// Change is one prepared edit to an agent's config file. Nothing touches
// disk until Apply, which backs the file up to <file>.bak first.
type Change struct {
	Agent  string
	File   string
	Key    string // dotted key path, e.g. "general.sessionRetention.maxAge"
	Before string // rendered current value, e.g. `30 (default, key unset)`
	After  string
	apply  func() error
}

// Apply writes the prepared change to disk.
func (c Change) Apply() error { return c.apply() }

// Row is one agent's fully-resolved line in the retention report.
type Row struct {
	Key       string
	Installed bool   // at least one store root exists
	Store     string // primary store path for display
	Bytes     int64
	Sessions  int
	Oldest    time.Time // zero when unknown / not installed
	Policy    Policy
	Config    string // resolved config file ("" when none)
	Settable  bool
}

// envOr returns the env value when set, else the fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func grokHome(home string) string {
	return envOr("GROK_HOME", filepath.Join(home, ".grok"))
}

func copilotHome(home string) string {
	return envOr("COPILOT_HOME", filepath.Join(home, ".copilot"))
}

func opencodeData(home string) string {
	return filepath.Join(envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share")), "opencode")
}

// under builds a Roots func for a single fixed path beneath home.
func under(parts ...string) func(string) []string {
	return func(h string) []string {
		return []string{filepath.Join(append([]string{h}, parts...)...)}
	}
}

// Sources is the canonical registry, in display order. Policy facts carry a
// citation comment; keep them in sync with upstream docs.
var Sources = []Source{
	{
		// Deletes transcripts older than cleanupPeriodDays (default 30) at
		// startup — https://code.claude.com/docs/en/settings
		Key:     "claude",
		Display: func(h string) string { return filepath.Join(h, ".claude", "projects") },
		Roots:   under(".claude", "projects"),
		Config:  func(h string) string { return filepath.Join(h, ".claude", "settings.json") },
		Inspect: inspectClaude,
		Count:   countClaude,
	},
	{
		// Rollouts are never auto-deleted (openai/codex#28187); [history]
		// keys in config.toml trim only history.jsonl.
		Key:     "codex",
		Display: func(h string) string { return filepath.Join(h, ".codex", "sessions") },
		Roots:   under(".codex", "sessions"),
		Inspect: keepForever("[history] in config.toml trims only history.jsonl, not sessions"),
		Count:   countCodex,
	},
	{
		// general.sessionRetention deletes chats after maxAge (default 30d,
		// on by default) — gemini-cli docs/cli/session-management.md
		Key:     "gemini",
		Display: func(h string) string { return filepath.Join(h, ".gemini", "tmp", "*", "chats") },
		Roots: func(h string) []string {
			m, _ := filepath.Glob(filepath.Join(h, ".gemini", "tmp", "*", "chats"))
			return m
		},
		Config:  func(h string) string { return filepath.Join(h, ".gemini", "settings.json") },
		Inspect: inspectGemini,
		Count:   countGemini,
	},
	{
		// No retention config exists — docs.x.ai/build/settings/reference
		Key:     "grok",
		Display: func(h string) string { return filepath.Join(grokHome(h), "sessions") },
		Roots:   func(h string) []string { return []string{filepath.Join(grokHome(h), "sessions")} },
		Inspect: keepForever("manual: grok sessions delete <id>"),
		Count:   countGrok,
	},
	{
		// No retention config exists — pi.dev/docs/latest/settings
		Key:     "pi",
		Display: func(h string) string { return filepath.Join(h, ".pi", "agent", "sessions") },
		Roots:   under(".pi", "agent", "sessions"),
		Inspect: keepForever("manual: delete from the /resume picker"),
		Count:   countPi,
	},
	{
		// Retention was explicitly declined upstream (opencode#22110).
		Key:     "opencode",
		Display: opencodeData,
		Roots:   func(h string) []string { return []string{opencodeData(h)} },
		Inspect: keepForever("manual: opencode session delete <id>"),
		Count:   countOpenCode,
	},
	{
		// Conversations are never auto-deleted; settings.json has no
		// retention keys — antigravity.google/docs/cli-settings
		Key:     "agy",
		Display: func(h string) string { return filepath.Join(h, ".gemini", "antigravity-cli") },
		Roots:   under(".gemini", "antigravity-cli"),
		Inspect: keepForever("manual: ctrl+delete in the /resume picker"),
		Count:   countAgy,
	},
	{
		// Local files are never deleted; sessionRetentionDays governs
		// Factory cloud copies only — docs.factory.ai/cli/configuration/settings
		Key:     "droid",
		Display: func(h string) string { return filepath.Join(h, ".factory", "sessions") },
		Roots:   under(".factory", "sessions"),
		Config:  func(h string) string { return filepath.Join(h, ".factory", "settings.json") },
		Inspect: inspectDroid,
		Count:   countDroid,
	},
	{
		// No auto-deletion, no retention setting (staff-confirmed on the
		// Cursor forum); manual cleanup only.
		Key:     "cursor",
		Display: func(h string) string { return filepath.Join(h, ".cursor", "chats") },
		Roots: func(h string) []string {
			// projects/ holds lots of non-session state; measure only the
			// per-project agent-transcripts dirs alongside the chat store.
			roots := []string{filepath.Join(h, ".cursor", "chats")}
			m, _ := filepath.Glob(filepath.Join(h, ".cursor", "projects", "*", "agent-transcripts"))
			return append(roots, m...)
		},
		Inspect: keepForever("manual: delete chats from cursor-agent ls"),
		Count:   countCursor,
	},
	{
		// Sessions are never auto-deleted (only logs are pruned); manual
		// /session prune --older-than DAYS — docs.github.com copilot-cli
		Key:     "copilot",
		Display: func(h string) string { return filepath.Join(copilotHome(h), "session-state") },
		Roots: func(h string) []string {
			ch := copilotHome(h)
			return []string{
				filepath.Join(ch, "session-state"),
				filepath.Join(ch, "history-session-state"),
				filepath.Join(ch, "session-store.db"),
			}
		},
		Inspect: keepForever("manual: /session prune --older-than DAYS"),
		Count:   countCopilot,
	},
}

// Keys lists the registry keys in display order.
func Keys() []string {
	out := make([]string, len(Sources))
	for i, s := range Sources {
		out[i] = s.Key
	}
	return out
}

// Report inspects every source concurrently and returns rows in registry order.
func Report(home string) []Row {
	rows := make([]Row, len(Sources))
	var wg sync.WaitGroup
	for i, src := range Sources {
		wg.Add(1)
		go func(i int, src Source) {
			defer wg.Done()
			rows[i] = buildRow(home, src)
		}(i, src)
	}
	wg.Wait()
	return rows
}

func buildRow(home string, src Source) Row {
	r := Row{Key: src.Key, Store: src.Display(home), Settable: src.Set != nil}
	if src.Config != nil {
		r.Config = src.Config(home)
	}
	roots := src.Roots(home)
	for _, root := range roots {
		if _, err := os.Stat(root); err == nil {
			r.Installed = true
		}
	}
	if r.Installed {
		for _, root := range roots {
			r.Bytes += dirBytes(root)
		}
		r.Sessions, r.Oldest = src.Count(home)
	}
	r.Policy = src.Inspect(home)
	return r
}

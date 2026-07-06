package retention

import (
	"strings"
	"testing"
)

// notesContain asserts one note mentions every fragment.
func notesContain(t *testing.T, notes []string, fragments ...string) {
	t.Helper()
	for _, note := range notes {
		ok := true
		for _, f := range fragments {
			if !strings.Contains(note, f) {
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}
	t.Errorf("no note contains %v in %q", fragments, notes)
}

func TestProtectPlanDefaults(t *testing.T) {
	home := t.TempDir() // no settings anywhere: both auto-deleters at 30d
	changes, notes := ProtectPlan(home, 365)

	if len(changes) != 2 {
		t.Fatalf("changes = %d (%+v), want 2 (claude + gemini)", len(changes), changes)
	}
	claude, gemini := changes[0], changes[1]
	if claude.Agent != "claude" || claude.Key != "cleanupPeriodDays" || claude.After != "365" {
		t.Errorf("claude change = %+v", claude)
	}
	if !strings.Contains(claude.Before, "30d (default, key unset)") {
		t.Errorf("claude before = %q", claude.Before)
	}
	if gemini.Agent != "gemini" || gemini.Key != "general.sessionRetention.maxAge" || gemini.After != `"365d"` {
		t.Errorf("gemini change = %+v", gemini)
	}
	notesContain(t, notes, "droid", "never deleted")
}

func TestProtectPlanSkipsSafeAgents(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".claude/settings.json", `{"cleanupPeriodDays": 500}`, tMid)
	writeFile(t, home, ".gemini/settings.json", `{"general": {"sessionRetention": {"enabled": false}}}`, tMid)

	changes, notes := ProtectPlan(home, 365)
	if len(changes) != 0 {
		t.Fatalf("changes = %+v, want none", changes)
	}
	notesContain(t, notes, "claude", "already keeps 500d")
	notesContain(t, notes, "gemini", "auto-delete already off")
}

func TestProtectPlanUnitsCompare(t *testing.T) {
	home := t.TempDir()
	// 52w = 364d < 365 -> still a change; 13m = 390d -> safe.
	writeFile(t, home, ".gemini/settings.json", `{"general": {"sessionRetention": {"maxAge": "52w"}}}`, tMid)
	changes, _ := ProtectPlan(home, 365)
	found := false
	for _, ch := range changes {
		if ch.Agent == "gemini" {
			found = true
		}
	}
	if !found {
		t.Error("52w (364d) must not satisfy 365d")
	}

	writeFile(t, home, ".gemini/settings.json", `{"general": {"sessionRetention": {"maxAge": "13m"}}}`, tMid)
	changes, notes := ProtectPlan(home, 365)
	for _, ch := range changes {
		if ch.Agent == "gemini" {
			t.Errorf("13m (390d) should be safe, got change %+v", ch)
		}
	}
	notesContain(t, notes, "gemini", "already keeps 13m")
}

func TestProtectPlanMalformedConfig(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".claude/settings.json", `{broken`, tMid)

	changes, notes := ProtectPlan(home, 365)
	// claude becomes a failure note; gemini is still planned.
	if len(changes) != 1 || changes[0].Agent != "gemini" {
		t.Fatalf("changes = %+v, want just gemini", changes)
	}
	notesContain(t, notes, "claude", "unreadable")
}

func TestProtectPlanApply(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".claude/settings.json", `{"model": "opus"}`, tMid)

	changes, _ := ProtectPlan(home, 365)
	for _, ch := range changes {
		if err := ch.Apply(); err != nil {
			t.Fatalf("%s: %v", ch.Agent, err)
		}
	}
	if p := inspectClaude(home); p.Effective != "365d" || p.Source != "configured" {
		t.Errorf("claude after apply = %+v", p)
	}
	if p := inspectGemini(home); p.Effective != "365d" || p.Source != "configured" {
		t.Errorf("gemini after apply = %+v", p)
	}
	// Second run: nothing left to do.
	if changes, _ := ProtectPlan(home, 365); len(changes) != 0 {
		t.Errorf("second run still plans %+v", changes)
	}
	// The pre-existing claude file must be backed up; sibling key intact.
	if got := claudeSettings(t, home); !strings.Contains(got, `"model": "opus"`) {
		t.Errorf("sibling key lost:\n%s", got)
	}
}

func TestApproxDays(t *testing.T) {
	cases := []struct {
		in   string
		days int
		ok   bool
	}{
		{"30d", 30, true}, {"48h", 2, true}, {"52w", 364, true}, {"12m", 360, true},
		{"off", 0, false}, {"forever", 0, false}, {"", 0, false}, {"x", 0, false},
	}
	for _, tc := range cases {
		d, ok := approxDays(tc.in)
		if d != tc.days || ok != tc.ok {
			t.Errorf("approxDays(%q) = %d,%t want %d,%t", tc.in, d, ok, tc.days, tc.ok)
		}
	}
}

package retention

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func claudeSettings(t *testing.T, home string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func geminiSettings(t *testing.T, home string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSetClaude(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".claude/settings.json", `{"model": "opus", "cleanupPeriodDays": 30}`, tMid)

	ch, err := Set(home, "claude", "365")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Key != "cleanupPeriodDays" || ch.After != "365" {
		t.Errorf("change = %+v", ch)
	}
	if !strings.Contains(ch.Before, "30d (configured)") {
		t.Errorf("before = %q, want the current 30d value", ch.Before)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	got := claudeSettings(t, home)
	if !strings.Contains(got, `"cleanupPeriodDays": 365`) || !strings.Contains(got, `"model": "opus"`) {
		t.Errorf("settings after apply:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json.bak")); err != nil {
		t.Error("backup missing after apply")
	}
}

func TestSetClaudeNormalizesAndOff(t *testing.T) {
	home := t.TempDir()

	ch, err := Set(home, "claude", "365d")
	if err != nil {
		t.Fatal(err)
	}
	if ch.After != "365" {
		t.Errorf(`"365d" normalized to %q, want "365"`, ch.After)
	}

	ch, err = Set(home, "claude", "off")
	if err != nil {
		t.Fatal(err)
	}
	if ch.After != "99999" {
		t.Errorf(`off maps to %q, want "99999"`, ch.After)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	if p := inspectClaude(home); p.Effective != "off" || p.AtRisk {
		t.Errorf("after off: policy = %+v", p)
	}
}

func TestSetClaudeRejectsBadValues(t *testing.T) {
	home := t.TempDir()
	for _, bad := range []string{"0", "0d", "-5", "abc", ""} {
		if _, err := Set(home, "claude", bad); err == nil {
			t.Errorf("Set(claude, %q) succeeded, want error", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Error("rejected values must not create a settings file")
	}
}

func TestSetClaudeCreatesMissingFile(t *testing.T) {
	home := t.TempDir()
	ch, err := Set(home, "claude", "400")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ch.Before, "default, key unset") {
		t.Errorf("before = %q, want it to note the default", ch.Before)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	if got := claudeSettings(t, home); !strings.Contains(got, `"cleanupPeriodDays": 400`) {
		t.Errorf("settings after apply:\n%s", got)
	}
}

func TestSetClaudeRefusesMalformed(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".claude/settings.json", `{broken`, tMid)
	if _, err := Set(home, "claude", "365"); err == nil {
		t.Fatal("want an error when settings.json is malformed")
	}
}

func TestSetGeminiMaxAge(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".gemini/settings.json",
		`{"theme": "dark", "general": {"sessionRetention": {"enabled": false, "minRetention": "1d"}}}`, tMid)

	ch, err := Set(home, "gemini", "365d")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Key != "general.sessionRetention.maxAge" || ch.After != `"365d"` {
		t.Errorf("change = %+v", ch)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	got := geminiSettings(t, home)
	for _, want := range []string{`"maxAge": "365d"`, `"enabled": true`, `"minRetention": "1d"`, `"theme": "dark"`} {
		if !strings.Contains(got, want) {
			t.Errorf("settings missing %s:\n%s", want, got)
		}
	}
	if p := inspectGemini(home); p.Effective != "365d" || p.Source != "configured" {
		t.Errorf("after apply: policy = %+v", p)
	}
}

func TestSetGeminiOff(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".gemini/settings.json",
		`{"general": {"sessionRetention": {"maxAge": "30d"}}}`, tMid)

	ch, err := Set(home, "gemini", "off")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Key != "general.sessionRetention.enabled" || ch.After != "false" {
		t.Errorf("change = %+v", ch)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	got := geminiSettings(t, home)
	// off flips the switch but leaves maxAge for a later re-enable.
	for _, want := range []string{`"enabled": false`, `"maxAge": "30d"`} {
		if !strings.Contains(got, want) {
			t.Errorf("settings missing %s:\n%s", want, got)
		}
	}
	if p := inspectGemini(home); p.Effective != "off" || p.AtRisk {
		t.Errorf("after off: policy = %+v", p)
	}
}

func TestSetGeminiBareDaysAndUnits(t *testing.T) {
	home := t.TempDir()
	for value, want := range map[string]string{"365": `"365d"`, "52w": `"52w"`, "12m": `"12m"`, "48h": `"48h"`} {
		ch, err := Set(home, "gemini", value)
		if err != nil {
			t.Errorf("Set(gemini, %q): %v", value, err)
			continue
		}
		if ch.After != want {
			t.Errorf("Set(gemini, %q) after = %s, want %s", value, ch.After, want)
		}
	}
	for _, bad := range []string{"90x", "0d", "0", "-3d", "d", ""} {
		if _, err := Set(home, "gemini", bad); err == nil {
			t.Errorf("Set(gemini, %q) succeeded, want error", bad)
		}
	}
}

func TestSetGeminiCreatesNestedPath(t *testing.T) {
	home := t.TempDir()
	ch, err := Set(home, "gemini", "400d")
	if err != nil {
		t.Fatal(err)
	}
	if err := ch.Apply(); err != nil {
		t.Fatal(err)
	}
	if p := inspectGemini(home); p.Effective != "400d" {
		t.Errorf("policy after create = %+v", p)
	}
}

func TestSetGeminiRefusesNonObject(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".gemini/settings.json", `{"general": "oops"}`, tMid)
	// The malformed shape surfaces at plan time (inspect reports it), so
	// nothing is ever written.
	if _, err := Set(home, "gemini", "365d"); err == nil || !strings.Contains(err.Error(), "want an object") {
		t.Fatalf("err = %v, want a want-an-object refusal", err)
	}
	got := geminiSettings(t, home)
	if got != `{"general": "oops"}` {
		t.Errorf("settings were modified: %s", got)
	}
}

func TestSetNotSettable(t *testing.T) {
	home := t.TempDir()
	for _, key := range []string{"codex", "grok", "pi", "opencode", "agy", "cursor", "copilot"} {
		_, err := Set(home, key, "365")
		if err == nil || !strings.Contains(err.Error(), "nothing to set") {
			t.Errorf("Set(%s) err = %v, want a nothing-to-set explanation", key, err)
		}
	}
	if _, err := Set(home, "droid", "365"); err == nil || !strings.Contains(err.Error(), "cloud") {
		t.Errorf("Set(droid) err = %v, want the cloud-only explanation", err)
	}
	if _, err := Set(home, "nope", "365"); err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("Set(nope) err = %v, want unknown-agent listing keys", err)
	}
}

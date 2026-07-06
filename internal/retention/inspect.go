package retention

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// claudeOffDays is the community "off" convention: Claude Code has no disable
// switch for its cleanup sweep, so "off" writes a horizon of ~274 years.
// Never write 0 — current versions reject it, and older versions treated it
// as "disable session persistence entirely" (anthropics/claude-code#23710).
const claudeOffDays = 99999

// geminiAgeRe validates Gemini's maxAge duration strings. The unit "m" means
// MONTHS (30 days), not minutes — an easy footgun.
var geminiAgeRe = regexp.MustCompile(`^(\d+)([dhwm])$`)

// keepForever builds an Inspect func for agents with no retention mechanism.
func keepForever(note string) func(string) Policy {
	p := Policy{Kind: KeepForever, Default: "forever", Effective: "forever", Source: "default", Note: note}
	return func(string) Policy { return p }
}

// inspectClaude resolves Claude Code's cleanupPeriodDays from the user-level
// settings file. Project, local, and managed settings can override it; v1
// reads user scope only (documented limitation).
func inspectClaude(home string) Policy {
	p := Policy{
		Kind: AutoDelete, Default: "30d", Effective: "30d", Source: "default", AtRisk: true,
		Note: "startup sweep also removes per-session plans, file history, and todos",
	}
	m, err := readJSONFile(filepath.Join(home, ".claude", "settings.json"))
	if os.IsNotExist(err) {
		return p
	}
	if err != nil {
		p.Err = err
		return p
	}
	v, ok := m["cleanupPeriodDays"]
	if !ok {
		return p
	}
	n, err := jsonInt(v)
	if err != nil {
		p.Err = fmt.Errorf("cleanupPeriodDays: %w", err)
		return p
	}
	if n >= claudeOffDays {
		p.Effective, p.Source, p.AtRisk = "off", "disabled", false
		return p
	}
	p.Effective, p.Source = fmt.Sprintf("%dd", n), "configured"
	return p
}

// inspectGemini resolves Gemini CLI's general.sessionRetention. Cleanup is on
// by default (30d) since March 2026, so a missing key still means auto-delete.
func inspectGemini(home string) Policy {
	p := Policy{
		Kind: AutoDelete, Default: "30d", Effective: "30d", Source: "default", AtRisk: true,
		Note: "deletes each chat with its plans, tool outputs, and logs",
	}
	m, err := readJSONFile(filepath.Join(home, ".gemini", "settings.json"))
	if os.IsNotExist(err) {
		return p
	}
	if err != nil {
		p.Err = err
		return p
	}
	sr, ok, err := childObject(m, "general", "sessionRetention")
	if err != nil {
		p.Err = err
		return p
	}
	if !ok {
		return p
	}
	if enabled, present := sr["enabled"]; present {
		if b, isBool := enabled.(bool); isBool && !b {
			p.Effective, p.Source, p.AtRisk = "off", "disabled", false
			return p
		}
	}
	if age, present := sr["maxAge"]; present {
		s, isStr := age.(string)
		if !isStr || !geminiAgeRe.MatchString(s) {
			p.Err = fmt.Errorf("sessionRetention.maxAge: want a duration like \"30d\", got %v", age)
			return p
		}
		p.Effective, p.Source = s, "configured"
	}
	if count, present := sr["maxCount"]; present {
		if n, err := jsonInt(count); err == nil {
			p.Note = fmt.Sprintf("also caps at %d sessions; %s", n, p.Note)
		}
	}
	return p
}

// inspectDroid reports Factory's cloud-side retention. Local session files
// are never deleted; sessionRetentionDays (14-365) governs only cloud-synced
// copies, so seshy displays it but does not manage it.
func inspectDroid(home string) Policy {
	p := Policy{
		Kind: CloudManaged, Default: "forever", Effective: "local: forever · cloud: Factory-managed",
		Source: "default", Note: "sessionRetentionDays governs Factory cloud copies only",
	}
	m, err := readJSONFile(filepath.Join(home, ".factory", "settings.json"))
	if os.IsNotExist(err) {
		return p
	}
	if err != nil {
		p.Err = err
		return p
	}
	if sync, present := m["cloudSessionSync"]; present {
		if b, isBool := sync.(bool); isBool && !b {
			p.Effective, p.Source = "local: forever · cloud: sync off", "configured"
			return p
		}
	}
	if v, present := m["sessionRetentionDays"]; present {
		if n, err := jsonInt(v); err == nil {
			p.Effective, p.Source = fmt.Sprintf("local: forever · cloud: %dd", n), "configured"
		}
	}
	return p
}

// jsonInt coerces a decoded JSON value (json.Number from our UseNumber
// decoder, or a raw float64) to a whole int.
func jsonInt(v any) (int, error) {
	switch n := v.(type) {
	case json.Number:
		i, err := strconv.ParseInt(n.String(), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("want a whole number, got %v", v)
		}
		return int(i), nil
	case float64:
		if n != float64(int(n)) {
			return 0, fmt.Errorf("want a whole number, got %v", v)
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("want a number, got %T", v)
	}
}

// childObject walks nested objects by key, distinguishing "missing" (ok
// false) from "present but not an object" (err).
func childObject(m map[string]any, path ...string) (map[string]any, bool, error) {
	cur := m
	for i, key := range path {
		v, present := cur[key]
		if !present {
			return nil, false, nil
		}
		child, isObj := v.(map[string]any)
		if !isObj {
			return nil, false, fmt.Errorf("%s: want an object, got %T", joinPath(path[:i+1]), v)
		}
		cur = child
	}
	return cur, true, nil
}

func joinPath(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "." + p
	}
	return out
}

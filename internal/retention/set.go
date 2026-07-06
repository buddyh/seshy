package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Set prepares (but does not apply) the retention change for one agent.
// Values: a day count ("365", "365d") or "off". Only agents that auto-delete
// locally are settable; everything else returns an explanatory error.
func Set(home, agent, value string) (Change, error) {
	for _, s := range Sources {
		if s.Key != agent {
			continue
		}
		if s.Set == nil {
			return Change{}, notSettable(s)
		}
		return s.Set(home, value)
	}
	return Change{}, fmt.Errorf("unknown agent %q (want one of %s)", agent, strings.Join(Keys(), ", "))
}

func notSettable(s Source) error {
	if s.Key == "droid" {
		return fmt.Errorf("droid's sessionRetentionDays governs Factory cloud copies, not local files; " +
			"local sessions are kept forever — nothing for seshy to manage")
	}
	return fmt.Errorf("%s never auto-deletes sessions; nothing to set", s.Key)
}

// parseDays accepts "365" or "365d" and rejects anything that is not a
// positive whole day count. 0 is rejected explicitly: current Claude Code
// versions error on it, and historically it disabled session persistence
// entirely (anthropics/claude-code#23710).
func parseDays(value string) (int, error) {
	v := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "d")
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("want a day count like 365, 365d, or off; got %q", value)
	}
	if n == 0 {
		return 0, fmt.Errorf("0 is not a valid retention period (Claude Code rejects it, " +
			"and old versions treated it as \"never save sessions\"); use off to keep sessions indefinitely")
	}
	if n < 0 {
		return 0, fmt.Errorf("retention days must be positive, got %d", n)
	}
	return n, nil
}

// setClaude prepares a cleanupPeriodDays write to ~/.claude/settings.json.
func setClaude(home, value string) (Change, error) {
	n := claudeOffDays
	if strings.ToLower(strings.TrimSpace(value)) != "off" {
		var err error
		if n, err = parseDays(value); err != nil {
			return Change{}, err
		}
	}
	path := filepath.Join(home, ".claude", "settings.json")
	cur := inspectClaude(home)
	if cur.Err != nil {
		return Change{}, fmt.Errorf("%s is unreadable; fix it before letting seshy edit it: %v", path, cur.Err)
	}
	ch := Change{
		Agent:   "claude",
		File:    path,
		Key:     "cleanupPeriodDays",
		Before:  describeCurrent(cur),
		After:   strconv.Itoa(n),
		NewFile: fileMissing(path),
	}
	ch.apply = func() error {
		return mutateJSON(path, true, func(m map[string]any) error {
			if n < 1 {
				return fmt.Errorf("refusing to write cleanupPeriodDays %d", n)
			}
			m["cleanupPeriodDays"] = n
			return nil
		})
	}
	return ch, nil
}

// setGemini prepares a general.sessionRetention write to
// ~/.gemini/settings.json: "off" flips enabled to false; a duration sets
// maxAge (and enabled true, since a horizon expresses intent to retain).
func setGemini(home, value string) (Change, error) {
	path := filepath.Join(home, ".gemini", "settings.json")
	cur := inspectGemini(home)
	if cur.Err != nil {
		return Change{}, fmt.Errorf("%s is unreadable; fix it before letting seshy edit it: %v", path, cur.Err)
	}
	ch := Change{Agent: "gemini", File: path, Before: describeCurrent(cur), NewFile: fileMissing(path)}

	if strings.ToLower(strings.TrimSpace(value)) == "off" {
		ch.Key, ch.After = "general.sessionRetention.enabled", "false"
		ch.apply = func() error {
			return mutateJSON(path, true, func(m map[string]any) error {
				sr, err := ensureObject(m, "general", "sessionRetention")
				if err != nil {
					return err
				}
				sr["enabled"] = false
				return nil
			})
		}
		return ch, nil
	}

	age := strings.TrimSpace(value)
	if _, err := strconv.Atoi(age); err == nil {
		age += "d" // bare day count
	}
	if !geminiAgeRe.MatchString(age) {
		return Change{}, fmt.Errorf("want a Gemini duration like 365d, 52w, or 12m (m = months!), or off; got %q", value)
	}
	if strings.TrimLeft(age[:len(age)-1], "0") == "" {
		return Change{}, fmt.Errorf("retention must be longer than zero, got %q", value)
	}
	ch.Key, ch.After = "general.sessionRetention.maxAge", fmt.Sprintf("%q", age)
	ch.apply = func() error {
		return mutateJSON(path, true, func(m map[string]any) error {
			sr, err := ensureObject(m, "general", "sessionRetention")
			if err != nil {
				return err
			}
			sr["maxAge"] = age
			sr["enabled"] = true
			return nil
		})
	}
	return ch, nil
}

func fileMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// describeCurrent renders a Policy's effective value for a Before field,
// e.g. `30d (default, key unset)` or `off (disabled)`.
func describeCurrent(p Policy) string {
	switch p.Source {
	case "default":
		return fmt.Sprintf("%s (default, key unset)", p.Effective)
	default:
		return fmt.Sprintf("%s (%s)", p.Effective, p.Source)
	}
}

// ensureObject walks (creating as needed) nested objects by key. A key that
// exists with a non-object value aborts rather than clobbering user config.
func ensureObject(m map[string]any, path ...string) (map[string]any, error) {
	cur := m
	for i, key := range path {
		v, present := cur[key]
		if !present {
			child := map[string]any{}
			cur[key] = child
			cur = child
			continue
		}
		child, isObj := v.(map[string]any)
		if !isObj {
			return nil, fmt.Errorf("%s: want an object, found %T — fix the file by hand", joinPath(path[:i+1]), v)
		}
		cur = child
	}
	return cur, nil
}

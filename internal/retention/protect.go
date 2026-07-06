package retention

import (
	"fmt"
	"strconv"
	"strings"
)

// ProtectPlan returns the changes that raise every auto-deleting agent to at
// least days of retention, plus notes for agents skipped (already safe,
// unreadable config, or cloud-managed).
func ProtectPlan(home string, days int) (changes []Change, notes []string) {
	for _, s := range Sources {
		p := s.Inspect(home)
		switch {
		case p.Kind == CloudManaged:
			notes = append(notes, fmt.Sprintf(
				"%s: cloud retention is managed by the vendor; local sessions are never deleted", s.Key))
		case p.Kind != AutoDelete:
			// keeps forever — nothing to protect
		case p.Err != nil:
			notes = append(notes, fmt.Sprintf("%s: config is unreadable; fix it, then re-run (%v)", s.Key, p.Err))
		case !p.AtRisk:
			notes = append(notes, fmt.Sprintf("%s: auto-delete already off", s.Key))
		default:
			if d, ok := approxDays(p.Effective); ok && d >= days {
				notes = append(notes, fmt.Sprintf("%s: already keeps %s (>= %dd)", s.Key, p.Effective, days))
				continue
			}
			// Every settable auto-deleter accepts "<n>d".
			ch, err := s.Set(home, fmt.Sprintf("%dd", days))
			if err != nil {
				notes = append(notes, fmt.Sprintf("%s: %v", s.Key, err))
				continue
			}
			changes = append(changes, ch)
		}
	}
	return changes, notes
}

// approxDays maps an effective retention value to days for horizon
// comparisons: h rounds down, w is 7d, and m is Gemini months (30d). The
// approximation only decides whether protect can skip an agent.
func approxDays(effective string) (int, bool) {
	if effective == "" || effective == "off" || effective == "forever" {
		return 0, false
	}
	unit := effective[len(effective)-1]
	n, err := strconv.Atoi(strings.TrimSpace(effective[:len(effective)-1]))
	if err != nil {
		return 0, false
	}
	switch unit {
	case 'd':
		return n, true
	case 'h':
		return n / 24, true
	case 'w':
		return n * 7, true
	case 'm':
		return n * 30, true
	default:
		return 0, false
	}
}

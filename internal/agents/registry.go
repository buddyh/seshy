package agents

import (
	"sort"
	"sync"
)

type collector = func(target string, sub, global bool) []Session

var collectors = []collector{
	collectClaude, collectCodex, collectGrok, collectPi,
	collectOpenCode, collectAgy, collectDroid,
}

// Filter controls which sessions are hidden from every listing. It is set once
// from user config at startup; the zero value hides nothing (default behavior).
var Filter struct {
	// HideClaudeHeadless drops Claude sessions started with `claude -p` / the
	// Agent SDK (entrypoint "sdk-cli").
	HideClaudeHeadless bool
	// HideCodexExec drops Codex sessions started via `codex exec` (source "exec").
	HideCodexExec bool
}

// Collect lists sessions for a single directory (newest-first), keeping the
// newest num per agent. Pass sub to include subdirectories.
func Collect(target string, num int, sub bool) []Session {
	return gather(num, 0, true, func(c collector) []Session { return c(target, sub, false) })
}

// CollectGlobal lists the most-recent sessions across ALL directories on the
// machine — every agent, every repo — newest-first, capped at num overall.
func CollectGlobal(num int) []Session {
	return gather(num, num, true, func(c collector) []Session { return c("", false, true) })
}

// CollectIndex lists every session across all agents and directories,
// newest-first and uncapped, WITHOUT reading file contents for previews. It is
// the fast index meant for tooling (search skills, pipelines) that only needs
// each session's path, agent, dir, mtime, id, and resume command.
func CollectIndex() []Session {
	return gather(0, 0, false, func(c collector) []Session { return c("", false, true) })
}

// gather runs every collector concurrently via call, keeps the newest perAgent
// rows per agent (0 = unlimited), merges newest-first, trims to overall
// (0 = no cap), then fills previews for the survivors.
func gather(perAgent, overall int, fillPrev bool, call func(collector) []Session) []Session {
	results := make([][]Session, len(collectors))
	var wg sync.WaitGroup
	for i, c := range collectors {
		wg.Add(1)
		go func(i int, c collector) {
			defer wg.Done()
			rows := call(c)
			sort.SliceStable(rows, func(a, b int) bool { return rows[a].Mtime.After(rows[b].Mtime) })
			if perAgent > 0 && len(rows) > perAgent {
				rows = rows[:perAgent]
			}
			results[i] = rows
		}(i, c)
	}
	wg.Wait()

	var merged []Session
	for _, r := range results {
		merged = append(merged, r...)
	}
	sort.SliceStable(merged, func(a, b int) bool { return merged[a].Mtime.After(merged[b].Mtime) })
	if overall > 0 && len(merged) > overall {
		merged = merged[:overall]
	}
	if !fillPrev {
		return merged
	}
	FillPreviews(merged)
	return merged
}

// FillPreviews resolves the first-user-message preview for each session in ss
// that does not already have one, concurrently and in place. Sessions whose
// preview was already filled by their collector are left untouched.
func FillPreviews(ss []Session) {
	var pw sync.WaitGroup
	sem := make(chan struct{}, 24)
	for i := range ss {
		if ss[i].preFilled || ss[i].Preview != "" {
			continue
		}
		pw.Add(1)
		go func(i int) {
			defer pw.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ss[i].Preview = FirstJSONLUser(ss[i].Path)
		}(i)
	}
	pw.Wait()
}

package agents

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/buddyh/seshy/internal/store"
)

func collectClaude(target string, sub, global bool) []Session {
	root := filepath.Join(store.Home, ".claude", "projects")
	var folders []string
	switch {
	case global:
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			if e.IsDir() {
				folders = append(folders, filepath.Join(root, e.Name()))
			}
		}
	case sub:
		enc := store.EncodeClaude(target)
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			if e.IsDir() && (e.Name() == enc || strings.HasPrefix(e.Name(), enc+"-")) {
				folders = append(folders, filepath.Join(root, e.Name()))
			}
		}
	default:
		enc := store.EncodeClaude(target)
		f := filepath.Join(root, enc)
		if fi, err := os.Stat(f); err == nil && fi.IsDir() {
			folders = append(folders, f)
		}
	}
	var out []Session
	for _, folder := range folders {
		dir := target
		if global {
			dir = claudeDir(filepath.Base(folder))
		}
		for _, jf := range store.Glob(filepath.Join(folder, "*.jsonl")) {
			if strings.HasPrefix(filepath.Base(jf), "agent-") {
				continue
			}
			fi, err := os.Stat(jf)
			if err != nil {
				continue
			}
			out = append(out, Session{
				Tool: "claude", ID: strings.TrimSuffix(filepath.Base(jf), ".jsonl"),
				Path: jf, Dir: dir, Mtime: fi.ModTime(),
			})
		}
	}
	if Filter.HideClaudeHeadless {
		out = dropClaudeHeadless(out)
	}
	return out
}

// dropClaudeHeadless removes sessions started headless (claude -p / Agent SDK),
// detected concurrently by reading each session's entrypoint field.
func dropClaudeHeadless(ss []Session) []Session {
	headless := make([]bool, len(ss))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 24)
	for i := range ss {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			headless[i] = claudeHeadless(ss[i].Path)
		}(i)
	}
	wg.Wait()
	out := ss[:0]
	for i, s := range ss {
		if !headless[i] {
			out = append(out, s)
		}
	}
	return out
}

// claudeHeadless reports whether a Claude session was started headless. The
// per-record "entrypoint" field is "cli" for the interactive TUI and "sdk-cli"
// for `claude -p` / the Agent SDK. It first appears a few lines in (after the
// opening queue/mode records), so scan a small window.
func claudeHeadless(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for i := 0; i < 16 && sc.Scan(); i++ {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"entrypoint"`)) {
			continue
		}
		var d struct {
			Entrypoint string `json:"entrypoint"`
		}
		if json.Unmarshal(line, &d) == nil && d.Entrypoint != "" {
			return d.Entrypoint != "cli"
		}
	}
	return false
}

// claudeDir reconstructs the absolute directory from a Claude project-folder
// name. Claude encodes the cwd by replacing every non-alphanumeric character
// with "-", which is lossy: a literal "-", "_", or "." in the path collapses to
// the same "-" as the path separators. So "/Users/b/repos/monitor-sizer" and
// "/Users/b/repos/monitor/sizer" encode identically, and a naive slash-decode
// turns the repo "monitor-sizer" into ".../monitor/sizer" (basename "sizer").
//
// We recover the true path by resolving the encoded name against the
// filesystem: at each level we look for the real child directory whose own
// encoded form matches the next run of tokens. This is exact whenever the
// directory still exists on disk; otherwise we fall back to the naive decode.
func claudeDir(folder string) string {
	enc := strings.TrimPrefix(folder, "-")
	if enc == "" {
		return "/"
	}
	if p, ok := resolveClaude("/", strings.Split(enc, "-")); ok {
		return p
	}
	return "/" + strings.ReplaceAll(enc, "-", "/")
}

// encChild is a real directory entry paired with the tokens its name encodes to.
type encChild struct {
	name   string
	tokens []string
}

// childCache memoizes the encoded children of a base dir for the process
// lifetime — folders share prefixes (e.g. ~/repos), so this avoids re-reading
// the same directory once per session folder during a global scan.
var childCache sync.Map // base path -> []encChild

func encChildren(base string) []encChild {
	if v, ok := childCache.Load(base); ok {
		return v.([]encChild)
	}
	entries, _ := os.ReadDir(base)
	cs := make([]encChild, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			cs = append(cs, encChild{e.Name(), strings.Split(store.EncodeClaude(e.Name()), "-")})
		}
	}
	childCache.Store(base, cs)
	return cs
}

// resolveClaude maps encoded tokens onto an existing path under base by matching
// each real subdirectory against the tokens it would encode to. Longer matches
// are tried first (so "monitor-sizer" wins over a sibling "monitor"), with
// backtracking. Returns the resolved path once every token is consumed.
func resolveClaude(base string, tokens []string) (string, bool) {
	if len(tokens) == 0 {
		return base, true
	}
	cands := encChildren(base)
	// Prefer children that consume more tokens to minimize backtracking.
	sort.Slice(cands, func(a, b int) bool { return len(cands[a].tokens) > len(cands[b].tokens) })
	for _, c := range cands {
		n := len(c.tokens)
		if n == 0 || n > len(tokens) {
			continue
		}
		match := true
		for i := 0; i < n; i++ {
			if c.tokens[i] != tokens[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if p, ok := resolveClaude(filepath.Join(base, c.name), tokens[n:]); ok {
			return p, true
		}
	}
	return "", false
}

func collectCodex(target string, sub, global bool) []Session {
	root := filepath.Join(store.Home, ".codex", "sessions")
	target = store.TrimSlash(target)
	var paths []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), ".jsonl") {
			paths = append(paths, p)
		}
		return nil
	})
	var out []Session
	for _, r := range store.ScanFirstLines(paths, 48) {
		if !strings.Contains(r.First, `"cwd"`) {
			continue
		}
		var d struct {
			Payload struct {
				ID  string `json:"id"`
				Cwd string `json:"cwd"`
				// Source is "cli", "exec", "vscode", or an object for subagents
				// — RawMessage so an object value does not fail the unmarshal.
				Source json.RawMessage `json:"source"`
			} `json:"payload"`
		}
		if json.Unmarshal([]byte(r.First), &d) != nil {
			continue
		}
		if Filter.HideCodexExec && string(d.Payload.Source) == `"exec"` {
			continue
		}
		cwd := store.TrimSlash(d.Payload.Cwd)
		if !global && cwd != target && !(sub && strings.HasPrefix(cwd, target+"/")) {
			continue
		}
		fi, err := os.Stat(r.Path)
		if err != nil {
			continue
		}
		id := d.Payload.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(r.Path), ".jsonl")
		}
		out = append(out, Session{Tool: "codex", ID: id, Path: r.Path, Dir: cwd, Mtime: fi.ModTime()})
	}
	return out
}

func collectPi(target string, sub, global bool) []Session {
	root := filepath.Join(store.Home, ".pi", "agent", "sessions")
	target = store.TrimSlash(target)
	paths := store.Glob(filepath.Join(root, "*", "*.jsonl"))
	var out []Session
	for _, r := range store.ScanFirstLines(paths, 48) {
		if !strings.Contains(r.First, `"cwd"`) {
			continue
		}
		var d struct {
			ID  string `json:"id"`
			Cwd string `json:"cwd"`
		}
		if json.Unmarshal([]byte(r.First), &d) != nil {
			continue
		}
		cwd := store.TrimSlash(d.Cwd)
		if !global && cwd != target && !(sub && strings.HasPrefix(cwd, target+"/")) {
			continue
		}
		fi, err := os.Stat(r.Path)
		if err != nil {
			continue
		}
		id := d.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(r.Path), ".jsonl")
		}
		out = append(out, Session{Tool: "pi", ID: id, Path: r.Path, Dir: cwd, Mtime: fi.ModTime()})
	}
	return out
}

func collectDroid(target string, sub, global bool) []Session {
	root := filepath.Join(store.Home, ".factory", "sessions")
	target = store.TrimSlash(target)
	paths := store.Glob(filepath.Join(root, "*", "*.jsonl"))
	var out []Session
	for _, r := range store.ScanFirstLines(paths, 48) {
		if !strings.Contains(r.First, `"cwd"`) {
			continue
		}
		var d struct {
			ID           string `json:"id"`
			Cwd          string `json:"cwd"`
			Title        string `json:"title"`
			SessionTitle string `json:"sessionTitle"`
		}
		if json.Unmarshal([]byte(r.First), &d) != nil {
			continue
		}
		cwd := store.TrimSlash(d.Cwd)
		if !global && cwd != target && !(sub && strings.HasPrefix(cwd, target+"/")) {
			continue
		}
		fi, err := os.Stat(r.Path)
		if err != nil {
			continue
		}
		id := d.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(r.Path), ".jsonl")
		}
		s := Session{Tool: "droid", ID: id, Path: r.Path, Dir: cwd, Mtime: fi.ModTime()}
		title := strings.TrimSpace(d.SessionTitle)
		if title == "" {
			title = strings.TrimSpace(d.Title)
		}
		if lt := strings.ToLower(title); title != "" && lt != "new session" && lt != "untitled" {
			s.Preview, s.preFilled = title, true
		}
		out = append(out, s)
	}
	return out
}

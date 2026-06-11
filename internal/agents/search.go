package agents

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Match is a session whose content matched a search, with an excerpt.
type Match struct {
	Session
	Snippet string `json:"snippet"`
}

// SearchOpts configures a content search across sessions.
type SearchOpts struct {
	Pattern    string
	Dir        string // "" = every directory on the machine
	Sub        bool
	Agent      string
	IgnoreCase bool
	Regex      bool
	Limit      int // 0 = all
}

// Search scans session contents for Pattern across the selected agents and
// directories, returning matches newest-first with an excerpt around the first
// hit. Text stores (Claude/Codex/pi/Droid jsonl, Grok json) are scanned on
// disk; binary stores (OpenCode/agy SQLite) are matched against their title.
func Search(o SearchOpts) ([]Match, error) {
	var re *regexp.Regexp
	if o.Regex {
		expr := o.Pattern
		if o.IgnoreCase {
			expr = "(?i)" + expr
		}
		r, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		re = r
	}
	needle := o.Pattern
	if o.IgnoreCase {
		needle = strings.ToLower(needle)
	}

	match := func(s string) bool {
		switch {
		case re != nil:
			return re.MatchString(s)
		case o.IgnoreCase:
			return strings.Contains(strings.ToLower(s), needle)
		default:
			return strings.Contains(s, o.Pattern)
		}
	}
	pos := func(line string) int {
		switch {
		case re != nil:
			if loc := re.FindStringIndex(line); loc != nil {
				return loc[0]
			}
			return -1
		case o.IgnoreCase:
			return strings.Index(strings.ToLower(line), needle)
		default:
			return strings.Index(line, o.Pattern)
		}
	}

	var sessions []Session
	if o.Dir == "" {
		sessions = CollectIndex()
	} else {
		sessions = Collect(o.Dir, 0, o.Sub)
	}
	if o.Agent != "" {
		var f []Session
		for _, s := range sessions {
			if s.Tool == o.Agent {
				f = append(f, s)
			}
		}
		sessions = f
	}

	out := make([]Match, len(sessions))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 24)
	for i, s := range sessions {
		wg.Add(1)
		go func(i int, s Session) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if snip, ok := searchOne(s, match, pos); ok {
				out[i] = Match{Session: s, Snippet: snip}
			}
		}(i, s)
	}
	wg.Wait()

	var matches []Match
	for _, m := range out {
		if m.Snippet != "" || m.Tool != "" {
			matches = append(matches, m)
		}
	}
	sort.SliceStable(matches, func(a, b int) bool { return matches[a].Mtime.After(matches[b].Mtime) })
	if o.Limit > 0 && len(matches) > o.Limit {
		matches = matches[:o.Limit]
	}
	return matches, nil
}

func searchOne(s Session, match func(string) bool, pos func(string) int) (string, bool) {
	// SQLite-backed stores aren't usefully grep-able; match the loaded title.
	if s.Tool == "opencode" || s.Tool == "agy" {
		if s.Preview != "" && match(s.Preview) {
			return clean(s.Preview), true
		}
		return "", false
	}
	f, err := os.Open(s.Path)
	if err != nil {
		if s.Preview != "" && match(s.Preview) {
			return clean(s.Preview), true
		}
		return "", false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if match(line) {
			return excerpt(line, pos(line)), true
		}
	}
	return "", false
}

// excerpt returns a cleaned ~150-char window of line centered on idx.
func excerpt(line string, idx int) string {
	if idx < 0 {
		return clip(clean(line), 150)
	}
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + 100
	if end > len(line) {
		end = len(line)
	}
	w := clean(line[start:end])
	if start > 0 {
		w = "…" + w
	}
	if end < len(line) {
		w += "…"
	}
	return w
}

var unescape = strings.NewReplacer(`\n`, " ", `\t`, " ", `\r`, " ", `\"`, `"`, `\\`, `\`)

// clean unescapes common JSON string escapes and flattens whitespace/control
// chars so a raw JSONL line reads as plain text in an excerpt.
func clean(s string) string {
	s = unescape.Replace(s)
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return ' '
		case r < 0x20:
			return -1
		default:
			return r
		}
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func clip(s string, n int) string {
	r := []rune(s)
	if n > 0 && len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

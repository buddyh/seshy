// Package render produces non-interactive output (table / json / summary).
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/buddyh/seshy/internal/agents"
)

// HumanAge renders a compact relative age.
func HumanAge(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%.1fy ago", d.Hours()/24/365)
	}
}

func label(tool string) lipgloss.Style {
	hex := "#FFFFFF"
	if m, ok := agents.Metas[tool]; ok {
		hex = m.Hex
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
}

var (
	dim = lipgloss.NewStyle().Faint(true)
	grn = lipgloss.NewStyle().Foreground(lipgloss.Color("#3FB950"))
	cyn = lipgloss.NewStyle().Foreground(lipgloss.Color("#39C5CF"))
)

// List prints sessions as a colored table for humans.
func List(w io.Writer, target string, ss []agents.Session, sub bool) {
	hdr := lipgloss.NewStyle().Bold(true).Render("seshy") + "  " + cyn.Render(target)
	if sub {
		hdr += dim.Render(" (+subdirs)")
	}
	fmt.Fprintln(w, "\n"+hdr)
	if len(ss) == 0 {
		fmt.Fprintln(w, "  "+lipgloss.NewStyle().Foreground(lipgloss.Color("#D29922")).Render("no agent sessions here")+"\n")
		return
	}
	top := ss[0]
	fmt.Fprintln(w, fmt.Sprintf("→ most recent: %s %s\n",
		label(top.Tool).Render(agents.Label(top.Tool)), grn.Render(HumanAge(top.Mtime))))
	for _, s := range ss {
		prev := s.Preview
		if prev == "" {
			prev = "—"
		}
		if len(prev) > 80 {
			prev = prev[:79] + "…"
		}
		badge := ""
		if s.Model != "" || s.Cost > 0 || s.Msgs > 0 {
			badge = dim.Render(" " + metaBadge(s))
		}
		fmt.Fprintf(w, "  %s  %s %s  %s%s\n",
			grn.Render(fmt.Sprintf("%9s", HumanAge(s.Mtime))),
			label(s.Tool).Render(fmt.Sprintf("%-8s", agents.Label(s.Tool))),
			dim.Render(short(s.ID)), prev, badge)
	}
	fmt.Fprintln(w)
}

// Recent prints the most-recent sessions across all repositories (the global
// view), showing the repo each session belongs to instead of one project path.
func Recent(w io.Writer, ss []agents.Session) {
	hdr := lipgloss.NewStyle().Bold(true).Render("seshy") + "  " + cyn.Render("most recent · all repositories")
	fmt.Fprintln(w, "\n"+hdr)
	if len(ss) == 0 {
		fmt.Fprintln(w, "  "+lipgloss.NewStyle().Foreground(lipgloss.Color("#D29922")).Render("no agent sessions found")+"\n")
		return
	}
	for _, s := range ss {
		prev := s.Preview
		if prev == "" {
			prev = "—"
		}
		fmt.Fprintf(w, "  %s  %s  %s  %s\n",
			grn.Render(fmt.Sprintf("%9s", HumanAge(s.Mtime))),
			label(s.Tool).Render(fmt.Sprintf("%-8s", agents.Label(s.Tool))),
			cyn.Render(fmt.Sprintf("%-18s", clip(filepath.Base(s.Dir), 18))),
			clip(prev, 70))
	}
	fmt.Fprintln(w)
}

// clip shortens s to at most n runes with an ellipsis.
func clip(s string, n int) string {
	r := []rune(s)
	if n > 0 && len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

type jsonMatch struct {
	jsonSession
	Snippet string `json:"snippet"`
}

// Matches writes search results: JSON/NDJSON for tooling (each with the matched
// excerpt and resume command), or a table with the excerpt and a copy-ready
// resume command per hit.
func Matches(w io.Writer, ms []agents.Match, format string) {
	switch format {
	case "json":
		out := struct {
			Count   int         `json:"count"`
			Matches []jsonMatch `json:"matches"`
		}{Count: len(ms)}
		for _, m := range ms {
			out.Matches = append(out.Matches, jsonMatch{toJSON(m.Session), m.Snippet})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	case "ndjson":
		enc := json.NewEncoder(w)
		for _, m := range ms {
			enc.Encode(jsonMatch{toJSON(m.Session), m.Snippet})
		}
	default:
		if len(ms) == 0 {
			fmt.Fprintln(w, "\n  "+lipgloss.NewStyle().Foreground(lipgloss.Color("#D29922")).Render("no matches")+"\n")
			return
		}
		fmt.Fprintln(w, "")
		for _, m := range ms {
			s := m.Session
			fmt.Fprintf(w, "  %s  %s  %s\n",
				grn.Render(fmt.Sprintf("%9s", HumanAge(s.Mtime))),
				label(s.Tool).Render(fmt.Sprintf("%-8s", agents.Label(s.Tool))),
				cyn.Render(filepath.Base(s.Dir)))
			fmt.Fprintln(w, "    "+m.Snippet)
			fmt.Fprintln(w, "    "+dim.Render(strings.Join(agents.ResumeArgv(s), " ")))
		}
		fmt.Fprintln(w, "")
	}
}

func metaBadge(s agents.Session) string {
	var p []string
	if s.Model != "" {
		p = append(p, s.Model)
	}
	if s.Cost > 0 {
		p = append(p, fmt.Sprintf("$%.2f", s.Cost))
	}
	if s.Msgs > 0 {
		p = append(p, fmt.Sprintf("%d msgs", s.Msgs))
	}
	if len(p) == 0 {
		return ""
	}
	return "· " + strings.Join(p, " · ")
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

type jsonOut struct {
	Directory  string        `json:"directory"`
	MostRecent *jsonSession  `json:"most_recent"`
	Sessions   []jsonSession `json:"sessions"`
	Version    string        `json:"format_version"`
}

type jsonSession struct {
	Agent   string  `json:"agent"`
	ID      string  `json:"id"`
	Path    string  `json:"path"`
	Dir     string  `json:"dir"`
	Mtime   string  `json:"mtime"`
	Preview string  `json:"preview,omitempty"`
	Model   string  `json:"model,omitempty"`
	Cost    float64 `json:"cost,omitempty"`
	Msgs    int     `json:"msgs,omitempty"`
	Resume  string  `json:"resume"`
}

func toJSON(s agents.Session) jsonSession {
	return jsonSession{
		Agent: s.Tool, ID: s.ID, Path: s.Path, Dir: s.Dir, Mtime: s.Mtime.UTC().Format(time.RFC3339),
		Preview: s.Preview, Model: s.Model, Cost: s.Cost, Msgs: s.Msgs,
		Resume: strings.Join(agents.ResumeArgv(s), " "),
	}
}

// JSON prints the full result as a single JSON object.
func JSON(w io.Writer, target string, ss []agents.Session) {
	out := jsonOut{Directory: target, Version: "1.0"}
	for _, s := range ss {
		js := toJSON(s)
		out.Sessions = append(out.Sessions, js)
	}
	if len(ss) > 0 {
		mr := toJSON(ss[0])
		out.MostRecent = &mr
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

// NDJSON prints one session per line.
func NDJSON(w io.Writer, ss []agents.Session) {
	enc := json.NewEncoder(w)
	for _, s := range ss {
		enc.Encode(toJSON(s))
	}
}

// Summary prints a compact agent-oriented digest.
func Summary(w io.Writer, target string, ss []agents.Session, asJSON bool) {
	byAgent := map[string]int{}
	for _, s := range ss {
		byAgent[s.Tool]++
	}
	if asJSON {
		type sum struct {
			Path       string         `json:"project_path"`
			Count      int            `json:"session_count"`
			ByAgent    map[string]int `json:"by_agent"`
			MostRecent string         `json:"most_recent_id,omitempty"`
			MostAgent  string         `json:"most_recent_agent,omitempty"`
			MostMtime  string         `json:"most_recent_mtime,omitempty"`
			Version    string         `json:"format_version"`
		}
		s := sum{Path: target, Count: len(ss), ByAgent: byAgent, Version: "1.0"}
		if len(ss) > 0 {
			s.MostRecent, s.MostAgent = ss[0].ID, ss[0].Tool
			s.MostMtime = ss[0].Mtime.UTC().Format(time.RFC3339)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(s)
		return
	}
	if len(ss) == 0 {
		fmt.Fprintf(w, "%s: no agent sessions\n", target)
		return
	}
	keys := make([]string, 0, len(byAgent))
	for k := range byAgent {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return byAgent[keys[a]] > byAgent[keys[b]] })
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s: %d", agents.Label(k), byAgent[k])
	}
	fmt.Fprintf(w, "%d sessions (%s) · most recent %s %s\n",
		len(ss), strings.Join(parts, ", "), agents.Label(ss[0].Tool), HumanAge(ss[0].Mtime))
}

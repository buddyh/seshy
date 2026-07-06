package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/buddyh/seshy/internal/agents"
	"github.com/buddyh/seshy/internal/retention"
)

var warn = lipgloss.NewStyle().Foreground(lipgloss.Color("#D29922"))

// humanBytes renders a byte count compactly (1024-based).
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	if v < 10 {
		return fmt.Sprintf("%.1f %s", v, suffix)
	}
	return fmt.Sprintf("%.0f %s", v, suffix)
}

// groupThousands renders an int with comma separators.
func groupThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// policyText renders one row's policy cell for the table.
func policyText(r retention.Row) (text string, atRisk bool) {
	p := r.Policy
	if p.Err != nil {
		return "config unreadable", true
	}
	switch {
	case p.Kind == retention.KeepForever:
		return "keeps sessions forever", false
	case p.Kind == retention.CloudManaged:
		return p.Effective, false
	case !p.AtRisk:
		return fmt.Sprintf("auto-delete off (%s)", p.Source), false
	default:
		return fmt.Sprintf("auto-deletes after %s (%s)", p.Effective, p.Source), true
	}
}

// homeTilde abbreviates the home prefix of path to ~.
func homeTilde(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// Retention writes the per-agent retention report as a table, JSON, or NDJSON.
func Retention(w io.Writer, rows []retention.Row, home, format string) {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(retentionJSON(rows))
	case "ndjson":
		enc := json.NewEncoder(w)
		for _, r := range rows {
			enc.Encode(toJSONRetention(r))
		}
	default:
		retentionTable(w, rows, home)
	}
}

func retentionTable(w io.Writer, rows []retention.Row, home string) {
	fmt.Fprintln(w, "\n"+lipgloss.NewStyle().Bold(true).Render("seshy")+"  "+cyn.Render("session retention"))
	if len(rows) == 0 {
		fmt.Fprintln(w, "  "+warn.Render("no agents to report")+"\n")
		return
	}

	storeW := len("STORE")
	for _, r := range rows {
		if n := len(homeTilde(r.Store, home)); n > storeW {
			storeW = n
		}
	}

	hdr := fmt.Sprintf("  %-9s %-*s %8s %9s %10s  %s", "AGENT", storeW, "STORE", "SIZE", "SESSIONS", "OLDEST", "POLICY")
	fmt.Fprintln(w, dim.Render(hdr))

	atRiskTotal := 0
	for _, r := range rows {
		policy, risk := policyText(r)
		size, sessions, oldest := "—", "—", "—"
		if r.Installed {
			size = humanBytes(r.Bytes)
			sessions = groupThousands(r.Sessions)
			if !r.Oldest.IsZero() {
				oldest = HumanAge(r.Oldest)
			}
		}
		agent := label(r.Key).Render(fmt.Sprintf("%-9s", agents.Label(r.Key)))
		store := dim.Render(fmt.Sprintf("%-*s", storeW, homeTilde(r.Store, home)))
		line := fmt.Sprintf("  %s %s %8s %9s %10s  ", agent, store, size, sessions, oldest)
		switch {
		case !r.Installed:
			line += dim.Render(policy + "  (not installed)")
		case risk:
			atRiskTotal++
			line += warn.Render("[!] " + policy)
		default:
			line += policy
		}
		fmt.Fprintln(w, line)
	}
	if atRiskTotal > 0 {
		fmt.Fprintf(w, "\n  %s\n", warn.Render(fmt.Sprintf(
			"[!] %d agent%s delete%s old sessions.", atRiskTotal, plural(atRiskTotal), singularVerb(atRiskTotal))))
		fmt.Fprintln(w, "      Keep a year of history: "+cyn.Render("seshy retention protect"))
	}
	fmt.Fprintln(w)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func singularVerb(n int) string {
	if n == 1 {
		return "s"
	}
	return ""
}

type jsonRetentionPolicy struct {
	Kind       string `json:"kind"`
	Default    string `json:"default"`
	Effective  string `json:"effective"`
	Source     string `json:"source"`
	AtRisk     bool   `json:"at_risk"`
	Note       string `json:"note,omitempty"`
	ConfigFile string `json:"config_file,omitempty"`
	Settable   bool   `json:"settable"`
	Error      string `json:"error,omitempty"`
}

type jsonRetentionAgent struct {
	Agent     string              `json:"agent"`
	Label     string              `json:"label"`
	Installed bool                `json:"installed"`
	Store     string              `json:"store"`
	DiskBytes int64               `json:"disk_bytes"`
	Sessions  int                 `json:"sessions"`
	Oldest    string              `json:"oldest,omitempty"`
	Policy    jsonRetentionPolicy `json:"policy"`
}

type jsonRetentionOut struct {
	Version string               `json:"format_version"`
	Agents  []jsonRetentionAgent `json:"agents"`
}

func toJSONRetention(r retention.Row) jsonRetentionAgent {
	a := jsonRetentionAgent{
		Agent: r.Key, Label: agents.Label(r.Key), Installed: r.Installed,
		Store: r.Store, DiskBytes: r.Bytes, Sessions: r.Sessions,
		Policy: jsonRetentionPolicy{
			Kind:       string(r.Policy.Kind),
			Default:    r.Policy.Default,
			Effective:  r.Policy.Effective,
			Source:     r.Policy.Source,
			AtRisk:     r.Policy.AtRisk,
			Note:       r.Policy.Note,
			ConfigFile: r.Config,
			Settable:   r.Settable,
		},
	}
	if !r.Oldest.IsZero() {
		a.Oldest = r.Oldest.UTC().Format(time.RFC3339)
	}
	if r.Policy.Err != nil {
		a.Policy.Error = r.Policy.Err.Error()
	}
	return a
}

func retentionJSON(rows []retention.Row) jsonRetentionOut {
	out := jsonRetentionOut{Version: "1.0"}
	for _, r := range rows {
		out.Agents = append(out.Agents, toJSONRetention(r))
	}
	return out
}

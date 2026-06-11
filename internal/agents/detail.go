package agents

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/buddyh/seshy/internal/store"
)

// Detail is the richer context shown in the preview pane.
type Detail struct {
	FirstPrompt string
	LastMessage string
	LastRole    string
	Turns       int    // user messages
	TaskMD      string // agy task.md (raw markdown), if present
	TaskDone    int
	TaskTotal   int
}

// extractMsg pulls (role, text) from a transcript record across agent shapes.
func extractMsg(d map[string]any) (role, text string) {
	p := d
	if pl, ok := d["payload"].(map[string]any); ok {
		p = pl
	}
	var content any
	if msg, ok := p["message"].(map[string]any); ok {
		if r, _ := msg["role"].(string); r != "" {
			role = r
		}
		content = msg["content"]
	}
	if role == "" {
		if r, _ := p["role"].(string); r != "" {
			role = r
		}
	}
	if role == "" {
		if t, _ := p["type"].(string); t == "user" || t == "assistant" {
			role = t
		}
	}
	if content == nil {
		content = p["content"]
	}
	switch c := content.(type) {
	case string:
		text = c
	case []any:
		var parts []string
		for _, b := range c {
			if blk, ok := b.(map[string]any); ok {
				t, _ := blk["type"].(string)
				if t == "text" || t == "input_text" || t == "output_text" {
					if s, _ := blk["text"].(string); s != "" {
						parts = append(parts, s)
					}
				}
			} else if s, ok := b.(string); ok {
				parts = append(parts, s)
			}
		}
		text = strings.Join(parts, " ")
	}
	return role, text
}

var noiseLast = []string{
	"<system-reminder", "<command-name>", "<command-message>", "<local-command",
	"<bash-", "Caveat:", "[Request interrupted", "<turn_aborted", "<environment",
	"<user_instructions", "# AGENTS.md",
}

func isNoise(raw string) bool {
	raw = strings.TrimLeft(raw, " \t")
	for _, n := range noiseLast {
		if strings.HasPrefix(raw, n) {
			return true
		}
	}
	return false
}

// TranscriptInfo scans a JSONL transcript for turn count and last real message.
func TranscriptInfo(path string) (turns int, lastRole, last string) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, `"role"`) && !strings.Contains(line, `"type":"user"`) &&
			!strings.Contains(line, `"type":"assistant"`) && !strings.Contains(line, `"type":"message"`) {
			continue
		}
		var d map[string]any
		if json.Unmarshal([]byte(line), &d) != nil {
			continue
		}
		role, text := extractMsg(d)
		if role != "user" && role != "assistant" {
			continue
		}
		if role == "user" {
			turns++
		}
		if t := cleanText(text); t != "" && !isNoise(text) {
			last, lastRole = t, role
		}
	}
	return turns, lastRole, last
}

var reCheckDone = regexp.MustCompile(`(?m)^\s*[-*]\s*\[[xX]\]`)
var reCheckTodo = regexp.MustCompile(`(?m)^\s*[-*]\s*\[ \]`)

// AgyTask reads an agy conversation's task.md (progress + raw markdown).
func AgyTask(id string) (md string, done, total int) {
	p := filepath.Join(store.Home, ".gemini", "antigravity-cli", "brain", id, "task.md")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", 0, 0
	}
	md = string(b)
	done = len(reCheckDone.FindAllString(md, -1))
	todo := len(reCheckTodo.FindAllString(md, -1))
	return md, done, done + todo
}

// LoadDetail assembles the preview detail for a session.
func LoadDetail(s Session) Detail {
	d := Detail{FirstPrompt: s.Preview}
	switch s.Tool {
	case "claude", "codex", "pi", "droid":
		turns, role, last := TranscriptInfo(s.Path)
		d.Turns, d.LastRole, d.LastMessage = turns, role, last
		if d.FirstPrompt == "" {
			d.FirstPrompt = FirstJSONLUser(s.Path)
		}
	case "agy":
		md, done, total := AgyTask(s.ID)
		d.TaskMD, d.TaskDone, d.TaskTotal = md, done, total
	}
	return d
}

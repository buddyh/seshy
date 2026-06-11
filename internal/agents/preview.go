package agents

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

var (
	reCmdBlock = regexp.MustCompile(`(?s)<command-(?:message|name|args)>.*?</command-(?:message|name|args)>`)
	reTag      = regexp.MustCompile(`<[^>]+>`)
	reWS       = regexp.MustCompile(`\s+`)
	reCmdName  = regexp.MustCompile(`<command-name>\s*(/?[^<\s]+)`)
	reCmdArgs  = regexp.MustCompile(`(?s)<command-args>\s*(.*?)\s*</command-args>`)
)

func cleanText(s string) string {
	s = reCmdBlock.ReplaceAllString(s, "")
	s = reTag.ReplaceAllString(s, "")
	return strings.TrimSpace(reWS.ReplaceAllString(s, " "))
}

func slashCommand(raw string) string {
	m := reCmdName.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	arg := ""
	if a := reCmdArgs.FindStringSubmatch(raw); a != nil {
		arg = reWS.ReplaceAllString(strings.TrimSpace(a[1]), " ")
	}
	return strings.TrimSpace(m[1] + " " + arg)
}

var previewSkip = []string{
	"Caveat:", "[Request interrupted", "<local-command", "<bash-", "<system-reminder",
	"# AGENTS.md", "<permissions", "<user_instructions", "<environment", "<turn_aborted",
	"The following is the Codex agent history",
}

// FirstJSONLUser returns the first genuine user message in a JSONL transcript
// (handles Claude / Codex / pi / Droid message shapes).
func FirstJSONLUser(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for i := 0; sc.Scan() && i < 120; i++ {
		line := sc.Text()
		if !strings.Contains(line, `"role":"user"`) && !strings.Contains(line, `"type":"user"`) {
			continue
		}
		var d map[string]any
		if json.Unmarshal([]byte(line), &d) != nil {
			continue
		}
		text := extractUserText(d)
		raw := strings.TrimLeft(text, " \t")
		if strings.HasPrefix(raw, "<command-name>") || strings.HasPrefix(raw, "<command-message>") {
			if sc := slashCommand(raw); sc != "" {
				return sc
			}
			continue
		}
		skip := false
		for _, p := range previewSkip {
			if strings.HasPrefix(raw, p) {
				skip = true
				break
			}
		}
		if skip || strings.Contains(raw[:min(80, len(raw))], "Global Codex Agent Guidelines") {
			continue
		}
		if t := cleanText(text); t != "" {
			return t
		}
	}
	return ""
}

// extractUserText pulls the first text blob from a transcript record's user message.
func extractUserText(d map[string]any) string {
	// payload wrapper (codex)
	p := d
	if pl, ok := d["payload"].(map[string]any); ok {
		p = pl
	}
	if role, ok := p["role"].(string); ok && role != "" && role != "user" {
		return ""
	}
	var content any
	if msg, ok := p["message"].(map[string]any); ok {
		content = msg["content"]
	}
	if content == nil {
		content = p["content"]
	}
	switch c := content.(type) {
	case string:
		return c
	case []any:
		for _, b := range c {
			switch blk := b.(type) {
			case map[string]any:
				if t, _ := blk["type"].(string); t == "text" || t == "input_text" {
					if s, _ := blk["text"].(string); s != "" {
						return s
					}
				}
			case string:
				return blk
			}
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

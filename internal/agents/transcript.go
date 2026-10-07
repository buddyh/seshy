package agents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// TranscriptMessage is a user or assistant text message with a source location.
// Tools, reasoning, system instructions, and injected setup records are omitted.
type TranscriptMessage struct {
	Role       string `json:"role"`
	Text       string `json:"text"`
	Timestamp  string `json:"timestamp,omitempty"`
	ByteOffset *int64 `json:"byte_offset,omitempty"`
	MessageID  string `json:"message_id,omitempty"`
	Truncated  bool   `json:"truncated"`
}

// TranscriptPage reports exactly which part of the source was readable.
type TranscriptPage struct {
	Messages         []TranscriptMessage `json:"messages"`
	NextOffset       *int64              `json:"next_offset"`
	SourceBytes      int64               `json:"source_bytes"`
	ScannedBytes     int64               `json:"scanned_bytes"`
	SkippedRecords   int                 `json:"skipped_records"`
	MalformedRecords int                 `json:"malformed_records"`
	Complete         bool                `json:"complete"`
	Mode             string              `json:"mode"`
	Warnings         []string            `json:"warnings"`
}

// TranscriptOptions bounds source scanning and returned text independently.
type TranscriptOptions struct {
	Offset       int64
	Limit        int
	MaxChars     int
	MaxBytes     int64
	Tail         bool
	LiteralQuery string
}

// ReadJSONLTranscript reads bounded conversation text from a selected transcript.
// Offset is a byte boundary returned by an earlier page, not a filesystem path.
func ReadJSONLTranscript(path string, o TranscriptOptions) (TranscriptPage, error) {
	p := TranscriptPage{Messages: []TranscriptMessage{}, Warnings: []string{}, Mode: "page"}
	if o.Limit < 1 || o.Limit > 100 || o.MaxChars < 256 || o.MaxChars > 64000 || o.MaxBytes < 1 || o.MaxBytes > 16<<20 || o.Offset < 0 {
		return p, fmt.Errorf("invalid transcript bounds")
	}
	f, err := os.Open(path)
	if err != nil {
		return p, fmt.Errorf("transcript unavailable: %w", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return p, err
	}
	if !st.Mode().IsRegular() {
		return p, fmt.Errorf("transcript must be a regular file")
	}
	p.SourceBytes = st.Size()
	start := o.Offset
	if o.Tail {
		start = max(0, st.Size()-o.MaxBytes)
		p.Mode = "recent"
	}
	if start > st.Size() {
		return p, fmt.Errorf("offset exceeds source size; refresh the session")
	}
	if start > 0 {
		var before [1]byte
		if _, err := f.ReadAt(before[:], start-1); err != nil {
			return p, err
		}
		if before[0] != '\n' && !o.Tail {
			return p, fmt.Errorf("offset must be a returned record boundary")
		}
	}
	scanBytes := st.Size() - start
	if o.MaxBytes < scanBytes {
		scanBytes = o.MaxBytes
	}
	r := bufio.NewReader(io.NewSectionReader(f, start, scanBytes))
	pos := start
	if o.Tail && start > 0 {
		var before [1]byte
		_, _ = f.ReadAt(before[:], start-1)
		if before[0] != '\n' {
			b, _ := r.ReadBytes('\n')
			pos += int64(len(b))
			p.ScannedBytes += int64(len(b))
		}
	}
	chars := 0
	for {
		at := pos
		b, readErr := r.ReadBytes('\n')
		pos += int64(len(b))
		p.ScannedBytes += int64(len(b))
		if len(b) == 0 {
			break
		}
		if readErr == io.EOF && pos < st.Size() && b[len(b)-1] != '\n' {
			if at == start {
				return p, fmt.Errorf("source record exceeds scan budget; cannot safely continue")
			}
			p.Warnings = appendOnce(p.Warnings, "Source scan budget ended inside a record; continue with next_offset.")
			p.NextOffset = &at
			break
		} else if len(b) > 1<<20 {
			p.MalformedRecords++
			p.Warnings = appendOnce(p.Warnings, "A source record exceeded the 1 MiB record limit.")
		} else {
			var d map[string]any
			if json.Unmarshal(b, &d) != nil {
				if strings.TrimSpace(string(b)) != "" {
					p.MalformedRecords++
				}
			} else if role, text := conversationText(d); text != "" {
				if o.LiteralQuery != "" {
					matched, found := LiteralExcerpt(text, o.LiteralQuery)
					if !found {
						p.SkippedRecords++
						if readErr == io.EOF {
							break
						}
						continue
					}
					text = matched
				}
				available := o.MaxChars - chars
				if o.Tail {
					available = min(o.MaxChars, 6000)
				}
				if available <= 0 {
					p.NextOffset = &at
					break
				}
				runes := []rune(text)
				m := TranscriptMessage{Role: role, Text: text, ByteOffset: &at, Truncated: len(runes) > available, Timestamp: messageTimestamp(d)}
				if m.Truncated {
					m.Text = string(runes[:available])
					p.Warnings = appendOnce(p.Warnings, "Message text was clipped to the output budget.")
				}
				p.Messages = append(p.Messages, m)
				chars += len([]rune(m.Text))
				if o.Tail {
					for len(p.Messages) > o.Limit || chars > o.MaxChars {
						chars -= len([]rune(p.Messages[0].Text))
						p.Messages = p.Messages[1:]
					}
				} else if len(p.Messages) >= o.Limit || chars >= o.MaxChars {
					if pos < st.Size() {
						p.NextOffset = &pos
					}
					break
				}
			} else {
				p.SkippedRecords++
				if d["type"] == "compacted" {
					p.Warnings = appendOnce(p.Warnings, "Source includes a compaction marker; earlier conversation coverage may be incomplete.")
				}
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return p, readErr
			}
			break
		}
	}
	if !o.Tail && p.NextOffset == nil && pos < st.Size() {
		p.NextOffset = &pos
	}
	p.Complete = start == 0 && pos >= st.Size() && p.MalformedRecords == 0 && len(p.Warnings) == 0
	if o.Tail && start > 0 {
		p.Warnings = appendOnce(p.Warnings, "Recent view omits earlier source records.")
	}
	if p.MalformedRecords > 0 {
		p.Warnings = appendOnce(p.Warnings, "Malformed or oversized source records were skipped.")
	}
	return p, nil
}

// LiteralExcerpt returns a case-insensitive match surrounded by source text.
// Rune positions preserve valid UTF-8 even when lowercasing changes byte lengths.
func LiteralExcerpt(text, query string) (string, bool) {
	if query == "" {
		return "", false
	}
	lower := strings.ToLower(text)
	at := strings.Index(lower, strings.ToLower(query))
	if at < 0 {
		return "", false
	}
	position := utf8.RuneCountInString(lower[:at])
	runes := []rune(text)
	return string(runes[max(0, position-100):min(len(runes), position+utf8.RuneCountInString(query)+300)]), true
}

func conversationText(d map[string]any) (string, string) {
	// Codex's event stream duplicates response_item messages. Only use the
	// canonical message records, not event_msg/user_message/agent_message.
	if t, _ := d["type"].(string); t == "event_msg" || t == "compacted" {
		return "", ""
	}
	if d["channel"] == "analysis" {
		return "", ""
	}
	if p, ok := d["payload"].(map[string]any); ok {
		if p["channel"] == "analysis" {
			return "", ""
		}
	}
	role, text := extractMsg(d)
	if role != "user" && role != "assistant" {
		return "", ""
	}
	text = strings.TrimSpace(text)
	if text == "" || isNoise(text) || strings.Contains(text[:min(160, len(text))], "Global Codex Agent Guidelines") {
		return "", ""
	}
	for _, prefix := range previewSkip {
		if strings.HasPrefix(text, prefix) {
			return "", ""
		}
	}
	return role, text
}

func messageTimestamp(d map[string]any) string {
	if s, _ := d["timestamp"].(string); s != "" {
		return s
	}
	if m, ok := d["message"].(map[string]any); ok {
		if s, _ := m["timestamp"].(string); s != "" {
			return s
		}
	}
	return ""
}

func appendOnce(ss []string, s string) []string {
	for _, v := range ss {
		if v == s {
			return ss
		}
	}
	return append(ss, s)
}

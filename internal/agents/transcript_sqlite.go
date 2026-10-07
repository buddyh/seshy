package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/buddyh/seshy/internal/store"
)

// ReadOpenCodeTranscript reads only text parts belonging to a selected session.
// Its offset is a message-row index; each result cites the stable message ID.
func ReadOpenCodeTranscript(s Session, o TranscriptOptions) (TranscriptPage, error) {
	p := TranscriptPage{Messages: []TranscriptMessage{}, Warnings: []string{}, Mode: "database_messages"}
	if o.Offset < 0 || o.Limit < 1 || o.Limit > 100 || o.MaxChars < 256 || o.MaxChars > 64000 || o.MaxBytes < 1 || o.MaxBytes > 16<<20 {
		return p, fmt.Errorf("invalid transcript bounds")
	}
	db, err := store.OpenRO(s.Path)
	if err != nil {
		return p, fmt.Errorf("OpenCode store unavailable: %w", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var total int64
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM message WHERE session_id=?", s.ID).Scan(&total); err != nil {
		return p, fmt.Errorf("unsupported OpenCode message schema: %w", err)
	}
	start := o.Offset
	if o.Tail {
		start = max(0, total-int64(o.Limit))
		p.Mode = "recent_database_messages"
	}
	if start > total {
		return p, fmt.Errorf("offset exceeds message count; refresh the session")
	}
	rows, err := db.QueryContext(ctx, "SELECT id,CAST(substr(CAST(data AS BLOB),1,1048577) AS TEXT),time_created FROM message WHERE session_id=? ORDER BY time_created,id LIMIT ? OFFSET ?", s.ID, o.Limit, start)
	if err != nil {
		return p, err
	}
	type record struct {
		id, data string
		ts       int64
	}
	var records []record
	metadataBytes := int64(0)
	for rows.Next() {
		var r record
		if err = rows.Scan(&r.id, &r.data, &r.ts); err != nil {
			rows.Close()
			return p, err
		}
		records = append(records, r)
		metadataBytes += int64(len(r.data))
		if metadataBytes >= o.MaxBytes {
			break
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return p, err
	}
	chars := 0
	processed := int64(0)
	for _, r := range records {
		if p.ScannedBytes >= o.MaxBytes {
			break
		}
		p.ScannedBytes += int64(len(r.data))
		if len(r.data) > 1<<20 {
			p.MalformedRecords++
			processed++
			continue
		}
		var meta map[string]any
		if json.Unmarshal([]byte(r.data), &meta) != nil {
			p.MalformedRecords++
			processed++
			continue
		}
		role, _ := meta["role"].(string)
		if role != "user" && role != "assistant" {
			p.SkippedRecords++
			processed++
			continue
		}
		parts, err := db.QueryContext(ctx, "SELECT CAST(substr(CAST(data AS BLOB),1,1048577) AS TEXT) FROM part WHERE session_id=? AND message_id=? ORDER BY time_created,id LIMIT 257", s.ID, r.id)
		if err != nil {
			return p, fmt.Errorf("unsupported OpenCode text-part schema: %w", err)
		}
		var texts []string
		partCount := 0
		for parts.Next() {
			var raw string
			if err = parts.Scan(&raw); err != nil {
				parts.Close()
				return p, err
			}
			p.ScannedBytes += int64(len(raw))
			partCount++
			if partCount > 256 || len(raw) > 1<<20 || p.ScannedBytes > o.MaxBytes {
				p.Warnings = appendOnce(p.Warnings, "OpenCode text-part scan budget exceeded; this message is incomplete.")
				break
			}
			var v map[string]any
			if json.Unmarshal([]byte(raw), &v) != nil {
				p.MalformedRecords++
				continue
			}
			if v["type"] == "text" {
				if t, ok := v["text"].(string); ok {
					texts = append(texts, t)
				}
			}
		}
		err = parts.Err()
		parts.Close()
		if err != nil {
			return p, err
		}
		text := strings.TrimSpace(strings.Join(texts, "\n"))
		if text == "" || isNoise(text) {
			p.SkippedRecords++
			processed++
			continue
		}
		if o.LiteralQuery != "" {
			matched, found := LiteralExcerpt(text, o.LiteralQuery)
			if !found {
				processed++
				continue
			}
			text = matched
		}
		remaining := o.MaxChars - chars
		if remaining <= 0 {
			break
		}
		runes := []rune(text)
		clipped := len(runes) > remaining
		if clipped {
			text = string(runes[:remaining])
			p.Warnings = appendOnce(p.Warnings, "Message text was clipped to the output budget.")
		}
		p.Messages = append(p.Messages, TranscriptMessage{Role: role, Text: text, MessageID: r.id, Timestamp: time.UnixMilli(r.ts).UTC().Format(time.RFC3339), Truncated: clipped})
		chars += len([]rune(text))
		processed++
	}
	if next := start + processed; next < total {
		p.NextOffset = &next
	}
	p.Complete = start == 0 && start+processed >= total && p.MalformedRecords == 0 && len(p.Warnings) == 0
	if o.Tail && start > 0 {
		p.Warnings = appendOnce(p.Warnings, "Recent view omits earlier messages.")
	}
	if p.MalformedRecords > 0 {
		p.Warnings = appendOnce(p.Warnings, "Malformed source records were skipped.")
	}
	return p, nil
}

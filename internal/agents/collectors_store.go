package agents

import (
	"bufio"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buddyh/seshy/internal/store"
)

func collectGrok(target string, sub, global bool) []Session {
	root := filepath.Join(store.Home, ".grok", "sessions")
	target = store.TrimSlash(target)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Session
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "%") {
			continue
		}
		cwd, err := url.PathUnescape(e.Name())
		if err != nil {
			continue
		}
		cwd = store.TrimSlash(cwd)
		if !global && cwd != target && !(sub && strings.HasPrefix(cwd, target+"/")) {
			continue
		}
		sessDir := filepath.Join(root, e.Name())
		sessions, _ := os.ReadDir(sessDir)
		for _, s := range sessions {
			summ := filepath.Join(sessDir, s.Name(), "summary.json")
			fi, err := os.Stat(summ)
			if err != nil {
				continue
			}
			var meta struct {
				Info struct {
					ID string `json:"id"`
				} `json:"info"`
				SessionSummary string `json:"session_summary"`
				GeneratedTitle string `json:"generated_title"`
				Model          string `json:"current_model_id"`
				Msgs           int    `json:"num_chat_messages"`
			}
			if b, err := os.ReadFile(summ); err == nil {
				json.Unmarshal(b, &meta)
			}
			id := meta.Info.ID
			if id == "" {
				id = s.Name()
			}
			prev := meta.SessionSummary
			if prev == "" {
				prev = meta.GeneratedTitle
			}
			out = append(out, Session{
				Tool: "grok", ID: id, Path: summ, Dir: cwd, Mtime: fi.ModTime(),
				Preview: prev, preFilled: true, Model: meta.Model, Msgs: meta.Msgs,
			})
		}
	}
	return out
}

func collectOpenCode(target string, sub, global bool) []Session {
	dbPath := filepath.Join(store.Home, ".local", "share", "opencode", "opencode.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	target = store.TrimSlash(target)
	db, err := store.OpenRO(dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()
	q := "SELECT id,directory,title,time_updated,model,cost FROM session"
	var args []any
	switch {
	case global:
		// all directories — no filter
	case sub:
		q += " WHERE directory=? OR directory LIKE ?"
		args = []any{target, target + "/%"}
	default:
		q += " WHERE directory=?"
		args = []any{target}
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var id, dir, title, model string
		var tup int64
		var cost float64
		if rows.Scan(&id, &dir, &title, &tup, &model, &cost) != nil {
			continue
		}
		out = append(out, Session{
			Tool: "opencode", ID: id, Path: dbPath, Dir: dir,
			Mtime: time.UnixMilli(tup), Preview: title, preFilled: true,
			Model: model, Cost: cost,
		})
	}
	return out
}

func collectAgy(target string, sub, global bool) []Session {
	C := filepath.Join(store.Home, ".gemini", "antigravity-cli")
	lc := filepath.Join(C, "cache", "last_conversations.json")
	b, err := os.ReadFile(lc)
	if err != nil {
		return nil
	}
	var mapping map[string]string
	if json.Unmarshal(b, &mapping) != nil {
		return nil
	}
	target = store.TrimSlash(target)
	hist := agyHistory(C)
	var out []Session
	for cwd, cid := range mapping {
		cwd = store.TrimSlash(cwd)
		if !global && cwd != target && !(sub && strings.HasPrefix(cwd, target+"/")) {
			continue
		}
		dbp := filepath.Join(C, "conversations", cid+".db")
		mtime := time.Now()
		if fi, err := os.Stat(dbp); err == nil {
			mtime = fi.ModTime()
		} else if h, ok := hist[cwd]; ok {
			mtime = time.UnixMilli(h.ts)
		}
		out = append(out, Session{
			Tool: "agy", ID: cid, Path: dbp, Dir: cwd, Mtime: mtime,
			Preview: hist[cwd].display, preFilled: true,
		})
	}
	return out
}

type agyEntry struct {
	ts      int64
	display string
}

func agyHistory(C string) map[string]agyEntry {
	out := map[string]agyEntry{}
	f, err := os.Open(filepath.Join(C, "history.jsonl"))
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var e struct {
			Display   string `json:"display"`
			Timestamp int64  `json:"timestamp"`
			Workspace string `json:"workspace"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		ws := store.TrimSlash(e.Workspace)
		if ws == "" || e.Display == "" {
			continue
		}
		if cur, ok := out[ws]; !ok || e.Timestamp >= cur.ts {
			out[ws] = agyEntry{ts: e.Timestamp, display: e.Display}
		}
	}
	return out
}

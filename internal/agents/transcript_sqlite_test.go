package agents

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeSelectedSessionReadOnlyAndMalformedParts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"CREATE TABLE message (id TEXT,session_id TEXT,data TEXT,time_created INTEGER)",
		"CREATE TABLE part (id TEXT,session_id TEXT,message_id TEXT,data TEXT,time_created INTEGER)",
		`INSERT INTO message VALUES ('m1','selected','{"role":"user"}',1000),('m2','other','{"role":"user"}',2000)`,
		`INSERT INTO part VALUES ('p1','selected','m1','{"type":"text","text":"Chosen conversation"}',1000),('p2','other','m2','{"type":"text","text":"Other conversation"}',2000),('p3','selected','m1','broken',3000)`,
	} {
		if _, err = db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	before, _ := os.ReadFile(path)
	p, err := ReadOpenCodeTranscript(Session{Path: path, ID: "selected"}, bounds())
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("read mutated database")
	}
	if len(p.Messages) != 1 || p.Messages[0].Text != "Chosen conversation" || p.Messages[0].MessageID != "m1" || p.Complete || p.MalformedRecords != 1 {
		t.Fatalf("selected scope/coverage: %+v", p)
	}
}

func TestOpenCodeRecentKeepsNewestMessagesWithinBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("a", 6000)
	for _, q := range []string{
		"CREATE TABLE message (id TEXT,session_id TEXT,data TEXT,time_created INTEGER)",
		"CREATE TABLE part (id TEXT,session_id TEXT,message_id TEXT,data TEXT,time_created INTEGER)",
		`INSERT INTO message VALUES ('m1','s','{"role":"user"}',1000),('m2','s','{"role":"assistant"}',2000),('m3','s','{"role":"assistant"}',3000)`,
		`INSERT INTO part VALUES ('p1','s','m1','{"type":"text","text":"` + long + `"}',1000),('p2','s','m2','{"type":"text","text":"` + long + `"}',2000),('p3','s','m3','{"type":"text","text":"Final decision"}',3000)`,
	} {
		if _, err = db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	o := bounds()
	o.Tail = true
	o.MaxChars = 6500
	p, err := ReadOpenCodeTranscript(Session{Path: path, ID: "s"}, o)
	if err != nil {
		t.Fatal(err)
	}
	n := len(p.Messages)
	if n == 0 || p.Messages[n-1].MessageID != "m3" || p.Messages[n-1].Text != "Final decision" {
		t.Fatalf("newest message dropped: %+v", p.Messages)
	}
	total := 0
	for _, m := range p.Messages {
		total += len([]rune(m.Text))
	}
	if total > o.MaxChars {
		t.Fatalf("budget exceeded: %d", total)
	}
	if p.Messages[0].MessageID != "m2" {
		t.Fatalf("expected oldest messages dropped first: %+v", p.Messages)
	}
}

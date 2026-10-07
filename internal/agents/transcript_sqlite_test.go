package agents

import (
	"database/sql"
	"os"
	"path/filepath"
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

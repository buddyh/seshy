package retention

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMutateJSONPreservesSiblings(t *testing.T) {
	home := t.TempDir()
	path := writeFile(t, home, "settings.json",
		`{"model": "opus", "port": 8080, "nested": {"keep": [1, 2]}, "cleanupPeriodDays": 30}`, tMid)

	err := mutateJSON(path, false, func(m map[string]any) error {
		m["cleanupPeriodDays"] = 365
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{`"model": "opus"`, `"port": 8080`, `"keep"`, `"cleanupPeriodDays": 365`} {
		if !strings.Contains(got, want) {
			t.Errorf("rewritten file missing %s:\n%s", want, got)
		}
	}
	// UseNumber round-trip: 8080 must not become 8.08e+03.
	if strings.Contains(got, "e+") {
		t.Errorf("number mangled in round-trip:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("missing trailing newline")
	}
}

func TestMutateJSONBackup(t *testing.T) {
	home := t.TempDir()
	original := `{"cleanupPeriodDays": 30}`
	path := writeFile(t, home, "settings.json", original, tMid)

	if err := mutateJSON(path, false, func(m map[string]any) error {
		m["cleanupPeriodDays"] = 365
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("no backup written: %v", err)
	}
	if string(bak) != original {
		t.Errorf("backup = %q, want the pre-write bytes %q", bak, original)
	}
}

func TestMutateJSONMalformedAborts(t *testing.T) {
	home := t.TempDir()
	broken := `{not json`
	path := writeFile(t, home, "settings.json", broken, tMid)

	err := mutateJSON(path, false, func(m map[string]any) error { return nil })
	if err == nil {
		t.Fatal("want an error on malformed JSON")
	}
	got, _ := os.ReadFile(path)
	if string(got) != broken {
		t.Errorf("malformed file was modified: %q", got)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Error("no backup should be written when aborting")
	}
}

func TestMutateJSONCreate(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "settings.json")

	if err := mutateJSON(path, true, func(m map[string]any) error {
		m["cleanupPeriodDays"] = 365
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("new settings file mode = %o, want 600", fi.Mode().Perm())
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Error("no backup should exist for a freshly created file")
	}
}

func TestMutateJSONMissingNoCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")
	if err := mutateJSON(path, false, func(m map[string]any) error { return nil }); err == nil {
		t.Fatal("want an error when the file is missing and create is false")
	}
}

func TestMutateJSONThroughSymlink(t *testing.T) {
	home := t.TempDir()
	target := writeFile(t, home, "real/settings.json", `{"cleanupPeriodDays": 30}`, tMid)
	link := filepath.Join(home, "settings.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := mutateJSON(link, false, func(m map[string]any) error {
		m["cleanupPeriodDays"] = 365
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The link must survive and the target must carry the change.
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced by a regular file")
	}
	out, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "365") {
		t.Errorf("target not rewritten through the link:\n%s", out)
	}
}

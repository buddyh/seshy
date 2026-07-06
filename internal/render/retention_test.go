package render_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buddyh/seshy/internal/render"
	"github.com/buddyh/seshy/internal/retention"
	"github.com/buddyh/seshy/internal/retention/retentiontest"
)

// goldenPath is the committed JSON contract for `seshy retention -o json`.
// Regenerate with: UPDATE_GOLDEN=1 go test ./internal/render/
const goldenPath = "testdata/retention-golden.json"

// renderJSON runs the full pipeline (fixture home -> Report -> JSON) and
// normalizes machine-dependent bits: the temp home becomes "~", and the
// opencode disk_bytes (a sqlite page-allocation detail of the driver) is
// pinned to a sentinel after asserting it is real. Everything else must be
// byte-stable.
func renderJSON(t *testing.T, home string) []byte {
	t.Helper()
	rows := retention.Report(home)
	var buf bytes.Buffer
	render.Retention(&buf, rows, home, "json")

	var out struct {
		Version string           `json:"format_version"`
		Agents  []map[string]any `json:"agents"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	for _, a := range out.Agents {
		if a["agent"] == "opencode" {
			n, _ := a["disk_bytes"].(float64)
			if n <= 0 {
				t.Fatalf("opencode disk_bytes = %v, want > 0", a["disk_bytes"])
			}
			a["disk_bytes"] = 4096 // sentinel: sqlite file size is a driver detail
		}
	}
	norm, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(bytes.ReplaceAll(norm, []byte(home), []byte("~")), '\n')
}

func TestRetentionJSONGolden(t *testing.T) {
	home := retentiontest.FixtureHome(t)
	got := renderJSON(t, home)

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_GOLDEN=1): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("retention JSON drifted from %s\n--- got ---\n%s", goldenPath, got)
	}
}

func TestRetentionJSONDeterministic(t *testing.T) {
	home := retentiontest.FixtureHome(t)
	if a, b := renderJSON(t, home), renderJSON(t, home); !bytes.Equal(a, b) {
		t.Error("two identical runs produced different JSON")
	}
}

func TestRetentionTableSmoke(t *testing.T) {
	home := retentiontest.FixtureHome(t)
	var buf bytes.Buffer
	render.Retention(&buf, retention.Report(home), home, "table")
	out := buf.String()
	for _, want := range []string{
		"session retention",
		"auto-deletes after 30d (default)",
		"keeps sessions forever",
		"local: forever",
		"seshy retention protect",
		"~/.claude/projects", // home abbreviated
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, home) {
		t.Errorf("table output leaks the raw home path\n%s", out)
	}
}

func TestRetentionTableNotInstalled(t *testing.T) {
	retentiontest.NeutralEnv(t)
	home := t.TempDir()
	var buf bytes.Buffer
	render.Retention(&buf, retention.Report(home), home, "table")
	if !strings.Contains(buf.String(), "(not installed)") {
		t.Errorf("empty home should mark rows not installed\n%s", buf.String())
	}
}

func TestRetentionNDJSON(t *testing.T) {
	home := retentiontest.FixtureHome(t)
	var buf bytes.Buffer
	render.Retention(&buf, retention.Report(home), home, "ndjson")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(retention.Sources) {
		t.Fatalf("ndjson lines = %d, want %d", len(lines), len(retention.Sources))
	}
	for _, line := range lines {
		var a map[string]any
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			t.Errorf("bad ndjson line %q: %v", line, err)
		}
	}
}

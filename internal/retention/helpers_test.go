package retention

import (
	"testing"
	"time"

	"github.com/buddyh/seshy/internal/retention/retentiontest"
)

// Thin delegates so test bodies stay terse; the fixture home itself lives in
// retentiontest, shared with render's golden test.
var tMid = retentiontest.TMid

func writeAt(t *testing.T, home, rel string, n int, mtime time.Time) string {
	t.Helper()
	return retentiontest.WriteAt(t, home, rel, n, mtime)
}

func writeFile(t *testing.T, home, rel, content string, mtime time.Time) string {
	t.Helper()
	return retentiontest.WriteFile(t, home, rel, content, mtime)
}

func neutralEnv(t *testing.T) {
	t.Helper()
	retentiontest.NeutralEnv(t)
}

func newFixtureHome(t *testing.T) string {
	t.Helper()
	return retentiontest.FixtureHome(t)
}

// rowFor pulls one agent's row out of a Report result.
func rowFor(t *testing.T, rows []Row, key string) Row {
	t.Helper()
	for _, r := range rows {
		if r.Key == key {
			return r
		}
	}
	t.Fatalf("no row for %q", key)
	return Row{}
}

package retention

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// readJSONFile decodes a JSON object with UseNumber so integers survive a
// later round-trip unmangled. A missing file returns the os.ReadFile error
// (check os.IsNotExist); malformed JSON returns a descriptive error.
func readJSONFile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return m, nil
}

// mutateJSON rewrites a JSON settings file in place: decode with UseNumber
// (integers survive the round-trip), apply mut, copy the original bytes to
// path+".bak", then atomically rename a re-marshaled temp file over the
// target. Unknown sibling keys ride along untouched; key order and exotic
// formatting are not preserved (encoding/json sorts keys).
//
// Safety: a malformed file aborts before any byte is written; a missing file
// starts from {} only when create is true (written 0600 — agent settings can
// hold sensitive values); a symlinked path is resolved first so the target
// is rewritten instead of the link being replaced.
func mutateJSON(path string, create bool, mut func(m map[string]any) error) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	var m map[string]any
	var mode os.FileMode = 0o600
	orig, readErr := os.ReadFile(path)
	existed := false
	switch {
	case readErr == nil:
		existed = true
		if fi, err := os.Stat(path); err == nil {
			mode = fi.Mode().Perm()
		}
		var err error
		if m, err = readJSONFile(path); err != nil {
			return fmt.Errorf("refusing to modify: %w", err)
		}
	case os.IsNotExist(readErr) && create:
		m = map[string]any{}
	default:
		return readErr
	}
	if err := mut(m); err != nil {
		return err
	}
	if existed {
		if err := os.WriteFile(path+".bak", orig, mode); err != nil {
			return fmt.Errorf("backup %s.bak: %w", path, err)
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, append(out, '\n'), mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

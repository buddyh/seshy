package retention

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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

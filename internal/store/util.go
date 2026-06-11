// Package store holds shared helpers for reading agent session stores.
package store

import (
	"bufio"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // pure-Go sqlite driver
)

// Home returns the user's home directory (cached).
var Home = func() string {
	h, _ := os.UserHomeDir()
	return h
}()

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// EncodeClaude maps an absolute path to Claude's project-folder name.
func EncodeClaude(abspath string) string {
	return nonAlnum.ReplaceAllString(abspath, "-")
}

// FirstLine returns the first line of a file.
func FirstLine(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\n"), nil
}

// Glob is a thin filepath.Glob wrapper that swallows the (always-nil) bad-pattern error.
func Glob(pattern string) []string {
	m, _ := filepath.Glob(pattern)
	return m
}

// ScanResult pairs a path with its parsed-out values from a concurrent scan.
type ScanResult struct {
	Path  string
	First string
}

// ScanFirstLines reads the first line of every path concurrently (bounded).
func ScanFirstLines(paths []string, limit int) []ScanResult {
	if limit <= 0 {
		limit = 32
	}
	sem := make(chan struct{}, limit)
	out := make([]ScanResult, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			first, _ := FirstLine(p)
			out[i] = ScanResult{Path: p, First: first}
		}(i, p)
	}
	wg.Wait()
	return out
}

// OpenRO opens a sqlite DB read-only (WAL-safe).
func OpenRO(path string) (*sql.DB, error) {
	return sql.Open("sqlite", "file:"+path+"?mode=ro")
}

// TrimSlash trims a single trailing slash.
func TrimSlash(s string) string { return strings.TrimRight(s, "/") }

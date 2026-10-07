package plugin

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPDiscoveryToolErrorsAndUIResource(t *testing.T) {
	s, _ := fixture(t, line("recover a goal"))
	server := NewServer(s)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"fixture","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_session","arguments":{"session_key":"codex:a"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"doctor","arguments":{"path":"/etc/passwd"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"ui://seshy/workspace.html"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"unknown"}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := server.Serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("notification produced a response: %d", len(lines))
	}
	var response map[string]any
	json.Unmarshal([]byte(lines[1]), &response)
	tools := response["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 6 {
		t.Fatal("missing tools")
	}
	for _, tool := range tools {
		if tool.(map[string]any)["annotations"].(map[string]any)["readOnlyHint"] != true {
			t.Fatal("missing read-only annotation")
		}
	}
	json.Unmarshal([]byte(lines[2]), &response)
	result := response["result"].(map[string]any)
	if result["isError"] != false || result["structuredContent"].(map[string]any)["session"].(map[string]any)["session_key"] != "codex:a" {
		t.Fatal("read contract failed")
	}
	json.Unmarshal([]byte(lines[3]), &response)
	if response["result"].(map[string]any)["isError"] != true {
		t.Fatal("unknown fields were accepted")
	}
	json.Unmarshal([]byte(lines[4]), &response)
	contents := response["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)
	html := contents["text"].(string)
	if contents["mimeType"] != appMIME || !strings.Contains(html, "class HostBridge") || strings.Contains(html, "/* SESHY_") {
		t.Fatal("app not self-contained")
	}
	json.Unmarshal([]byte(lines[5]), &response)
	if response["error"].(map[string]any)["code"] != float64(-32601) {
		t.Fatal("unknown method contract")
	}
}

func TestMCPInitializationParseErrorsAndFrameBound(t *testing.T) {
	s, _ := fixture(t)
	server := NewServer(s)
	var output bytes.Buffer
	if err := server.Serve(strings.NewReader("broken\n"+`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatal("error responses missing")
	}
	var r map[string]any
	json.Unmarshal([]byte(lines[0]), &r)
	if r["id"] != nil || r["error"].(map[string]any)["code"] != float64(-32700) {
		t.Fatal("bad parse error")
	}
	json.Unmarshal([]byte(lines[1]), &r)
	if r["error"].(map[string]any)["code"] != float64(-32000) {
		t.Fatal("uninitialized calls accepted")
	}
	if err := server.Serve(strings.NewReader(strings.Repeat("x", 2<<20)), &output); err == nil {
		t.Fatal("oversized frame accepted")
	}
}

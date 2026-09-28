package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPServerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(origWd)
	}()
	_ = os.Chdir(tmpDir)

	// 1. Send initialize
	// 2. Send tools/list
	// 3. Send tools/call (generate_key_pair)
	// 4. Send tools/call (record_agent_action)

	req1 := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	req2 := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	req3 := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"generate_key_pair","arguments":{"key_name":"mcp-agent"}}}` + "\n"
	req4 := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"record_agent_action","arguments":{"agent":"cline","intent":"refactor","action":"git.commit","output_file":"receipt.json","private_key_path":"mcp-agent.key"}}}` + "\n"
	req5 := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"verify_receipt","arguments":{"receipt_path":"receipt.json"}}}` + "\n"

	inputData := req1 + req2 + req3 + req4 + req5
	inBuf := bytes.NewBufferString(inputData)
	var outBuf bytes.Buffer

	srv := NewServer(inBuf, &outBuf)
	if err := srv.Run(); err != nil {
		t.Fatalf("srv.Run() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 JSON-RPC responses, got %d:\n%s", len(lines), outBuf.String())
	}

	// Verify initialize response
	var r1 JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &r1); err != nil || r1.Error != nil {
		t.Fatalf("initialize failed: %+v", r1)
	}

	// Verify tools/list response
	var r2 JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &r2); err != nil || r2.Error != nil {
		t.Fatalf("tools/list failed: %+v", r2)
	}

	// Verify verify_receipt response
	var r5 JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[4]), &r5); err != nil || r5.Error != nil {
		t.Fatalf("verify_receipt tool call failed: %+v", r5)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "receipt.json")); os.IsNotExist(err) {
		t.Fatal("receipt.json was not created")
	}
}

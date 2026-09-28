package receipt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFileAndBytes(t *testing.T) {
	data := []byte("agent proof content")
	hash := HashBytes(data)
	if hash == "" {
		t.Fatal("expected non-empty hash from HashBytes")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	ev, err := HashFile(filePath)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	if ev.Hash != hash {
		t.Fatalf("HashFile hash %s != HashBytes hash %s", ev.Hash, hash)
	}
	if ev.URI != filePath {
		t.Fatalf("HashFile URI %s != expected %s", ev.URI, filePath)
	}
}

func TestHashDir(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "file1.txt")
	f2 := filepath.Join(tmpDir, "file2.txt")

	_ = os.WriteFile(f1, []byte("file1"), 0644)
	_ = os.WriteFile(f2, []byte("file2"), 0644)

	evidenceMap, err := HashDir(tmpDir)
	if err != nil {
		t.Fatalf("HashDir() error = %v", err)
	}

	if len(evidenceMap) != 2 {
		t.Fatalf("expected 2 items, got %d", len(evidenceMap))
	}
}

func TestUnmarshalAndString(t *testing.T) {
	jsonBlob := []byte(`{
		"version": "0.1",
		"agent": "test-agent",
		"intent": "unit-test",
		"action": "test",
		"action_timestamp": "2026-09-28T22:00:00Z",
		"signature": "c2ln",
		"public_key": "cHVi"
	}`)

	sr, err := UnmarshalReceipt(jsonBlob)
	if err != nil {
		t.Fatalf("UnmarshalReceipt() error = %v", err)
	}

	if sr.Agent != "test-agent" || sr.Signature != "c2ln" {
		t.Fatalf("unexpected parsed values: %+v", sr)
	}

	s := sr.String()
	if s == "" {
		t.Fatal("expected non-empty string representation")
	}
}

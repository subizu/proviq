package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordAndVerifyE2E(t *testing.T) {
	tmpDir := t.TempDir()
	receiptPath := filepath.Join(tmpDir, "test_receipt.json")
	inputFile := filepath.Join(tmpDir, "sample_code.go")
	policyFile := filepath.Join(tmpDir, "security_policy.json")

	if err := os.WriteFile(inputFile, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile(inputFile) error = %v", err)
	}
	if err := os.WriteFile(policyFile, []byte(`{"allow_commit": true}`), 0644); err != nil {
		t.Fatalf("WriteFile(policyFile) error = %v", err)
	}

	// Change working directory to tmpDir for keys/relative paths
	origWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(origWd)
	}()
	_ = os.Chdir(tmpDir)

	// Execute Record command via RootCmd
	RootCmd.SetArgs([]string{
		"record",
		"--agent", "claude-code-test",
		"--intent", "implement-feature",
		"--action", "git.commit",
		"--input-file", inputFile,
		"--policy-file", policyFile,
		"--generate-key",
		"--output", receiptPath,
	})

	var recordBuf bytes.Buffer
	RootCmd.SetOut(&recordBuf)
	RootCmd.SetErr(&recordBuf)

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("RootCmd record error = %v\nOutput: %s", err, recordBuf.String())
	}

	if _, err := os.Stat(receiptPath); os.IsNotExist(err) {
		t.Fatalf("expected receipt file at %s, but does not exist", receiptPath)
	}

	// Execute Verify command via RootCmd
	RootCmd.SetArgs([]string{
		"verify",
		receiptPath,
	})

	var verifyBuf bytes.Buffer
	RootCmd.SetOut(&verifyBuf)
	RootCmd.SetErr(&verifyBuf)

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("RootCmd verify error = %v\nOutput: %s", err, verifyBuf.String())
	}
}

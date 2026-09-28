package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/subizu/proviq/pkg/receipt"
)

func TestGenerateAndSignVerify(t *testing.T) {
	pub, priv, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("GenerateEd25519Key() error = %v", err)
	}

	r := receipt.Receipt{
		Version:         "0.1",
		Agent:           "claude-code",
		Intent:          "fix auth vulnerability",
		Action:          "git.commit",
		ActionTimestamp: "2026-09-28T20:00:00Z",
		Inputs: map[string]receipt.Evidence{
			"auth.go": {Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		},
	}

	signed, err := SignReceipt(r, priv)
	if err != nil {
		t.Fatalf("SignReceipt() error = %v", err)
	}

	if signed.Signature == "" {
		t.Fatal("expected non-empty signature")
	}
	if signed.PublicKey == "" {
		t.Fatal("expected non-empty public key in signed receipt")
	}

	// Verify with generated pub key
	if err := VerifyReceipt(signed, pub); err != nil {
		t.Fatalf("VerifyReceipt() error = %v", err)
	}

	// Verify using decoded public key from string
	pubDecoded, err := PublicKeyFromBase64(signed.PublicKey)
	if err != nil {
		t.Fatalf("PublicKeyFromBase64() error = %v", err)
	}
	if err := VerifyReceipt(signed, pubDecoded); err != nil {
		t.Fatalf("VerifyReceipt() with decoded pub key error = %v", err)
	}

	// Tampering test: modify agent name
	tampered := signed
	tampered.Agent = "evil-agent"
	if err := VerifyReceipt(tampered, pub); err == nil {
		t.Fatal("expected verification to fail on tampered receipt")
	}
}

func TestPEMKeySerialization(t *testing.T) {
	pub, priv, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("GenerateEd25519Key() error = %v", err)
	}

	tmpDir := t.TempDir()
	privPath := filepath.Join(tmpDir, "test.key")
	pubPath := filepath.Join(tmpDir, "test.pub")

	if err := WritePrivateKey(priv, privPath); err != nil {
		t.Fatalf("WritePrivateKey() error = %v", err)
	}
	if err := WritePublicKey(pub, pubPath); err != nil {
		t.Fatalf("WritePublicKey() error = %v", err)
	}

	loadedPriv, err := LoadPrivateKey(privPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	loadedPub, err := LoadPublicKey(pubPath)
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	r := receipt.Receipt{
		Version:         "0.1",
		Agent:           "test-agent",
		Intent:          "pem-test",
		Action:          "test",
		ActionTimestamp: "2026-09-28T20:00:00Z",
	}

	signed, err := SignReceipt(r, loadedPriv)
	if err != nil {
		t.Fatalf("SignReceipt with loaded key error = %v", err)
	}

	if err := VerifyReceipt(signed, loadedPub); err != nil {
		t.Fatalf("VerifyReceipt with loaded key error = %v", err)
	}
}

func TestHashDataAndFile(t *testing.T) {
	data := []byte("hello digital provenance")
	expectedHash := "2be7601f5e6abacf0e7123a0c6ab943dd069ca9e3f38e8b2422f6489661ef756"

	if h := HashData(data); h != expectedHash {
		t.Fatalf("HashData() got %s, want %s", h, expectedHash)
	}

	tmpFile := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	fileHash, err := HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}
	if fileHash != expectedHash {
		t.Fatalf("HashFile() got %s, want %s", fileHash, expectedHash)
	}
}

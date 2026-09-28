package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/agent-proof/agent-proof/pkg/receipt"
)

// PublicKey is an Ed25519 public key.
type PublicKey = ed25519.PublicKey

// PrivateKey is an Ed25519 private key.
type PrivateKey = ed25519.PrivateKey

// GenerateEd25519Key creates a new Ed25519 key pair.
func GenerateEd25519Key() (PublicKey, PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating ed25519 key: %w", err)
	}
	return pub, priv, nil
}

// canonicalPayload returns the canonical JSON-marshaled receipt payload (without signature fields)
// that is signed and verified.
func canonicalPayload(r receipt.Receipt) ([]byte, error) {
	return r.MarshalCanonical()
}

// SignReceipt signs a Receipt and returns a SignedReceipt.
func SignReceipt(r receipt.Receipt, priv PrivateKey) (receipt.SignedReceipt, error) {
	payload, err := canonicalPayload(r)
	if err != nil {
		return receipt.SignedReceipt{}, fmt.Errorf("marshaling payload: %w", err)
	}

	sig := ed25519.Sign(priv, payload)

	return receipt.SignedReceipt{
		Receipt:   r,
		Signature: base64.StdEncoding.EncodeToString(sig),
		PublicKey: base64.StdEncoding.EncodeToString(priv.Public().(PublicKey)),
	}, nil
}

// VerifyReceipt verifies that the signature on a SignedReceipt is valid.
func VerifyReceipt(sr receipt.SignedReceipt, pub PublicKey) error {
	payload, err := canonicalPayload(sr.Receipt)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	sig, err := base64.StdEncoding.DecodeString(sr.Signature)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}

	if !ed25519.Verify(pub, payload, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// PublicKeyFromBase64 decodes a base64-encoded public key.
func PublicKeyFromBase64(b64 string) (PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: got %d bytes, expected %d", len(data), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(data), nil
}

// WritePrivateKey writes an Ed25519 private key to a file in PEM format.
func WritePrivateKey(priv PrivateKey, path string) error {
	// Ensure PKCS8 format for portability
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}

	return os.WriteFile(path, pem.EncodeToMemory(block), 0600)
}

// WritePublicKey writes an Ed25519 public key to a file in PEM format.
func WritePublicKey(pub PublicKey, path string) error {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}

	return os.WriteFile(path, pem.EncodeToMemory(block), 0644)
}

// LoadPrivateKey reads an Ed25519 private key from a PEM file.
func LoadPrivateKey(path string) (PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decoding PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	priv, ok := key.(PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ed25519 private key")
	}

	return priv, nil
}

// LoadPublicKey reads an Ed25519 public key from a PEM file.
func LoadPublicKey(path string) (PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decoding PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	pub, ok := key.(PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ed25519 public key")
	}

	return pub, nil
}

// HashData returns a sha256 hex hash of the given data.
func HashData(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// HashFile hashes a file and returns the hex hash.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return HashData(data), nil
}

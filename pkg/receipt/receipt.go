package receipt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Receipt represents the payload of an agent action receipt.
// It contains all the information needed to independently verify
// that a specific agent performed a specific action.
type Receipt struct {
	// Version is the receipt schema version
	Version string `json:"version"`

	// Agent is the identifier of the agent that performed the action
	Agent string `json:"agent"`

	// Intent describes the human-readable intent behind the action
	Intent string `json:"intent"`

	// Action describes the type of action performed (e.g. "git.commit", "file.edit")
	Action string `json:"action"`

	// Inputs maps input names to their evidence (content hash)
	Inputs map[string]Evidence `json:"inputs,omitempty"`

	// Policies maps policy names to their content hashes
	Policies map[string]string `json:"policies,omitempty"`

	// PolicyRef is a hash identifying the policy set
	PolicyRef string `json:"policy_ref,omitempty"`

	// GitContext contains optional git repository context
	GitContext *GitContext `json:"git_context,omitempty"`

	// ActionTimestamp is when the action was performed (RFC3339)
	ActionTimestamp string `json:"action_timestamp"`

	// Outputs maps output names to their evidence
	Outputs map[string]Evidence `json:"outputs,omitempty"`
}

// Evidence represents a piece of evidence with its content hash.
type Evidence struct {
	Hash string `json:"hash"`         // sha256 hex digest of content
	URI  string `json:"uri,omitempty"` // optional URI where content can be retrieved
}

// GitContext captures git repository state at the time of the action.
type GitContext struct {
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Author     string `json:"author,omitempty"`
	Committer  string `json:"committer,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

// SignedReceipt is a Receipt with its cryptographic signature and public key.
type SignedReceipt struct {
	Receipt
	Signature string `json:"signature"`  // base64(ed25519_sign(payload))
	PublicKey string `json:"public_key"` // base64(ed25519_pubkey)
}

// HashFile computes the sha256 hash of a file and returns an Evidence struct.
func HashFile(path string) (Evidence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Evidence{}, fmt.Errorf("reading file %s: %w", path, err)
	}

	h := sha256.Sum256(data)
	return Evidence{
		Hash: hex.EncodeToString(h[:]),
		URI:  path,
	}, nil
}

// HashBytes computes the sha256 hash of raw data.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashDir computes hashes for all files in a directory.
func HashDir(dir string) (map[string]Evidence, error) {
	evidence := make(map[string]Evidence)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		ev, err := HashFile(path)
		if err != nil {
			continue // skip files we can't read
		}
		evidence[path] = ev
	}

	return evidence, nil
}

// MarshalCanonical returns the canonical JSON marshaling of the receipt,
// used for signing and verification (RFC 8785 compatible).
func (r *Receipt) MarshalCanonical() ([]byte, error) {
	return Canonicalize(r)
}

// UnmarshalReceipt parses a SignedReceipt from JSON bytes.
func UnmarshalReceipt(data []byte) (SignedReceipt, error) {
	var sr SignedReceipt
	err := json.Unmarshal(data, &sr)
	return sr, err
}

// String returns a human-readable summary of the receipt.
func (sr *SignedReceipt) String() string {
	return fmt.Sprintf("Receipt{version=%s, agent=%s, action=%s, timestamp=%s}",
		sr.Version, sr.Agent, sr.Action, sr.ActionTimestamp)
}

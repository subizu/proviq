package receipt

import (
	"testing"
)

func TestCanonicalizeDeterministic(t *testing.T) {
	// Two JSON objects with keys in different orders and different whitespace
	json1 := []byte(`{"z": 100, "a": "hello", "m": [3, 2, 1], "b": {"nested_z": true, "nested_a": false}}`)
	json2 := []byte(`{
		"a": "hello",
		"b": {
			"nested_a": false,
			"nested_z": true
		},
		"m": [3, 2, 1],
		"z": 100
	}`)

	c1, err := CanonicalizeJSONBytes(json1)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes(json1) error = %v", err)
	}

	c2, err := CanonicalizeJSONBytes(json2)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes(json2) error = %v", err)
	}

	if string(c1) != string(c2) {
		t.Fatalf("Canonical JSON mismatch:\nC1: %s\nC2: %s", string(c1), string(c2))
	}

	expected := `{"a":"hello","b":{"nested_a":false,"nested_z":true},"m":[3,2,1],"z":100}`
	if string(c1) != expected {
		t.Fatalf("Got %s, want %s", string(c1), expected)
	}
}

func TestReceiptMarshalCanonical(t *testing.T) {
	r := Receipt{
		Version:         "0.1",
		Agent:           "test-agent",
		Intent:          "canonicalize",
		Action:          "test",
		ActionTimestamp: "2026-09-28T22:00:00Z",
		Inputs: map[string]Evidence{
			"z_file.go": {Hash: "1111"},
			"a_file.go": {Hash: "2222"},
		},
	}

	canonicalBytes, err := r.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical() error = %v", err)
	}

	// Verify that a_file comes before z_file in canonical serialized output
	expectedSubstr := `"inputs":{"a_file.go":{"hash":"2222"},"z_file.go":{"hash":"1111"}}`
	if !contains(string(canonicalBytes), expectedSubstr) {
		t.Fatalf("canonical output does not contain sorted inputs: %s", string(canonicalBytes))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package receipt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Canonicalize converts an arbitrary data structure (or raw JSON bytes)
// into a deterministic, canonical JSON representation (RFC 8785 compatible).
func Canonicalize(v any) ([]byte, error) {
	// First marshal to standard JSON to normalize struct tags and types
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("canonicalize marshal: %w", err)
	}

	// Parse into generic JSON value
	var generic any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // avoid float precision issues
	if err := dec.Decode(&generic); err != nil {
		return nil, fmt.Errorf("canonicalize decode: %w", err)
	}

	var buf bytes.Buffer
	if err := formatCanonical(&buf, generic); err != nil {
		return nil, fmt.Errorf("canonicalize format: %w", err)
	}

	return buf.Bytes(), nil
}

// CanonicalizeJSONBytes canonicalizes raw JSON bytes.
func CanonicalizeJSONBytes(data []byte) ([]byte, error) {
	var generic any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, fmt.Errorf("canonicalize JSON bytes: %w", err)
	}

	var buf bytes.Buffer
	if err := formatCanonical(&buf, generic); err != nil {
		return nil, fmt.Errorf("canonicalize format: %w", err)
	}

	return buf.Bytes(), nil
}

func formatCanonical(buf *bytes.Buffer, v any) error {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		buf.WriteString(val.String())
	case string:
		escaped, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(escaped)
	case []any:
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := formatCanonical(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		buf.WriteByte('{')
		// Sort keys in lexicographical (UTF-8 byte) order
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			escapedKey, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(escapedKey)
			buf.WriteByte(':')
			if err := formatCanonical(buf, val[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case float64:
		buf.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
	case int:
		buf.WriteString(strconv.Itoa(val))
	case int64:
		buf.WriteString(strconv.FormatInt(val, 10))
	default:
		// Fallback to standard json encoding
		data, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(data)
	}
	return nil
}

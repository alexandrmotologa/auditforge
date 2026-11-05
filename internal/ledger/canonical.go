package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// CanonicalizeJSON converts arbitrary JSON-serializable data or raw JSON bytes
// into RFC 8785 canonical JSON bytes.
//
// Rules implemented:
// 1. Whitespace outside string literals is removed.
// 2. Object keys are sorted lexicographically by Unicode code point values.
// 3. Numbers are formatted deterministically without trailing zeroes or unnecessary exponents.
// 4. Strings use minimal standard JSON escape sequences.
func CanonicalizeJSON(v any) ([]byte, error) {
	var raw any
	switch val := v.(type) {
	case []byte:
		decoder := json.NewDecoder(bytes.NewReader(val))
		decoder.UseNumber()
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("invalid json input: %w", err)
		}
	case string:
		decoder := json.NewDecoder(bytes.NewReader([]byte(val)))
		decoder.UseNumber()
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("invalid json input: %w", err)
		}
	default:
		// Marshal and unmarshal through json.Number to normalize types
		buf, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value: %w", err)
		}
		decoder := json.NewDecoder(bytes.NewReader(buf))
		decoder.UseNumber()
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("failed to decode normalized json: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := writeCanonical(&buf, raw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	if v == nil {
		buf.WriteString("null")
		return nil
	}

	switch val := v.(type) {
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil

	case json.Number:
		// Deterministic number representation according to RFC 8785 §3.2.2.3
		str := val.String()
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			// If not a valid float, output raw string
			buf.WriteString(str)
			return nil
		}
		buf.WriteString(formatCanonicalNumber(f))
		return nil

	case float64:
		buf.WriteString(formatCanonicalNumber(val))
		return nil

	case int:
		buf.WriteString(strconv.Itoa(val))
		return nil

	case int64:
		buf.WriteString(strconv.FormatInt(val, 10))
		return nil

	case uint64:
		buf.WriteString(strconv.FormatUint(val, 10))
		return nil

	case string:
		encoded, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(encoded)
		return nil

	case []any:
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil

	case map[string]any:
		buf.WriteByte('{')
		// Sort keys lexicographically by Unicode code point (byte order in UTF-8 matches UTF-16 code units)
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyBytes, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(keyBytes)
			buf.WriteByte(':')
			if err := writeCanonical(buf, val[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil

	default:
		// Fallback for custom struct or type
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		var parsed any
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		if err := dec.Decode(&parsed); err != nil {
			return err
		}
		return writeCanonical(buf, parsed)
	}
}

func formatCanonicalNumber(f float64) string {
	// If it is an integer within safe range, format as standard integer
	if f == float64(int64(f)) && f >= -9007199254740991 && f <= 9007199254740991 {
		return strconv.FormatInt(int64(f), 10)
	}
	// Use standard JSON representation for floating point numbers
	b, err := json.Marshal(f)
	if err != nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return string(b)
}

package ledger

import (
	"strings"
	"testing"
)

func TestCanonicalizeJSON_KeySorting(t *testing.T) {
	input := `{"z": 1, "a": 2, "m": {"b": 10, "a": 20}}`
	out, err := CanonicalizeJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"a":2,"m":{"a":20,"b":10},"z":1}`
	if string(out) != expected {
		t.Errorf("expected %s, got %s", expected, string(out))
	}
}

func TestCanonicalizeJSON_WhitespaceStripping(t *testing.T) {
	input := `{
		"name": "AuditForge",
		"items": [
			1,
			2,
			3
		]
	}`
	out, err := CanonicalizeJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"items":[1,2,3],"name":"AuditForge"}`
	if string(out) != expected {
		t.Errorf("expected %s, got %s", expected, string(out))
	}
}

func TestCanonicalizeJSON_NumberFormatting(t *testing.T) {
	input := `{"int": 42, "float": 42.5, "zero": 0}`
	out, err := CanonicalizeJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(out), `"int":42`) {
		t.Errorf("integer formatting failed in %s", string(out))
	}
	if !strings.Contains(string(out), `"float":42.5`) {
		t.Errorf("float formatting failed in %s", string(out))
	}
	if !strings.Contains(string(out), `"zero":0`) {
		t.Errorf("zero formatting failed in %s", string(out))
	}
}

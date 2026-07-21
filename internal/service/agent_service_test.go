package service

import "testing"

// ===== cleanJSON =====

func TestCleanJSON_PlainJSON(t *testing.T) {
	input := `{"key": "value"}`
	got := cleanJSON(input)
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("cleanJSON(plain) = %q, want %q", got, want)
	}
}

func TestCleanJSON_JsonCodeBlock(t *testing.T) {
	input := "```json\n{\"key\": \"value\"}\n```"
	got := cleanJSON(input)
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("cleanJSON(```json) = %q, want %q", got, want)
	}
}

func TestCleanJSON_GenericCodeBlock(t *testing.T) {
	input := "```\n{\"key\": \"value\"}\n```"
	got := cleanJSON(input)
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("cleanJSON(```) = %q, want %q", got, want)
	}
}

func TestCleanJSON_WithLeadingTrailingWhitespace(t *testing.T) {
	input := "  \n  ```json\n{\"key\": \"value\"}\n```  \n  "
	got := cleanJSON(input)
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("cleanJSON(whitespace) = %q, want %q", got, want)
	}
}

func TestCleanJSON_EmptyString(t *testing.T) {
	got := cleanJSON("")
	if got != "" {
		t.Errorf("cleanJSON('') = %q, want empty", got)
	}
}

func TestCleanJSON_OnlyWhitespace(t *testing.T) {
	got := cleanJSON("   \n\t  ")
	if got != "" {
		t.Errorf("cleanJSON(whitespace-only) = %q, want empty", got)
	}
}

func TestCleanJSON_NestedJSON(t *testing.T) {
	input := "```json\n{\"a\": [1, 2, {\"b\": \"c\"}]}\n```"
	got := cleanJSON(input)
	want := `{"a": [1, 2, {"b": "c"}]}`
	if got != want {
		t.Errorf("cleanJSON(nested) = %q, want %q", got, want)
	}
}

func TestCleanJSON_NoTrailingBackticks(t *testing.T) {
	input := "```json\n{\"key\": \"value\"}"
	got := cleanJSON(input)
	want := `{"key": "value"}`
	if got != want {
		t.Errorf("cleanJSON(no trailing ```) = %q, want %q", got, want)
	}
}

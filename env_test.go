package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp env file: %v", err)
	}
	return path
}

func TestParseEnvFile_BasicKeyValue(t *testing.T) {
	path := writeTempEnvFile(t, "FOO=bar\nBAZ=qux\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
	if env["BAZ"] != "qux" {
		t.Errorf("BAZ = %q, want %q", env["BAZ"], "qux")
	}
}

func TestParseEnvFile_SkipsComments(t *testing.T) {
	path := writeTempEnvFile(t, "# this is a comment\nFOO=bar\n# another comment\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(env))
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
}

func TestParseEnvFile_SkipsBlankLines(t *testing.T) {
	path := writeTempEnvFile(t, "\n\nFOO=bar\n\n\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(env))
	}
}

func TestParseEnvFile_EmptyValue(t *testing.T) {
	path := writeTempEnvFile(t, "FOO=\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, ok := env["FOO"]
	if !ok {
		t.Fatal("expected FOO to be present in map")
	}
	if val != "" {
		t.Errorf("FOO = %q, want empty string", val)
	}
}

func TestParseEnvFile_ErrorOnMissingFile(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/path/.env")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseEnvFile_ReturnsEmptyMapForEmptyFile(t *testing.T) {
	path := writeTempEnvFile(t, "")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty map, got %d entries", len(env))
	}
}

func TestParseEnvFile_SkipsInvalidLines(t *testing.T) {
	// Lines that don't match KEY=VALUE are silently skipped.
	path := writeTempEnvFile(t, "not-valid\nFOO=bar\nalso not valid\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(env))
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
}

func TestParseEnvFile_MultipleEntries(t *testing.T) {
	path := writeTempEnvFile(t, "DATABASE_URL=postgres://localhost/db\nREDIS_URL=redis://localhost:6379\nDEBUG=true\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(env))
	}
}

func TestParseEnvValue_Unquoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"value=with=equals", "value=with=equals"},
		{"", ""},
		{"123", "123"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseEnvValue(tc.input)
			if got != tc.want {
				t.Errorf("parseEnvValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseEnvValue_SingleQuoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"'hello'", "hello"},
		{"'hello world'", "hello world"},
		// Single-quoted: no escape processing, backslash-n is literal.
		{"'hello\\nworld'", `hello\nworld`},
		{"'it''s'", "it''s"}, // Only outermost quotes stripped; inner single quotes are literal.
		{"''", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseEnvValue(tc.input)
			if got != tc.want {
				t.Errorf("parseEnvValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseEnvValue_DoubleQuoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`"hello world"`, "hello world"},
		{`"hello\nworld"`, "hello\nworld"}, // \n becomes newline
		{`"hello\\world"`, `hello\world`},  // \\ becomes single backslash
		{`"line1\nline2\nline3"`, "line1\nline2\nline3"},
		{`""`, ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseEnvValue(tc.input)
			if got != tc.want {
				t.Errorf("parseEnvValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseEnvValue_SingleCharNotQuoted(t *testing.T) {
	// A single character that happens to equal a quote character is not treated
	// as a quoted string (len < 2 requirement).
	got := parseEnvValue("'")
	if got != "'" {
		t.Errorf("parseEnvValue(\"'\") = %q, want \"'\"", got)
	}
	got = parseEnvValue(`"`)
	if got != `"` {
		t.Errorf("parseEnvValue(`\"`) = %q, want `\"`", got)
	}
}

func TestParseEnvValue_MismatchedQuotes(t *testing.T) {
	// Mismatched quotes should not be treated as quoted — returned as-is.
	got := parseEnvValue("'hello\"")
	if got != "'hello\"" {
		t.Errorf("parseEnvValue(%q) = %q, want %q", "'hello\"", got, "'hello\"")
	}
}

func TestParseEnvFile_ValueWithEquals(t *testing.T) {
	// Values containing equals signs should be preserved intact.
	path := writeTempEnvFile(t, "DATABASE_URL=postgres://user:pass@host/db?sslmode=disable\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "postgres://user:pass@host/db?sslmode=disable"
	if env["DATABASE_URL"] != want {
		t.Errorf("DATABASE_URL = %q, want %q", env["DATABASE_URL"], want)
	}
}

func TestParseEnvFile_UnderscoreAndDigitsInKey(t *testing.T) {
	path := writeTempEnvFile(t, "MY_VAR_123=value\n")

	env, err := ParseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["MY_VAR_123"] != "value" {
		t.Errorf("MY_VAR_123 = %q, want %q", env["MY_VAR_123"], "value")
	}
}

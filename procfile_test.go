package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempProcfile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "Procfile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp Procfile: %v", err)
	}
	return path
}

func TestParseProcfile_BasicEntries(t *testing.T) {
	path := writeTempProcfile(t, "web: bundle exec rails server\nworker: bundle exec sidekiq\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Name != "web" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "web")
	}
	if entries[0].Command != "bundle exec rails server" {
		t.Errorf("entries[0].Command = %q, want %q", entries[0].Command, "bundle exec rails server")
	}
	if entries[1].Name != "worker" {
		t.Errorf("entries[1].Name = %q, want %q", entries[1].Name, "worker")
	}
	if entries[1].Command != "bundle exec sidekiq" {
		t.Errorf("entries[1].Command = %q, want %q", entries[1].Command, "bundle exec sidekiq")
	}
}

func TestParseProcfile_SkipsBlankLines(t *testing.T) {
	path := writeTempProcfile(t, "\n\nweb: python app.py\n\n\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "web" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "web")
	}
}

func TestParseProcfile_SkipsCommentLines(t *testing.T) {
	path := writeTempProcfile(t, "# this is a comment\nweb: python app.py\n# another comment\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "web" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "web")
	}
}

func TestParseProcfile_SkipsInvalidLines(t *testing.T) {
	// Lines without "name: command" format are silently skipped.
	path := writeTempProcfile(t, "this is not valid\nweb: python app.py\nalso invalid\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "web" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "web")
	}
}

func TestParseProcfile_ErrorOnNoEntries(t *testing.T) {
	path := writeTempProcfile(t, "# only comments\n\n# and blank lines\n")

	_, err := ParseProcfile(path)
	if err == nil {
		t.Fatal("expected error for empty Procfile, got nil")
	}
}

func TestParseProcfile_ErrorOnMissingFile(t *testing.T) {
	_, err := ParseProcfile("/nonexistent/path/Procfile")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseProcfile_PreservesOrder(t *testing.T) {
	path := writeTempProcfile(t, "web: cmd1\nworker: cmd2\nclock: cmd3\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	names := []string{entries[0].Name, entries[1].Name, entries[2].Name}
	expected := []string{"web", "worker", "clock"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("entries[%d].Name = %q, want %q", i, names[i], want)
		}
	}
}

func TestParseProcfile_AllowedNameCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"alphanumeric", "web123: cmd\n", false},
		{"underscore", "my_worker: cmd\n", false},
		{"hyphen", "my-worker: cmd\n", false},
		{"uppercase", "WebServer: cmd\n", false},
		{"mixed", "Web_Server-1: cmd\n", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempProcfile(t, tc.content)
			entries, err := ParseProcfile(path)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(entries))
			}
		})
	}
}

func TestParseProcfile_CommandWithColonInIt(t *testing.T) {
	// The command itself can contain colons.
	path := writeTempProcfile(t, "web: sh -c 'echo hello:world'\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Command != "sh -c 'echo hello:world'" {
		t.Errorf("Command = %q, want %q", entries[0].Command, "sh -c 'echo hello:world'")
	}
}

func TestParseProcfile_TrimsWhitespaceAroundCommand(t *testing.T) {
	path := writeTempProcfile(t, "web:   python app.py   \n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Command is trimmed; leading spaces after colon are consumed by regex \s*,
	// trailing spaces are trimmed by TrimSpace in the source.
	if entries[0].Command != "python app.py" {
		t.Errorf("Command = %q, want %q", entries[0].Command, "python app.py")
	}
}

func TestParseProcfile_SingleEntry(t *testing.T) {
	path := writeTempProcfile(t, "web: ./server\n")

	entries, err := ParseProcfile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

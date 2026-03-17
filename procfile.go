package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var procfileRe = regexp.MustCompile(`^([A-Za-z0-9_-]+):\s*(.+)$`)

// Entry represents a single Procfile process definition.
type Entry struct {
	Name    string
	Command string
}

// ParseProcfile reads the Procfile at path and returns ordered entries.
func ParseProcfile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := procfileRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		entries = append(entries, Entry{Name: m[1], Command: strings.TrimSpace(m[2])})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no processes found in %s", path)
	}
	return entries, nil
}

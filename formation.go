package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseFormation parses a formation string like "all=1,web=2,worker=3".
// Returns a map of process name → count. "all" sets the default.
func ParseFormation(spec string) (map[string]int, int, error) {
	formation := make(map[string]int)
	defaultCount := 1

	if spec == "" {
		return formation, defaultCount, nil
	}

	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx < 0 {
			return nil, 0, fmt.Errorf("invalid formation entry: %q", part)
		}
		name := strings.TrimSpace(part[:idx])
		countStr := strings.TrimSpace(part[idx+1:])
		n, err := strconv.Atoi(countStr)
		if err != nil || n < 0 {
			return nil, 0, fmt.Errorf("invalid count for %q: %q", name, countStr)
		}
		if name == "all" {
			defaultCount = n
		} else {
			formation[name] = n
		}
	}
	return formation, defaultCount, nil
}

// CountFor returns how many instances of processName should be spawned.
func CountFor(formation map[string]int, defaultCount int, processName string) int {
	if n, ok := formation[processName]; ok {
		return n
	}
	return defaultCount
}

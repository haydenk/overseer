package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envRe = regexp.MustCompile(`^([A-Za-z_0-9]+)=(.*)$`)

// ParseEnvFile reads a .env file and returns a map of key→value pairs.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := envRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], m[2]
		env[key] = parseEnvValue(val)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return env, nil
}

func parseEnvValue(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return v[1 : len(v)-1]
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		inner := v[1 : len(v)-1]
		inner = strings.ReplaceAll(inner, `\n`, "\n")
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	return v
}

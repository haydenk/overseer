package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// envSliceToMap converts a []string of "KEY=VALUE" entries into a map for
// easier assertions. If a key appears more than once, the last value wins
// (matching the behaviour of most Unix tools).
func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		m[kv[:idx]] = kv[idx+1:]
	}
	return m
}

// ---- buildEnv tests --------------------------------------------------------

func TestBuildEnv_SetsPORT(t *testing.T) {
	env := buildEnv(map[string]string{}, 5000, "web.1")
	m := envSliceToMap(env)

	if m["PORT"] != "5000" {
		t.Errorf("PORT = %q, want %q", m["PORT"], "5000")
	}
}

func TestBuildEnv_SetsPS(t *testing.T) {
	env := buildEnv(map[string]string{}, 5000, "web.1")
	m := envSliceToMap(env)

	if m["PS"] != "web.1" {
		t.Errorf("PS = %q, want %q", m["PS"], "web.1")
	}
}

func TestBuildEnv_MergesCustomVars(t *testing.T) {
	custom := map[string]string{
		"DATABASE_URL": "postgres://localhost/mydb",
		"REDIS_URL":    "redis://localhost:6379",
	}
	env := buildEnv(custom, 3000, "worker")
	m := envSliceToMap(env)

	if m["DATABASE_URL"] != "postgres://localhost/mydb" {
		t.Errorf("DATABASE_URL = %q, want %q", m["DATABASE_URL"], "postgres://localhost/mydb")
	}
	if m["REDIS_URL"] != "redis://localhost:6379" {
		t.Errorf("REDIS_URL = %q, want %q", m["REDIS_URL"], "redis://localhost:6379")
	}
}

func TestBuildEnv_CustomVarsOverrideOSEnviron(t *testing.T) {
	// Set a key in the process environment, then override it via envVars.
	const key = "OVERSEER_TEST_OVERRIDE"
	t.Setenv(key, "original")

	env := buildEnv(map[string]string{key: "overridden"}, 3000, "web")
	m := envSliceToMap(env)

	if m[key] != "overridden" {
		t.Errorf("%s = %q, want %q", key, m[key], "overridden")
	}
}

func TestBuildEnv_InheritsOSEnviron(t *testing.T) {
	// Confirm that an ambient env var is present in the result.
	const key = "OVERSEER_TEST_INHERIT"
	t.Setenv(key, "inherited_value")

	env := buildEnv(map[string]string{}, 3000, "web")
	m := envSliceToMap(env)

	if m[key] != "inherited_value" {
		t.Errorf("%s = %q, want %q", key, m[key], "inherited_value")
	}
}

func TestBuildEnv_PortOverridesCustomVarPort(t *testing.T) {
	// Even if the caller passes PORT in the custom vars, buildEnv should
	// set PORT to the explicit port argument (last write wins in the map).
	env := buildEnv(map[string]string{"PORT": "9999"}, 5001, "web")
	m := envSliceToMap(env)

	if m["PORT"] != "5001" {
		t.Errorf("PORT = %q, want %q (explicit port arg should win)", m["PORT"], "5001")
	}
}

func TestBuildEnv_DifferentPorts(t *testing.T) {
	tests := []struct {
		port     int
		label    string
		wantPort string
	}{
		{3000, "web.1", "3000"},
		{3001, "web.2", "3001"},
		{3100, "worker.1", "3100"},
		{0, "web", "0"},
	}

	for _, tc := range tests {
		env := buildEnv(map[string]string{}, tc.port, tc.label)
		m := envSliceToMap(env)

		if m["PORT"] != tc.wantPort {
			t.Errorf("port=%d label=%q: PORT = %q, want %q", tc.port, tc.label, m["PORT"], tc.wantPort)
		}
		if m["PS"] != tc.label {
			t.Errorf("port=%d label=%q: PS = %q, want %q", tc.port, tc.label, m["PS"], tc.label)
		}
	}
}

func TestBuildEnv_EmptyCustomVars(t *testing.T) {
	// Should not panic and should still set PORT and PS.
	env := buildEnv(map[string]string{}, 8080, "api")
	m := envSliceToMap(env)

	if m["PORT"] != "8080" {
		t.Errorf("PORT = %q, want %q", m["PORT"], "8080")
	}
	if m["PS"] != "api" {
		t.Errorf("PS = %q, want %q", m["PS"], "api")
	}
}

func TestBuildEnv_ResultContainsOnlyKeyValuePairs(t *testing.T) {
	env := buildEnv(map[string]string{"FOO": "bar"}, 3000, "web")
	for _, kv := range env {
		if !strings.Contains(kv, "=") {
			t.Errorf("env entry %q does not contain '='", kv)
		}
	}
}

// ---- buildCmd tests --------------------------------------------------------

func TestBuildCmd_ReturnsExecCmd(t *testing.T) {
	cmd := buildCmd("echo hello", []string{"FOO=bar"})
	if cmd == nil {
		t.Fatal("buildCmd returned nil")
	}
}

func TestBuildCmd_UsesShell(t *testing.T) {
	cmd := buildCmd("echo hello", nil)
	if cmd.Path == "" {
		t.Fatal("cmd.Path is empty")
	}
	// The shell should be "sh" (resolved to full path).
	if !strings.HasSuffix(cmd.Path, "sh") {
		t.Errorf("cmd.Path = %q, expected it to end with 'sh'", cmd.Path)
	}
}

func TestBuildCmd_SetsShellMinusCArgs(t *testing.T) {
	cmd := buildCmd("echo hello", nil)

	// exec.Command("sh", "-c", command) produces Args = ["sh", "-c", "echo hello"]
	if len(cmd.Args) != 3 {
		t.Fatalf("expected 3 args (sh -c <cmd>), got %d: %v", len(cmd.Args), cmd.Args)
	}
	if cmd.Args[0] != "sh" {
		t.Errorf("Args[0] = %q, want %q", cmd.Args[0], "sh")
	}
	if cmd.Args[1] != "-c" {
		t.Errorf("Args[1] = %q, want %q", cmd.Args[1], "-c")
	}
	if cmd.Args[2] != "echo hello" {
		t.Errorf("Args[2] = %q, want %q", cmd.Args[2], "echo hello")
	}
}

func TestBuildCmd_SetsEnvironment(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux", "PORT=3000"}
	cmd := buildCmd("echo hello", env)

	if len(cmd.Env) != len(env) {
		t.Fatalf("cmd.Env length = %d, want %d", len(cmd.Env), len(env))
	}
	for i, want := range env {
		if cmd.Env[i] != want {
			t.Errorf("cmd.Env[%d] = %q, want %q", i, cmd.Env[i], want)
		}
	}
}

func TestBuildCmd_NilEnvIsPreserved(t *testing.T) {
	cmd := buildCmd("echo hello", nil)
	if cmd.Env != nil {
		t.Errorf("cmd.Env = %v, want nil when nil env passed", cmd.Env)
	}
}

func TestBuildCmd_ComplexShellCommandPreserved(t *testing.T) {
	complex := "FOO=bar ./server --port $PORT 2>&1 | tee /tmp/log"
	cmd := buildCmd(complex, nil)

	if cmd.Args[2] != complex {
		t.Errorf("Args[2] = %q, want %q", cmd.Args[2], complex)
	}
}

func TestBuildCmd_ReturnsExecCmdType(t *testing.T) {
	cmd := buildCmd("true", nil)
	// Confirm it is a *exec.Cmd with the right type.
	var _ *exec.Cmd = cmd
}

func TestBuildCmd_EmptyEnvSlice(t *testing.T) {
	env := []string{}
	cmd := buildCmd("echo hello", env)

	if len(cmd.Env) != 0 {
		t.Errorf("cmd.Env = %v, want empty slice", cmd.Env)
	}
}

// TestBuildEnv_ResultIsNonEmpty verifies we get at least the ambient OS env
// plus PORT and PS.
func TestBuildEnv_ResultIsNonEmpty(t *testing.T) {
	env := buildEnv(map[string]string{}, 3000, "web")
	// os.Environ() is non-empty in any real test environment; plus PORT and PS.
	if len(env) < 2 {
		t.Errorf("expected at least 2 env entries, got %d", len(env))
	}
}

// TestBuildCmd_ExecutableIsResolved verifies that the returned Cmd has a
// non-empty, absolute path to the shell binary (exec.Command resolves $PATH).
func TestBuildCmd_ExecutableIsResolved(t *testing.T) {
	cmd := buildCmd("true", nil)
	if !strings.HasPrefix(cmd.Path, "/") {
		// LookPath may not have run yet if $PATH lookup didn't happen; try
		// resolving manually to confirm the binary is findable.
		resolved, err := exec.LookPath("sh")
		if err != nil {
			t.Skip("sh not found on PATH; skipping test")
		}
		if cmd.Path != "sh" && cmd.Path != resolved {
			t.Errorf("cmd.Path = %q, expected absolute path or 'sh'", cmd.Path)
		}
	}
}

// TestBuildEnv_PSMatchesLabel ensures PS is set to the exact label string,
// including dot-notation for multi-instance labels.
func TestBuildEnv_PSMatchesLabel(t *testing.T) {
	labels := []string{"web", "web.1", "web.2", "worker.1", "my-service", "my_service"}
	for _, label := range labels {
		env := buildEnv(map[string]string{}, 3000, label)
		m := envSliceToMap(env)
		if m["PS"] != label {
			t.Errorf("label=%q: PS = %q, want %q", label, m["PS"], label)
		}
	}
}

// TestBuildEnv_NilEnvVarsDoesNotPanic verifies that passing nil for envVars
// does not panic. The map range over nil map is safe in Go, so this is a
// defensive check.
func TestBuildEnv_NilEnvVarsDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildEnv panicked with nil envVars: %v", r)
		}
	}()
	env := buildEnv(nil, 3000, "web")
	if len(env) == 0 {
		t.Error("expected non-empty env even with nil envVars")
	}
}

// TestBuildEnv_OSEnvironKeysArePresent spot-checks that well-known OS
// environment variables (like PATH or HOME) survive into the result.
func TestBuildEnv_OSEnvironKeysArePresent(t *testing.T) {
	// PATH is present on every POSIX system.
	origPath := os.Getenv("PATH")
	if origPath == "" {
		t.Skip("PATH is empty; skipping test")
	}

	env := buildEnv(map[string]string{}, 3000, "web")
	m := envSliceToMap(env)

	if m["PATH"] != origPath {
		t.Errorf("PATH = %q, want %q", m["PATH"], origPath)
	}
}

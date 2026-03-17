# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

This project uses `mise` as the task runner. All development tasks are run via `mise run <task>`.

| Task | Command | Description |
|---|---|---|
| Build | `mise run build` | Compiles binary to `bin/overseer` |
| Test | `mise run test` | Runs `go test ./...` |
| Single test | `go test -run TestName ./...` | Standard Go test filtering |
| Format | `mise run fmt` | Formats code with gofmt/goimports |
| Lint | `mise run lint` | Runs `go vet` + `staticcheck` |
| Full QA | `mise run check` | Runs fmt → lint → test in sequence |
| Dev run | `mise run dev` | Runs processes from `Procfile.dev` or `Procfile` |
| Install | `mise run install` | Installs binary to `$GOBIN` (`.mise/bin/`) |
| Clean | `mise run clean` | Removes build artifacts |

The project uses Go 1.26+ with `CGO_ENABLED=0` for static builds.

## Architecture

Overseer is a Procfile-based process manager (a zero-dependency Go clone of Foreman). It reads a Procfile, spawns each process with shell execution, streams prefixed/colored output, and handles graceful shutdown.

### Core Components

- **main.go** — CLI entry point with three commands: `check`, `start`, `run`. Handles flag parsing and delegates to the runner or standalone execution.
- **runner.go** — The orchestration core. `Run()` spawns all process instances, multiplexes their stdout/stderr through the `Writer`, forwards OS signals to all children, and manages graceful shutdown (SIGTERM → wait → SIGKILL).
- **procfile.go** — Parses `name: command` format, validates with regex, skips comments and blank lines.
- **formation.go** — Parses formation strings like `"all=1,web=2,worker=3"`. The `all` key sets the default count for any unlisted process type.
- **env.go** — Parses `.env` files supporting unquoted, single-quoted, and double-quoted values (double-quoted handles `\n` and `\\` escapes).
- **output.go** — Thread-safe `Writer` struct with mutex. Cycles through 6 ANSI colors, pads labels to uniform width, and optionally prepends timestamps.

### Data Flow

```
Procfile + .env + flags
  → parse entries, env vars, formation
  → create Instances (with port, color, label)
  → Run(): spawn all, pipe stdout/stderr to Writer
  → signal forwarding loop
  → graceful shutdown on first exit or signal
  → exit with first process's exit code
```

### Port Assignment

```
PORT = basePort + (processTypeIndex × 100) + instanceIndex
```

Example: base 5000, web=2, worker=2 → web.1:5000, web.2:5001, worker.1:5100, worker.2:5101

### Key Design Decisions

- **Zero external dependencies** — pure Go standard library only; keep it this way
- **Process execution** — commands run via `/bin/sh -c <command>` to support shell features in Procfiles
- **Signal handling** — SIGINT/SIGTERM trigger graceful shutdown; SIGHUP/SIGUSR1/SIGUSR2 are forwarded to all child processes
- **First exit wins** — when any process exits, the runner initiates shutdown of all others

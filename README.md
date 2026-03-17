# overseer

[![Go Reference](https://pkg.go.dev/badge/github.com/haydenk/overseer.svg)](https://pkg.go.dev/github.com/haydenk/overseer)
[![Go Report Card](https://goreportcard.com/badge/github.com/haydenk/overseer)](https://goreportcard.com/report/github.com/haydenk/overseer)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go)

A zero-dependency Go clone of [foreman](https://github.com/ddollar/foreman) — a `Procfile`-based process manager. Overseer reads a `Procfile`, spawns each process, streams prefixed and colored output, and handles graceful shutdown on signals.

## Features

- **Zero external dependencies** — pure Go standard library
- **Colored, timestamped output** — one color per process type, padded labels
- **Formation support** — run multiple instances of any process type
- **Port assignment** — auto-assigns `PORT` per instance (`base + processIndex*100 + instanceIndex`)
- **Dual-stack networking** — reports IPv4 and IPv6 bind addresses at startup
- **`.env` file loading** — single-quoted, double-quoted, and unquoted values
- **Graceful shutdown** — `SIGTERM` to all children, `SIGKILL` after configurable timeout
- **Signal forwarding** — `SIGUSR1`/`SIGUSR2` forwarded to all children

## Installation

**Requires Go 1.26+** (managed automatically via [mise](https://mise.jdx.dev) if you use it).

```sh
go install github.com/haydenk/overseer@latest
```

Or build from source:

```sh
git clone https://github.com/haydenk/overseer.git
cd overseer
mise run build        # → bin/overseer
# or
go build -o bin/overseer .
```

## Usage

### `overseer check`

Validate a Procfile and list detected process types.

```sh
overseer check [-f Procfile]
```

```
valid procfile detected (web, worker)
```

### `overseer start`

Start all processes (or a single named process) defined in the Procfile.

```sh
overseer start [PROCESS] [flags]

  -f string         Procfile path (default "Procfile")
  -e string         .env file path (default ".env")
  -m string         formation, e.g. all=1,web=2,worker=3
  -p string         base port — overrides $PORT (default 5000)
  -t int            graceful shutdown timeout in seconds (default 5)
  -c                enable colored output (default true)
  -no-timestamp     disable timestamps
```

**Examples:**

```sh
# Start all processes
overseer start

# Start only the web process
overseer start web

# Run two web instances and one worker
overseer start -m web=2,worker=1

# Use a custom Procfile and env file
overseer start -f Procfile.staging -e .env.staging

# Disable color and timestamps (useful for log aggregators)
overseer start -c=false -no-timestamp
```

### `overseer run`

Run a single command (or a named Procfile entry) with the environment loaded from the `.env` file.

```sh
overseer run [flags] COMMAND

  -f string   Procfile path (default "Procfile")
  -e string   .env file path (default ".env")
```

**Examples:**

```sh
# Inspect the environment overseer would inject
overseer run env

# Run a Procfile entry by name
overseer run -e .env.test worker

# Run an arbitrary command with the loaded env
overseer run -e .env "bundle exec rake db:migrate"
```

## Procfile format

```
web:    python3 -m http.server $PORT
worker: sleep 100
clock:  python3 clock.py
```

- One process per line: `name: command`
- `$PORT` is injected automatically per instance
- Lines starting with `#` are ignored

## .env format

```sh
DATABASE_URL=postgres://localhost/myapp
SECRET_KEY='no escape processing here'
GREETING="Hello\nWorld"
```

- **Unquoted** — value used as-is
- **Single-quoted** — no escape processing
- **Double-quoted** — `\n` → newline, `\\` → `\`

## Formation & port assignment

The `-m` flag controls how many instances of each process type are spawned:

| Flag | Effect |
|---|---|
| *(none)* | one instance of every process (`all=1`) |
| `-m all=2` | two instances of every process |
| `-m web=2,worker=0` | two web instances, no workers |
| `overseer start web` | shorthand for `-m web=1` |

Ports are assigned as:

```
PORT = basePort + (processTypeIndex × 100) + instanceIndex
```

For example, with base port `5000` and `web=2,worker=2`:

| Instance | PORT |
|---|---|
| web.1 | 5000 |
| web.2 | 5001 |
| worker.1 | 5100 |
| worker.2 | 5101 |

Override the base with `-p PORT` or the `$PORT` environment variable.

## Signal handling

| Signal | Behavior |
|---|---|
| `SIGINT`, `SIGTERM`, `SIGHUP` | Graceful shutdown: `SIGTERM` all children, then `SIGKILL` after timeout |
| `SIGUSR1`, `SIGUSR2` | Forwarded directly to all child processes |

## Development

This project uses [mise](https://mise.jdx.dev) to manage the Go toolchain and common tasks.

```sh
# First-time setup
mise run setup

# Build
mise run build

# Run the dev server (uses Procfile.dev if present, falls back to Procfile)
mise run dev

# Test
mise run test

# Format
mise run fmt

# Lint (go vet + staticcheck)
mise run lint

# Full quality gate (fmt → lint → test)
mise run check

# Install binary to $GOBIN
mise run install

# Remove build artifacts
mise run clean
```

A [devcontainer](.devcontainer/devcontainer.json) is included for VS Code / GitHub Codespaces — it installs mise and runs `mise run setup` automatically on attach.

## Contributing

Contributions are welcome. Please read the [Code of Conduct](CODE_OF_CONDUCT.md) before participating.

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Run `mise run check` before pushing
4. Open a pull request

## Security

Please do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md) for the disclosure process.

## License

[GNU General Public License v3.0](LICENSE)

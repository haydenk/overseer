# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.3] - 2026-03-20

### Fixed

- Release workflow now correctly triggers when a tag is created

## [0.0.2] - 2026-03-20

### Added

- Cross-platform build support via platform-specific signal handling (`signal_unix.go`, `signal_windows.go`)

### Changed

- Set `CGO_ENABLED=0` for fully static binaries across all platforms

### Fixed

- Signal handling refactored out of `runner.go` into OS-specific files to enable Windows compatibility

## [0.0.1] - 2026-03-19

### Added

- `overseer start` — spawn all Procfile processes with prefixed, colored output and graceful shutdown
- `overseer check` — validate a Procfile and list detected process types
- `overseer run` — run a single command or named Procfile entry with the loaded environment
- Procfile parsing supporting `name: command` format; comments and blank lines ignored
- Formation support via `-m` flag (e.g. `web=2,worker=1`); `all=N` sets a default for unlisted types
- Automatic port assignment per instance: `PORT = basePort + (processTypeIndex × 100) + instanceIndex`
- `.env` file loading with unquoted, single-quoted, and double-quoted value support (double-quoted handles `\n` and `\\` escapes)
- Thread-safe output writer with 6 cycling ANSI colors, padded labels, and optional timestamps
- Signal forwarding: `SIGINT`/`SIGTERM`/`SIGHUP` trigger graceful shutdown; `SIGUSR1`/`SIGUSR2` forwarded to all child processes
- Graceful shutdown: `SIGTERM` to all children, then `SIGKILL` after configurable timeout (`-t` flag, default 5s)
- First-exit-wins shutdown — any process exiting triggers teardown of all others
- Devcontainer configuration for VS Code / GitHub Codespaces
- GitHub Actions workflows: CI, release (cross-platform binaries), GitFlow release automation, branch policy, stale, welcome, and labeler
- GitHub issue templates (bug report, feature request, question) and pull request template
- Zero external dependencies — pure Go standard library

[0.0.3]: https://github.com/haydenk/overseer/releases/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/haydenk/overseer/releases/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/haydenk/overseer/releases/tag/v0.0.1

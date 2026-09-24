# AGENTS.md

## Project Overview

`env-exec` is a Go CLI tool that injects environment variables from multiple sources before executing a command. Inspired by Kubernetes pod spec syntax. Sources: plain values, GCP Secret Manager, GitLab CI/CD variables.

## Directory Structure

```
cmd/env-exec/main.go        # CLI entrypoint, flag parsing, orchestration
internal/config/             # Config loading (reads .env-exec.yaml from CWD; overridden by ENV_EXEC_YAML env var)
internal/env/                # Prints export statements, sets process env vars, writes asFile files
internal/exec/               # Runs command via os/exec, forwards signals and exit code
internal/provider/           # Provider interface + implementations (plain, gcp, gitlab)
```

## Build & Test

```bash
go build ./...
go test ./...
go mod tidy
```

**Go version: 1.26** — keep in sync across `go.mod` and all GitHub workflows.

## Architecture

- **Provider pattern**: Each provider implements `Provide(config *config.RootConfig, envVars map[string]string) error`. Providers run in fixed order: `plain` → `gcp` → `gitlab`. All populate a shared `envVars` map. Note: `valueFrom` providers run after `plain`, so `valueFrom` values overwrite `value` entries for the same key.
- **Execution model**: Vars are injected into the current process via `os.Setenv` before `exec.Command` spawns the target command. `main.run()` returns the exit code instead of calling `os.Exit`, so deferred cleanup runs.
- **File-backed vars (`asFile`)**: `env.WriteFiles` writes the resolved value to a `0600` file in a private dir under `$XDG_RUNTIME_DIR` (else `os.TempDir()`) and replaces the value with the path. The dir is removed after the child exits. Rejected in export mode; masked as `<file>` in dry-run.
- **Signals**: `exec.Run` catches `SIGINT`/`SIGTERM`, forwards them to the child and waits for it. A `SIGINT` is not forwarded while env-exec is in the terminal's foreground process group, because Ctrl-C already reached the child and a second interrupt makes tools like terraform abort immediately. A child killed by a signal exits `128+n`.
- **Config loading**: Reads `.env-exec.yaml` from the current working directory. Overridable via `ENV_EXEC_YAML` env var only (no `--config` flag).

## Critical Known Issues (Do NOT Regress)

| Issue | Location | Details |
|-------|----------|---------|
| Silent failures | `internal/provider/gcp.go:48-51`, `internal/provider/gitlab.go:45-47` | Provider fetch failures log a warning and skip — downstream command fails with confusing "missing env var" |
| Wrong precedence warning | `internal/config/validation.go:40` | Warning text says "value takes precedence" but `valueFrom` actually wins (providers overwrite) |
| Dead code | `internal/config/validation.go:44-47` | Inner `Name == ""` check can never fire because `hasGCP` is `Name != ""` — dead code |
| No log control | `gcp.go:49`, `gitlab.go:40,46` | Providers use `log.Printf` for warnings — callers can't control format/destination/level; `plain.go` has no logging |

## Conventions

- No comments unless explicitly requested
- Tests in `*_test.go` alongside source files
- Follow existing code style and patterns

# AGENTS.md

## Project Overview

`env-exec` is a Go CLI tool that injects environment variables from multiple sources before executing a command. Inspired by Kubernetes pod spec syntax. Sources: plain values, GCP Secret Manager, GitLab CI/CD variables.

## Directory Structure

```
cmd/env-exec/main.go        # CLI entrypoint, flag parsing, orchestration
internal/config/             # Config loading (search up directory tree for .env-exec.yaml)
internal/env/                # Prints export statements, sets process env vars
internal/exec/               # Runs command via os/exec, forwards exit code
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

- **Provider pattern**: Each provider implements `Fetch(*config.RootConfig, map[string]string)`. Providers run in fixed order: `plain` → `gcp` → `gitlab`. All populate a shared `envVars` map.
- **Execution model**: Vars are injected into the current process via `os.Setenv` before `exec.Command` spawns the target command.
- **Config loading**: Searches current directory upward for `.env-exec.yaml`. Overridable via `ENV_EXEC_YAML` env var or `--config` flag.

## Critical Known Issues (Do NOT Regress)

| Issue | Location | Details |
|-------|----------|---------|
| Windows exit code | `internal/exec/exec.go:21` | Uses `syscall.WaitStatus` which is Unix-only. Fix: use `exitError.ExitCode()` |
| Shell injection | `internal/env/env.go:12` | Only escapes single quotes; `$`, `` ` ``, `\` in values are vulnerable |
| Hardcoded GitLab | `internal/provider/gitlab.go:66` | URL is `https://gitlab.com`; self-hosted unsupported |
| Silent failures | `gcp.go:37-41`, `gitlab.go:40-47` | Provider fetch failures log a warning and skip — downstream command fails with confusing "missing env var" |
| No log control | All providers | Use `log.Printf` for warnings — callers can't control format/destination/level |

## Conventions

- No comments unless explicitly requested
- Tests in `*_test.go` alongside source files
- Follow existing code style and patterns

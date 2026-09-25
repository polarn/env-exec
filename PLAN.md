# PLAN.md

## Do

### 3. GitLab client cleanup
- `gitlab.go:60`: `http.Client{}` has no timeout; set one (30s).
- `gitlab.go:14-21`: `GitlabVariable` is only used in `getGitlabVariable` and four of its fields are unused; make it local and drop them.
- `gitlab.go:69`: the ignored error on the error-body read loses only the body text (the status is already in the message). Fix in passing or leave.

## Optional, on demand

### 5. `-c` / `--config` flag
`ENV_EXEC_YAML=.env-exec.dev.yaml` already selects a per-environment config; a flag only makes it discoverable.

### 6. Configurable GitLab host
`gitlab.go:51` hardcodes `https://gitlab.com`. Add `defaults.gitlab.host` (or honour `CI_SERVER_URL`) when a self-hosted instance turns up.

### 7. Mask secrets in `--dry-run`
Dry-run prints every value in plaintext. Printing `<secret>` for `valueFrom` values by default, as `<file>` is printed for `asFile`, is simpler than a `--masked` flag, but it is a behaviour change: decide first.

### 8. `fileSuffix` for `asFile` entries
Files are named after the variable with no extension. No known tool checks it (GCP libraries and Snowflake do not); add an optional suffix when one does.

### 9. Document: Linux and macOS only
`internal/exec/exec.go` uses `golang.org/x/sys/unix`, Windows is not a goreleaser target and `GOOS=windows` does not build. The `export` output and `source <(env-exec)` are POSIX-shell only. One README line; do not port.

## Rejected

- **Validate GCP secret version as numeric or `latest`** — wrong: Secret Manager supports version aliases (`Secret.VersionAliases`), so `version: prod` is valid.
- **Provider registry / non-empty provider structs** — three providers whose order defines precedence; provider config already flows through `cfg.Defaults`.
- **Enable/disable providers** — a provider already runs only when an entry references it.
- **Templating in values** — the shell does it. If a concrete need appears, `${VAR}` via `os.Expand` over the resolved map, not Go templates.
- **`defer os.Unsetenv` in a test loop** (`env_test.go:101-103`) — correct as written; defers run at subtest end.
- **`GITLAB_TOKEN` visible in the process environment** — it is the user's own input, not a code change. The child inherits it on purpose.

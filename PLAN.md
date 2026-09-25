# PLAN.md

## Now

### 1. Release 1.1.1
- `main` has grpc 1.83.2 (PRs #14 and #18), which closes the three Dependabot alerts. The 1.1.0 tarballs still carry 1.82.1.
- env-exec is only a gRPC client, so the advisories (xDS server DoS, HTTP/2 OOM) never applied at runtime; the release is for scanners and package repos.

## Do

### 2. Config validation: one source per entry, unique valid names
- **File**: `internal/config/validation.go`
- Reject, instead of warn or accept:
  - both `value` and `valueFrom` (`:39-41`, warning text is also wrong: `valueFrom` wins, not `value`)
  - both `gcpSecretKeyRef` and `gitlabVariableKeyRef` (undocumented last-writer-wins today)
  - duplicate names (`:23-26`, warning, last wins)
  - names outside `[A-Za-z_][A-Za-z0-9_]*`. `os.Setenv` accepts `FOO BAR`, `FOO-BAR` and `1FOO` and the child sees them; only `=` is refused. The name is also the `asFile` filename.
- Remove the dead check at `:44-47` (`hasGCP` already means `Name != ""`).
- Ship together with #3: today a `value` beside a failing `valueFrom` acts as a silent fallback, and both changes remove that.

### 3. Fetch failures are fatal
- **Files**: `internal/provider/gcp.go:61-81`, `internal/provider/gitlab.go:39-48`
- Missing project, access failure and empty payload log a warning and `continue`; the command then fails later with a confusing missing-variable error. Return an error instead.
- Wrap the GCP provider in `context.WithTimeout` (`gcp.go:44`). gax already gives `AccessSecretVersion` a 60s timeout with retries (2s → 60s backoff), so it cannot hang forever, but the worst case is minutes.
- Replace `hasGCPSecrets` / `hasGitlabVariables` (`gcp.go:89-96`, `gitlab.go:56-63`) with `slices.ContainsFunc`.
- The `gcp_test.go` "skips only that secret" cases become error cases.

### 4. GitLab client cleanup
- `gitlab.go:75`: `http.Client{}` has no timeout; set one (30s).
- `gitlab.go:14-21`: `GitlabVariable` is only used in `getGitlabVariable` and four of its fields are unused; make it local and drop them.
- `gitlab.go:84`: the ignored error on the error-body read loses only the body text (the status is already in the message). Fix in passing or leave.

### 5. Test hygiene
- `internal/env/env_test.go:11-24`: `captureStdout` must `defer` the `os.Stdout` restore, or a panic breaks every later test. The copy in `cmd/env-exec/main_test.go` already does.

## Optional, on demand

### 6. `-c` / `--config` flag
`ENV_EXEC_YAML=.env-exec.dev.yaml` already selects a per-environment config; a flag only makes it discoverable.

### 7. Configurable GitLab host
`gitlab.go:66` hardcodes `https://gitlab.com`. Add `defaults.gitlab.host` (or honour `CI_SERVER_URL`) when a self-hosted instance turns up.

### 8. Mask secrets in `--dry-run`
Dry-run prints every value in plaintext. Printing `<secret>` for `valueFrom` values by default, as `<file>` is printed for `asFile`, is simpler than a `--masked` flag, but it is a behaviour change: decide first.

### 9. `fileSuffix` for `asFile` entries
Files are named after the variable with no extension. No known tool checks it (GCP libraries and Snowflake do not); add an optional suffix when one does.

### 10. Document: Linux and macOS only
`internal/exec/exec.go` uses `golang.org/x/sys/unix`, Windows is not a goreleaser target and `GOOS=windows` does not build. The `export` output and `source <(env-exec)` are POSIX-shell only. One README line; do not port.

## Rejected

- **Validate GCP secret version as numeric or `latest`** — wrong: Secret Manager supports version aliases (`Secret.VersionAliases`), so `version: prod` is valid.
- **Provider registry / non-empty provider structs** — three providers whose order defines precedence; provider config already flows through `cfg.Defaults`.
- **Enable/disable providers** — a provider already runs only when an entry references it.
- **Templating in values** — the shell does it. If a concrete need appears, `${VAR}` via `os.Expand` over the resolved map, not Go templates.
- **`defer os.Unsetenv` in a test loop** (`env_test.go:96-98`) — correct as written; defers run at subtest end.
- **`GITLAB_TOKEN` visible in the process environment** — it is the user's own input, not a code change. The child inherits it on purpose.

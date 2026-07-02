# PLAN.md

## Bugs

### 1. Non-cross-platform exit code extraction
- **File**: `internal/exec/exec.go:21-22`
- **Issue**: `syscall.WaitStatus` is Unix-only. On Windows, `Sys()` returns a different type and the type assertion fails, causing the exit code to be silently lost.
- **Fix**: Use `exitError.ExitCode()` — it's cross-platform and available on `*exec.ExitError`.

### 2. HTTP client with no timeout
- **File**: `internal/provider/gitlab.go:75`
- **Issue**: `http.Client{}` has zero timeout. If the GitLab API is slow or unresponsive, the request hangs indefinitely.
- **Fix**: Add a `Timeout` (e.g., 30s).

### 3. No context deadline on GCP calls
- **File**: `internal/provider/gcp.go:19`
- **Issue**: `context.Background()` with no deadline — hung GCP API calls block forever.
- **Fix**: Use `context.WithTimeout` or pass a context through the provider.

### 4. Silent skip on unresolvable GCP secrets
- **File**: `internal/provider/gcp.go:37-41`
- **Issue**: Fetch failure logs a warning and `continue`s. The user's command fails with a confusing "missing env var" error and the real cause may be scrolled off screen.
- **Fix**: Make fetch failures fatal (return error), or at minimum buffer all warnings and print them once before execution.

### 5. Same silent-skip for GitLab variables
- **File**: `internal/provider/gitlab.go:40-47`
- **Issue**: Identical pattern to Bug 4.
- **Fix**: Same fix as Bug 4.

### 6. `captureStdout` does not restore `os.Stdout` on panic
- **File**: `internal/env/env_test.go:12-23`
- **Issue**: If the test function panics, `os.Stdout` is never restored, causing cascading failures in subsequent tests.
- **Fix**: Use `defer os.Stdout = old` after saving the original.

### 7. Duplicate env var names only produce a warning
- **File**: `internal/config/validation.go:23-26`
- **Issue**: Duplicate names silently overwrite the first value ("last wins"). Missing names are a hard error — this is inconsistent.
- **Fix**: Either make duplicates an error or document the "last wins" behavior.

### 8. Having both `value` and `valueFrom` is only a warning
- **File**: `internal/config/validation.go:40`
- **Issue**: A config entry can specify both sources simultaneously; `value` silently takes precedence but the warning may be missed.
- **Fix**: Make this an error, or clearly document that `value` always overrides `valueFrom`.

## Security

### 1. Shell injection vulnerability
- **File**: `internal/env/env.go:12`
- **Issue**: Only escapes single quotes. Values containing `$`, `` ` ``, or `\` are vulnerable to shell interpretation when used with `source <(env-exec)`.
- **Fix**: Escape `$`, `` ` ``, and `\` as well, or document that `source <(env-exec)` should only be used with trusted config files.

### 2. Hardcoded GitLab instance URL
- **File**: `internal/provider/gitlab.go:66`
- **Issue**: `https://gitlab.com/api/v4/...` — self-hosted GitLab instances unsupported. A user could accidentally authenticate with a token that has no access to gitlab.com.
- **Fix**: Add a `gitlabHost` config field (or env var) that defaults to `https://gitlab.com`.

### 3. Deprecated GitLab authorization header
- **File**: `internal/provider/gitlab.go:73`
- **Issue**: `PRIVATE-TOKEN` header is deprecated by GitLab. Modern approach is `Authorization: Bearer <token>`.
- **Fix**: Switch to `Authorization: Bearer <token>`.

### 4. Token visible in process environment
- **File**: `internal/provider/gitlab.go:29`
- **Issue**: Token lives in process env — visible via `/proc/<pid>/environ` to any process with sufficient permissions.
- **Fix**: Document that the token should have minimal required permissions.

### 5. No environment variable name validation
- **File**: `internal/config/validation.go`
- **Issue**: No validation on env var name format. POSIX requires `[A-Za-z_][A-Za-z0-9_]*`. Names with spaces, hyphens, or leading digits are silently accepted and will cause the spawned command to fail.
- **Fix**: Validate names in `Validate()`.

### 6. No mutual exclusion check on `valueFrom` sources
- **File**: `internal/config/validation.go`
- **Issue**: A `valueFrom` block can specify both `gcpSecretKeyRef` and `gitlabVariableKeyRef` simultaneously. The behavior depends on provider execution order and is undefined.
- **Fix**: Add validation that rejects `valueFrom` entries with more than one source set.

### 7. Swallowed HTTP error body read
- **File**: `internal/provider/gitlab.go:83-85`
- **Issue**: On HTTP error, `bodyBytes, _ := io.ReadAll(resp.Body)` ignores the read error. An I/O failure during error body read is silently lost.
- **Fix**: Check the error or use `defer resp.Body.Close()` at the top of the function.

## Missing Validation

### 1. `valueFrom` mutual exclusion
No validation that `valueFrom` sources are mutually exclusive. A user could specify both GCP and GitLab refs simultaneously.

### 2. GCP secret version format
No client-side validation of the `version` field. Non-numeric strings for numeric versions produce opaque API errors.

### 3. GCP secret name emptiness
No client-side check that `gcpSecretKeyRef.name` is non-empty before calling the API — the provider checks it at runtime, but this should be config-time validation.

## Code Quality

### 1. Duplicate `has*()` functions
- **Files**: `internal/provider/gcp.go:59-66` and `gitlab.go:56-63`
- **Issue**: Two structurally identical functions iterating over `cfg.Env` to check if a specific nested field is non-empty.
- **Fix**: Factor into a generic helper: `func hasValueSource(cfg, check func(EnvConfig) bool) bool`.

### 2. Empty struct providers
- **File**: `internal/provider/provider.go:12-24`
- **Issue**: Providers are empty structs serving only as type identifiers. This makes adding provider configuration impossible without changing the interface.
- **Fix**: Change `Fetch` to accept a provider-specific config, or use `interface{}` / options pattern.

### 3. Hardcoded provider registry
- **File**: `internal/provider/provider.go:27-33`
- **Issue**: Adding a new provider requires modifying `AllProviders()`.
- **Fix**: Use a registry pattern (`var providers = make(map[string]Provider)`) with `Register(name, Provider)` for extensibility.

### 4. Package-level struct for single use
- **File**: `internal/provider/gitlab.go:14-21`
- **Issue**: `GitlabVariable` struct is only used inside `getGitlabVariable()`. Should be a local type to reduce package-level API surface.

### 5. Fragile defer in test loop
- **File**: `internal/env/env_test.go:95-98`
- **Issue**: `defer os.Unsetenv(k)` inside a loop — fragile, breaks with `t.Parallel()`.

### 6. Non-idiomatic HTTP request construction
- **File**: `internal/provider/gitlab.go:68`
- **Issue**: `http.NewRequest("GET", url, nil)` — modern form is `http.NewRequestWithContext(context.Background(), ...)`.

## Design Improvements

### 1. No way to enable/disable providers
All providers always run. No configuration to skip a provider (e.g., skip GCP when running locally during development).

### 2. `--dry-run` leaks secrets
All values including secrets are printed in plaintext during dry-run. A `--masked` option (similar to GitLab CI masked variables) would prevent accidental secret exposure.

### 3. No environment-specific configs
No support for dev/staging/prod config selection (e.g., `.env-exec.dev.yaml`).

### 4. No templating
No support for referencing other variables in values (e.g., `{{ .Env.DB_HOST }}`).

## Portability

### 1. `syscall.WaitStatus` is Unix-only
- **File**: `internal/exec/exec.go:21`
- **Fix**: Use `exitError.ExitCode()`.

### 2. POSIX shell output format
- **File**: `internal/env/env.go:12-13`
- **Issue**: Outputs `export VAR='value'` syntax — not compatible with Windows cmd/PowerShell. The `source <(env-exec)` example uses bash-specific process substitution.
- **Fix**: Document the limitation, or add a `--shell` flag for output format selection.

## Recommended Priority Order

| Priority | Issue | Effort | Impact |
|----------|-------|--------|--------|
| 1 | Bug 1: Windows exit code + Security 1: Shell injection | Low | High — core functionality |
| 2 | Bug 2: HTTP timeout + Bug 3: Context timeout | Low | High — reliability |
| 3 | Security 2: GitLab host config | Low | Medium — common pain point |
| 4 | Bug 4-5: Silent skip on fetch failure | Medium | Medium — UX |
| 5 | Design 1: Provider enable/disable | Low | Medium — developer UX |
| 6 | Quality 1: Dedup has*() functions | Low | Low — cleanup |
| 7 | Quality 3: Provider registry | Medium | Medium — extensibility |

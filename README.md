# env-exec

A CLI tool that injects environment variables from various sources before executing a command. Useful for running commands that need secrets, such as `terraform plan` with cloud credentials.

## Installation

**Arch Linux (AUR):**
```bash
yay -S env-exec-bin
```

**macOS (Homebrew):**
```bash
brew install polarn/tap/env-exec
```

**Debian/Ubuntu:**
```bash
# Download from releases or use the apt repository
sudo dpkg -i env-exec_*.deb
```

## Usage

```bash
env-exec terraform plan
```

This runs `terraform plan` with environment variables injected from `.env-exec.yaml` in the current directory.

### CLI Flags

```
-h, --help      Show help
-v, --version   Show version
-n, --dry-run   Print environment variables without executing command
```

### Export to Shell

Run without arguments to output `export` statements:

```bash
source <(env-exec)
```

### Custom Config Path

```bash
ENV_EXEC_YAML=/path/to/config.yaml env-exec terraform plan
```

## Configuration

Create a `.env-exec.yaml` file:

```yaml
defaults:
  gcp:
    project: "my-gcp-project"

env:
  # Plain values
  - name: MY_VAR
    value: "static-value"

  # GCP Secret Manager
  - name: DB_PASSWORD
    valueFrom:
      gcpSecretKeyRef:
        name: database-password
        version: latest  # optional, defaults to "latest"
        project: other-project  # optional, overrides default

  # GitLab CI/CD Variables
  - name: DEPLOY_TOKEN
    valueFrom:
      gitlabVariableKeyRef:
        project: "12345"
        key: deploy-token

  # Written to a file; the variable holds the file's path
  - name: GOOGLE_APPLICATION_CREDENTIALS
    asFile: true
    valueFrom:
      gcpSecretKeyRef:
        name: service-account-key
```

The syntax is inspired by Kubernetes pod specs.

### File-backed Variables

Some tools take a *path* to a secret rather than the secret itself (a private key, a service account key, a kubeconfig). Set `asFile: true` on any entry, whatever its source, and env-exec writes the value to a file and sets the variable to that file's path.

- The file is `0600`, in a private `0700` directory under `$XDG_RUNTIME_DIR` (usually a per-user tmpfs), or the system temp directory when that is unset. It is named after the variable.
- The directory is removed when the command exits, including after Ctrl-C or `SIGTERM`. Only `SIGKILL` or a crash leaves it behind.
- Export mode (`source <(env-exec)`) rejects `asFile` entries, because no command runs whose exit could trigger the cleanup.
- `--dry-run` prints `<file>` in place of the path and writes nothing.

## Providers

### Plain Values

Static key-value pairs defined directly in the config.

### GCP Secret Manager

Fetches secrets from Google Cloud Secret Manager. Requires GCP credentials (e.g., `gcloud auth application-default login`).

- `name` - Secret name (required)
- `version` - Secret version (optional, defaults to `latest`)
- `project` - GCP project (optional if `defaults.gcp.project` is set)

### GitLab CI/CD Variables

Fetches project variables from GitLab. Requires `GITLAB_TOKEN` environment variable with API access.

```bash
export GITLAB_TOKEN=glpat-xxxx
env-exec terraform plan
```

- `project` - GitLab project ID (required)
- `key` - Variable key (required)

## License

Apache-2.0

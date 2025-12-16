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

  # GitHub Actions Variables
  - name: MY_SECRET
    valueFrom:
      githubVariableKeyRef:
        repo: "owner/repo"  # optional if defaults.github.repo is set
        name: my-variable
```

The syntax is inspired by Kubernetes pod specs.

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

### GitHub Actions Variables

Fetches repository variables from GitHub Actions. Requires `GITHUB_TOKEN` environment variable with `repo` scope.

```bash
export GITHUB_TOKEN=ghp_xxxx
env-exec terraform plan
```

- `repo` - GitHub repository in `owner/repo` format (optional if `defaults.github.repo` is set)
- `name` - Variable name (required)

You can set a default repo in your config:

```yaml
defaults:
  github:
    repo: "owner/repo"
```

## License

Apache-2.0

# env-exec

A simple tool that you prefix your important command that needs some environment variables injected to work, for example when running Terraform and you need Azure secrets, etc to be able to `terraform plan`.

## Installation

For Arch Linux it's available in the AUR, so install with for example yay:

```bash
yay -S env-exec-bin
```

For Mac you can use brew:

```bash
brew install polarn/tap/env-exec
```

## Usage

```bash
env-exec terraform plan
```

would run `terraform plan`, but before running, `env-exec` will read a file called `.env-exec.yaml` in the same folder which will define environment variables. These variables can be static or they can be fetched from GCP Secrets Manager, which is actually the main use case for the tool.

The configuration file can look like this:

```yaml
defaults:
  gcp:
    project: "my-gcp-project"
env:
  - name: NON_SECRET_VARIABLE
    value: "test1"
  - name: ANOTHER_NON_SECRET_VAR
    value: "test2"
  - name: VARIABLE_FROM_GCP_SECRET_MANAGER
    valueFrom:
      gcpSecretKeyRef:
        name: secret-in-gcp-secrets-manager
  - name: VARIABLE_FROM_GITLAB
    valueFrom:
      gitlabVariableKeyRef:
        project: 12345
        key: key-from-gitlab-variables
```

The GCP project can be defined as a default project, used for all secrets, but you can also define `project` inside the `gcpSecretKeyRef` as well.

The YAML syntax is inspired by the Kubernetes pod spec.

If you run the tool without any arguments it will output the environment variables with `export` in front of them. The idea then is you can use it to set variables for your local shell.

```bash
source <(env-exec)
```

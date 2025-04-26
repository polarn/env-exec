# env-exec

A simple tool that you prefix your important command that needs some environment variables injected to work, for example when running Terraform and you need Azure secrets, etc to be able to `terraform plan`.

So for example:

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
  - name: ARM_TENANT_ID
    value: "<tenant-id>"
  - name: ARM_SUBSCRIPTION_ID
    value: "<subscription-id>"
  - name: ARM_CLIENT_ID
    valueFrom:
      gcpSecretKeyRef:
        name: azure-dev-arm-client-id
  - name: ARM_CLIENT_SECRET
    valueFrom:
      gcpSecretKeyRef:
        name: azure-dev-arm-client-secret
```

The GCP project can be defined as a default project, used for all secrets, but you can also define `project` inside the `gcpSecretKeyRef` as well.

The YAML syntax is taken from the Kubernetes pod spec.

If you run the tool without any arguments it will output the environment variables with `export` in front of them. The idea then is you can use it to set variables for your local shell.

```bash
source <(env-exec)
```

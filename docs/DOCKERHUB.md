# Vault Raft Backup & Restore Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Chapsvision-dev/vault-raft-backup-restore)](https://go.dev/)

A lightweight Go-based automation operator for **backup** and **restore** of HashiCorp Vault Raft snapshots to cloud storage.

## Quick Start

### Backup

```bash
docker run --rm \
  -e VAULT_ADDR=https://vault.example.com:8200 \
  -e VAULT_TOKEN=your-token \
  -e AZURE_STORAGE_ACCOUNT=mystorageaccount \
  -e AZURE_STORAGE_CONTAINER=vault-backups \
  -e AZURE_SAS_TOKEN=your-sas-token \
  chapsvision/vault-raft-backup-operator:latest \
  backup --provider azure
```

### Restore

```bash
docker run --rm \
  -e VAULT_ADDR=https://vault.example.com:8200 \
  -e VAULT_TOKEN=your-token \
  -e AZURE_STORAGE_ACCOUNT=mystorageaccount \
  -e AZURE_STORAGE_CONTAINER=vault-backups \
  -e AZURE_SAS_TOKEN=your-sas-token \
  chapsvision/vault-raft-backup-operator:latest \
  restore --provider azure --snapshot-name snapshot-20250101-120000.snap
```

## Supported Storage Providers

- **Azure Blob Storage** (Service Principal, Managed Identity, or SAS token)
- More providers coming soon (AWS S3, GCS, MinIO)

## Authentication Methods

### Vault Authentication
- **Static Token** (dev/testing)
- **Kubernetes ServiceAccount** (production)

### Azure Authentication
- **Service Principal** (credentials-based)
- **Managed Identity** (AKS Workload Identity - recommended)
- **SAS Token** (time-limited access)

## Environment Variables

### Vault Configuration
- `VAULT_ADDR` - Vault server address (required)
- `VAULT_TOKEN` - Vault token (required unless using K8s auth)
- `VAULT_SKIP_VERIFY` - Skip TLS verification (default: false)

### Azure Blob Storage
- `AZURE_STORAGE_ACCOUNT` - Storage account name (required)
- `AZURE_STORAGE_CONTAINER` - Container name (required)
- `AZURE_TENANT_ID` - Azure AD tenant ID (for Service Principal)
- `AZURE_CLIENT_ID` - Service Principal client ID (for Service Principal)
- `AZURE_CLIENT_SECRET` - Service Principal secret (for Service Principal)
- `AZURE_SAS_TOKEN` - SAS token (alternative to Service Principal)

## Production Deployment

For production use in Kubernetes with best practices:

**📦 Source Code & Documentation:**
https://github.com/Chapsvision-dev/vault-raft-backup-restore

The repository includes:
- **Kubernetes manifests** with CronJob examples
- **Terraform modules** for infrastructure automation
- **Docker Compose** examples for local testing
- **Complete documentation** with security best practices

### Recommended Approach (Kubernetes + Workload Identity)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: vault-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            azure.workload.identity/use: "true"
        spec:
          serviceAccountName: vault-backup
          containers:
          - name: backup
            image: chapsvision/vault-raft-backup-operator:latest
            args: ["backup", "--provider", "azure"]
            env:
            - name: VAULT_ADDR
              value: "http://vault:8200"
            - name: AZURE_STORAGE_ACCOUNT
              value: "mystorageaccount"
            - name: AZURE_STORAGE_CONTAINER
              value: "vault-backups"
          restartPolicy: OnFailure
```

See the [Kubernetes examples](https://github.com/Chapsvision-dev/vault-raft-backup-restore/tree/main/examples/kubernetes) for complete setup guides.

## Security Best Practices

- ✅ Never commit Vault tokens or credentials to version control
- ✅ Use Kubernetes ServiceAccount authentication in production
- ✅ Use Workload Identity (passwordless) instead of Service Principals when possible
- ✅ Enable TLS verification for Vault (`VAULT_SKIP_VERIFY=false`)
- ✅ Encrypt snapshots at rest using cloud provider encryption
- ✅ Restrict access to snapshot storage with minimal permissions
- ✅ Set appropriate expiration times on SAS tokens

**⚠️ Important:** Vault snapshots contain sensitive data including all secrets, encryption keys, and audit logs. Always treat snapshots as highly sensitive data.

## Documentation & Support

- 📖 **Full Documentation:** https://github.com/Chapsvision-dev/vault-raft-backup-restore
- 🐛 **Issues:** https://github.com/Chapsvision-dev/vault-raft-backup-restore/issues
- 💬 **Discussions:** https://github.com/Chapsvision-dev/vault-raft-backup-restore/discussions
- 📋 **Examples:** https://github.com/Chapsvision-dev/vault-raft-backup-restore/tree/main/examples

## Features

- ✅ HashiCorp Vault Raft snapshot support
- ✅ Multiple authentication methods (Token, K8s ServiceAccount)
- ✅ Cloud storage providers (Azure Blob, more coming)
- ✅ Kubernetes CronJob ready
- ✅ Terraform-ready with IaC modules
- ✅ Security-first design
- ✅ Production-tested

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](https://github.com/Chapsvision-dev/vault-raft-backup-restore/blob/main/LICENSE.md) for details.

---

**Maintained by:** [Chapsvision](https://github.com/Chapsvision-dev)

# Deployment Examples

This directory contains ready-to-use deployment examples for the Vault Raft Backup & Restore Operator.

## 📁 Directory Structure

```
examples/
├── docker-compose/           # Docker Compose examples (local dev/testing)
│   ├── backup/              # Backup example with Azure
│   └── restore/             # Restore example with Azure
├── kubernetes/              # Kubernetes manifests
│   └── azure/               # Azure provider
│       ├── service-principal/    # With Service Principal auth
│       └── workload-identity/    # With Workload Identity auth (recommended)
├── terraform/               # Terraform modules
│   └── azure/               # Azure provider
│       ├── service-principal/    # With Service Principal auth
│       └── workload-identity/    # With Workload Identity auth (recommended)
└── vault-policy/            # Vault policy for Raft operations
```

## 🚀 Quick Start by Use Case

### Local Development & Testing

Use Docker Compose for quick local testing:
- [Docker Compose Backup Example](docker-compose/backup/)
- [Docker Compose Restore Example](docker-compose/restore/)

### Kubernetes Deployments

Deploy as CronJob for scheduled backups or one-time Job for restores.

**Choose your Azure authentication method:**
- 🎯 [Workload Identity (Recommended)](kubernetes/azure/workload-identity/) - Passwordless, secret-free auth for AKS
- 🔐 [Service Principal](kubernetes/azure/service-principal/) - Traditional credential-based auth

[View all Kubernetes examples →](kubernetes/)

### Production with Terraform

Full infrastructure-as-code setup with Azure integration.

**Choose your Azure authentication method:**
- 🎯 [Workload Identity (Recommended)](terraform/azure/workload-identity/) - Managed Identity with federated credentials
- 🔐 [Service Principal](terraform/azure/service-principal/) - Service Principal with auto-rotation

[View all Terraform modules →](terraform/)

## 📋 Prerequisites

Before deploying, ensure you have:

1. **Vault Configuration**
   - Vault cluster with Raft storage backend
   - Authentication method configured (Token or Kubernetes)
   - Required policy attached (see [Vault Policy](#vault-policy))

2. **Azure Resources** (for Azure provider)
   - Azure Storage Account
   - Storage Container for snapshots
   - **For Service Principal**: Azure AD Service Principal with Storage Blob Data Contributor role
   - **For Workload Identity**: AKS cluster with OIDC + Workload Identity enabled, Managed Identity

3. **Kubernetes Resources** (for K8s deployments)
   - ServiceAccount with proper annotations (Workload Identity) or access to credentials (Service Principal)
   - Vault Kubernetes auth backend role

## 🔐 Vault Policy

The ServiceAccount or Token must have the following Vault policy:

\`\`\`hcl
# GET /v1/sys/storage/raft/snapshot
path "sys/storage/raft/snapshot" {
  capabilities = ["read"]
}

# POST /v1/sys/storage/raft/snapshot (restore)
path "sys/storage/raft/snapshot" {
  capabilities = ["update"]
}

# POST /v1/sys/storage/raft/snapshot-force (forced restore)
path "sys/storage/raft/snapshot-force" {
  capabilities = ["update"]
}
\`\`\`

A ready-to-use policy file is available at [vault-policy/raft-backup-restore.hcl](vault-policy/raft-backup-restore.hcl).

Apply with:

\`\`\`bash
vault policy write raft-backup-restore vault-policy/raft-backup-restore.hcl
\`\`\`

## 🔧 Configuration Reference

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `VAULT_ADDR` | Yes | Vault server address | `http://vault:8200` |
| `VAULT_AUTH_METHOD` | Yes | Auth method (`token` or `kubernetes`) | `kubernetes` |
| `VAULT_TOKEN` | If token auth | Static token | `s.xxxxx` |
| `VAULT_K8S_ROLE` | If k8s auth | Kubernetes auth role name | `vault-raft-operator` |
| `VAULT_K8S_MOUNT` | If k8s auth | Auth backend mount path | `kubernetes` |
| `BACKUP_PROVIDER` | Yes | Storage provider | `azure` |
| `AZURE_STORAGE_ACCOUNT` | Yes | Azure storage account name | `mystorageaccount` |
| `AZURE_STORAGE_CONTAINER` | Yes | Container name | `vault-snapshots` |
| `AZURE_CLIENT_ID` | If SP auth | Service Principal client ID | `xxx-xxx` |
| `AZURE_TENANT_ID` | If SP auth | Azure tenant ID | `xxx-xxx` |
| `AZURE_CLIENT_SECRET` | If SP auth | Service Principal secret | `xxx` |
| `BACKUP_SOURCE` | Backup | Local snapshot path | `/data/snapshot.snap` |
| `BACKUP_TARGET` | Backup | Remote blob prefix | `snapshots` |
| `RESTORE_SOURCE` | Restore | Remote blob key | `snapshots/2025-01-01.snap` |
| `RESTORE_TARGET` | Restore | Local restore path | `/data/restored.snap` |
| `LOG_LEVEL` | No | Log level | `info` (default: `info`) |
| `LOG_FORMAT` | No | Log format | `json` (default: `json`) |

## 📚 Detailed Examples

- [Docker Compose Examples](docker-compose/)
- [Kubernetes Examples](kubernetes/)
- [Terraform Module](terraform/)

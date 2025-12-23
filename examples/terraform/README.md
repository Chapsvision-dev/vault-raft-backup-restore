# Terraform Modules

Infrastructure-as-code modules for deploying Vault Raft Backup & Restore Operator using Terraform.

## Available Modules

### Azure Provider

Choose your authentication method:

#### 🎯 [Workload Identity](azure/workload-identity/) (Recommended)
Complete module using Azure AD Workload Identity for passwordless authentication.

**Best for:**
- AKS clusters with Workload Identity enabled
- Production environments requiring zero-trust security
- Eliminating credential management overhead
- Organizations prioritizing secret-free architecture

**Features:**
- ✅ No secrets to manage
- ✅ Auto-rotated short-lived tokens
- ✅ Federated identity credentials
- ✅ Better audit trail via Azure AD sign-in logs

[View Workload Identity Module →](azure/workload-identity/)

---

#### 🔐 [Service Principal](azure/service-principal/)
Complete module using Azure Service Principal credentials.

**Best for:**
- Non-AKS Kubernetes clusters
- Quick setup without additional Azure configuration
- Environments where Workload Identity is not available
- Multi-cloud deployments

**Features:**
- ✅ Works on any Kubernetes cluster
- ✅ Automatic annual credential rotation
- ✅ Simple setup process
- ✅ Credentials stored in Kubernetes Secrets

[View Service Principal Module →](azure/service-principal/)

---

## Module Comparison

| Feature | Workload Identity | Service Principal |
|---------|-------------------|-------------------|
| **Authentication** | Managed Identity + Federated Credential | Service Principal + Secret |
| **Secrets** | None | Kubernetes Secret |
| **Token Lifetime** | Short-lived (auto-refresh) | 1 year (auto-rotate) |
| **AKS Required** | Yes (with OIDC + Workload Identity) | No |
| **Security** | Zero-trust, no secrets | Credential-based |
| **Audit Trail** | Azure AD sign-in logs | Storage access logs |
| **Setup Complexity** | Medium (requires AKS features) | Low |
| **Maintenance** | None (fully automated) | Low (annual rotation) |

## Prerequisites

### Common Prerequisites

1. **Terraform**
   - Terraform >= 1.5
   - Azure CLI configured with appropriate permissions

2. **Azure**
   - Active Azure subscription
   - Existing Storage Account
   - Resource Group

3. **Kubernetes**
   - Kubernetes cluster (1.23+)
   - kubectl with admin access
   - Cluster credentials (CA cert, token)

4. **Vault**
   - Vault cluster with Raft storage backend
   - Vault admin token
   - Network connectivity from Kubernetes to Vault

### Workload Identity Additional Requirements

- AKS cluster with:
  - OIDC issuer enabled
  - Workload Identity feature enabled

### Service Principal Additional Requirements

- Permissions to create Azure AD Service Principals

## Quick Start

### 1. Choose Your Module

```bash
# For Workload Identity (recommended for AKS)
cd azure/workload-identity/

# For Service Principal
cd azure/service-principal/
```

### 2. Configure Variables

```bash
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
```

### 3. Deploy

```bash
terraform init
terraform plan
terraform apply
```

### 4. Verify

```bash
# Check CronJob
kubectl get cronjob -n vault-hashicorp vault-raft-backup

# Trigger manual backup
terraform output -raw manual_backup_command | bash

# View logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup --follow
```

## What These Modules Create

### Azure Resources
- **Workload Identity**: Managed Identity + Federated Credential
- **Service Principal**: Azure AD Application + Service Principal + Password
- Storage Container for snapshots
- RBAC role assignment (Storage Blob Data Contributor)

### Kubernetes Resources
- Namespace (`vault-hashicorp`)
- ServiceAccount (with Workload Identity annotations or regular)
- **Service Principal only**: Kubernetes Secret with credentials
- CronJob for scheduled backups

### Vault Resources
- Kubernetes auth backend
- Vault policy for Raft snapshot operations
- Kubernetes auth backend role

## Module Outputs

Both modules provide useful outputs:

```bash
# View all outputs
terraform output

# Specific outputs
terraform output storage_account_name
terraform output cronjob_name
terraform output manual_backup_command

# Authentication-specific
terraform output managed_identity_client_id     # Workload Identity
terraform output service_principal_client_id    # Service Principal
```

## Migration Between Modules

### From Service Principal to Workload Identity

1. **Prerequisites:**
   - Enable Workload Identity on AKS cluster
   - Enable OIDC issuer on AKS cluster

2. **Deploy Workload Identity module:**
   ```bash
   cd azure/workload-identity/
   terraform init
   terraform apply
   ```

3. **Verify backup works:**
   ```bash
   kubectl create job -n vault-hashicorp \
     --from=cronjob/vault-raft-backup \
     test-workload-identity-$(date +%s)
   ```

4. **Destroy Service Principal resources:**
   ```bash
   cd ../service-principal/
   terraform destroy
   ```

## Remote State Backend

For production use, configure remote state storage:

```hcl
# Add to main.tf
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate"
    container_name       = "tfstate"
    key                  = "vault-raft-operator.tfstate"
  }
}
```

Initialize with backend:
```bash
terraform init -backend-config="storage_account_name=tfstate"
```

## Security Best Practices

1. **State Files**
   - Use remote backend with encryption
   - Restrict access to state files
   - Enable versioning

2. **Vault Token**
   - Use limited-TTL token for setup
   - Store securely (Azure Key Vault, HashiCorp Vault)
   - Revoke after infrastructure deployment

3. **Kubernetes Token**
   - Use dedicated ServiceAccount for Terraform
   - Limit token duration
   - Rotate periodically

4. **Azure Credentials**
   - Use Azure CLI authentication when possible
   - Avoid hardcoding credentials
   - Use Managed Identity for CI/CD pipelines

## Backup and Restore

### Automated Backups

Both modules create a CronJob that automatically backs up Vault snapshots to Azure Storage:

- Default schedule: Daily at 2 AM UTC
- Configurable via `backup_schedule` variable
- History: 3 successful jobs, 3 failed jobs retained

### Manual Backup

```bash
# Get command from Terraform output
terraform output -raw manual_backup_command | bash

# Or manually create job
kubectl create job -n vault-hashicorp \
  --from=cronjob/vault-raft-backup \
  vault-raft-backup-manual-$(date +%s)
```

### Restore Operations

Restore requires manual Job creation:

1. List snapshots in Azure Storage
2. Create restore Job manifest with snapshot blob key
3. Seal Vault
4. Apply Job and monitor
5. Unseal Vault

See individual module READMEs for detailed restore procedures.

## Troubleshooting

### Common Issues

**Issue: Terraform provider authentication fails**
```bash
# Login to Azure
az login

# Set subscription
az account set --subscription <subscription-id>

# Verify credentials
az account show
```

**Issue: Kubernetes provider connection fails**
```bash
# Verify kubectl works
kubectl cluster-info

# Check token validity
kubectl auth can-i get pods --all-namespaces
```

**Issue: Vault provider authentication fails**
```bash
# Test Vault connectivity
vault status

# Check token
vault token lookup
```

### Module-Specific Troubleshooting

- [Workload Identity Troubleshooting](azure/workload-identity/README.md#troubleshooting)
- [Service Principal Troubleshooting](azure/service-principal/README.md#troubleshooting)

## Documentation

- [Workload Identity Module Documentation](azure/workload-identity/README.md)
- [Service Principal Module Documentation](azure/service-principal/README.md)
- [Main Project README](../../README.md)
- [Kubernetes Examples](../kubernetes/)
- [Docker Compose Examples](../docker-compose/)

## Support

For issues, questions, or contributions:
- [GitHub Issues](https://github.com/Chapsvision-dev/vault-raft-backup-restore/issues)
- [Project Documentation](../../README.md)

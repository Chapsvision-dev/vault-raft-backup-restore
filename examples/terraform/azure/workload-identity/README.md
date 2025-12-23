# Terraform Module - Azure with Workload Identity

Complete Terraform module to deploy Vault Raft Backup & Restore Operator with **Azure AD Workload Identity** for passwordless, secret-free authentication.

## What This Module Creates

### Azure Resources
- User-Assigned Managed Identity
- Federated Identity Credential (for Workload Identity)
- Storage Container for snapshots
- RBAC role assignment (Storage Blob Data Contributor)

### Kubernetes Resources
- Namespace (`vault-hashicorp`)
- ServiceAccount with Workload Identity annotations
- CronJob for scheduled backups

### Vault Resources
- Kubernetes auth backend
- Vault policy for Raft operations
- Kubernetes auth backend role

## Prerequisites

1. **Azure**
   - Azure subscription with active credentials
   - AKS cluster with **Workload Identity enabled**
   - AKS cluster with **OIDC issuer enabled**
   - Existing Storage Account
   - Permissions to create Managed Identities and federated credentials

2. **Kubernetes**
   - AKS cluster (1.23+) with Workload Identity feature enabled
   - kubectl configured with admin access
   - Cluster CA certificate and authentication token

3. **Vault**
   - Vault cluster with Raft storage backend
   - Vault admin token
   - Network connectivity from Kubernetes to Vault

4. **Terraform**
   - Terraform >= 1.5
   - Required providers:
     - azurerm ~> 4.0
     - azuread ~> 3.0
     - kubernetes ~> 2.20
     - vault ~> 4.0

## Enable Workload Identity on AKS

If not already enabled, enable Workload Identity on your AKS cluster:

```bash
# Enable OIDC issuer
az aks update \
  --resource-group <resource-group> \
  --name <aks-cluster-name> \
  --enable-oidc-issuer

# Enable Workload Identity
az aks update \
  --resource-group <resource-group> \
  --name <aks-cluster-name> \
  --enable-workload-identity

# Get OIDC issuer URL (for verification)
az aks show \
  --resource-group <resource-group> \
  --name <aks-cluster-name> \
  --query "oidcIssuerProfile.issuerUrl" -o tsv
```

## Quick Start

### 1. Configure Variables

Copy the example file and update with your values:

```bash
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
```

**Required variables:**
- `azure_subscription_id` - Your Azure subscription ID
- `resource_group_name` - Resource group containing storage account and AKS
- `storage_account_name` - Existing storage account name
- `aks_cluster_name` - AKS cluster name (for OIDC issuer URL)
- `kubernetes_host` - Kubernetes API server endpoint
- `kubernetes_cluster_ca_certificate` - Base64 encoded CA cert
- `kubernetes_token` - Kubernetes authentication token
- `vault_address` - Vault server address (external)
- `vault_token` - Vault admin token

### 2. Get Kubernetes Credentials

```bash
# For AKS
az aks get-credentials --resource-group <rg> --name <aks-cluster-name>

# Get API server
kubectl cluster-info | grep "Kubernetes control plane"

# Get CA certificate (base64 encoded)
kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}'

# Create a service account token for Terraform
kubectl create serviceaccount terraform-provisioner -n kube-system
kubectl create clusterrolebinding terraform-provisioner \
  --clusterrole=cluster-admin \
  --serviceaccount=kube-system:terraform-provisioner

# Get token
kubectl create token terraform-provisioner -n kube-system --duration=8760h
```

### 3. Initialize and Apply

```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply
```

### 4. Verify Deployment

```bash
# Check Managed Identity
az identity show \
  --name $(terraform output -raw managed_identity_name) \
  --resource-group <resource-group>

# Check federated credential
az identity federated-credential list \
  --identity-name $(terraform output -raw managed_identity_name) \
  --resource-group <resource-group>

# Check CronJob
kubectl get cronjob -n vault-hashicorp vault-raft-backup

# Check ServiceAccount annotation
kubectl get sa -n vault-hashicorp vault-raft-operator-sa -o yaml

# Trigger manual backup
terraform output -raw manual_backup_command | bash

# Check logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup --follow
```

## Configuration

### Backup Schedule

Customize backup frequency (default: daily at 2 AM UTC):

```hcl
backup_schedule = "0 */6 * * *"  # Every 6 hours
```

Common schedules:
- Every 6 hours: `"0 */6 * * *"`
- Every 12 hours: `"0 */12 * * *"`
- Daily at 3 AM: `"0 3 * * *"`
- Weekly on Sunday: `"0 2 * * 0"`

### Resource Limits

Adjust based on snapshot size:

```hcl
resource_requests_cpu    = "200m"
resource_requests_memory = "512Mi"
resource_limits_cpu      = "1000m"
resource_limits_memory   = "1Gi"
```

### Vault Internal Address

If Vault runs in the same cluster:

```hcl
vault_internal_address = "http://vault.vault-hashicorp.svc:8200"
```

If Vault is external:

```hcl
vault_internal_address = "https://vault.example.com:8200"
```

## Outputs

After applying, view important information:

```bash
# All outputs
terraform output

# Specific output
terraform output cronjob_name
terraform output managed_identity_client_id
terraform output oidc_issuer_url
terraform output manual_backup_command
```

## How Workload Identity Works

```
┌─────────────────────┐
│   Kubernetes Pod    │
│  (ServiceAccount)   │
│  with annotation    │
└──────────┬──────────┘
           │ 1. Pod gets projected SA token
           │
           ▼
┌─────────────────────┐
│   Azure AD          │
│   Token Exchange    │
└──────────┬──────────┘
           │ 2. Exchange K8s token for Azure AD token
           │    (using federated credential)
           ▼
┌─────────────────────┐
│  Managed Identity   │
│  Azure AD Token     │
└──────────┬──────────┘
           │ 3. Use token to access Azure Storage
           │    (RBAC enforced)
           ▼
┌─────────────────────┐
│  Azure Blob         │
│    Storage          │
└─────────────────────┘
```

Key components:
1. **ServiceAccount annotation**: Links K8s SA to Azure Managed Identity
2. **Pod label**: `azure.workload.identity/use: "true"` enables token projection
3. **Federated credential**: Establishes trust between AKS OIDC and Managed Identity
4. **No secrets**: Tokens are short-lived and automatically rotated

## Restore Operations

This module only sets up backup automation. For restores, you need to manually create a Job.

### Create Restore Job

1. **List snapshots:**
   ```bash
   az storage blob list \
     --account-name $(terraform output -raw storage_account_name) \
     --container-name $(terraform output -raw storage_container_name) \
     --auth-mode login \
     --query "[].name" -o table
   ```

2. **Create restore Job manifest** (`restore-job.yaml`):
   ```yaml
   apiVersion: batch/v1
   kind: Job
   metadata:
     name: vault-raft-restore-manual
     namespace: vault-hashicorp
   spec:
     template:
       metadata:
         labels:
           azure.workload.identity/use: "true"  # Required
       spec:
         serviceAccountName: vault-raft-operator-sa
         restartPolicy: Never
         containers:
           - name: operator
             image: chapsvision/vault-raft-backup-operator:latest
             args: ["restore"]
             env:
               - name: VAULT_ADDR
                 value: "http://vault.vault-hashicorp.svc:8200"
               - name: VAULT_AUTH_METHOD
                 value: "kubernetes"
               - name: VAULT_K8S_ROLE
                 value: "vault-raft-operator"
               - name: BACKUP_PROVIDER
                 value: "azure"
               - name: AZURE_STORAGE_ACCOUNT
                 value: "mystorageaccount"
               - name: AZURE_STORAGE_CONTAINER
                 value: "vault-snapshots"
               - name: RESTORE_SOURCE
                 value: "snapshots/2025-01-15-02-00-00.snap"  # UPDATE THIS
               - name: RESTORE_TARGET
                 value: "/data/restored.snap"
   ```

3. **Seal Vault, apply Job, then unseal:**
   ```bash
   vault operator seal
   kubectl apply -f restore-job.yaml
   kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-restore --follow
   vault operator unseal <key1>
   vault operator unseal <key2>
   vault operator unseal <key3>
   ```

## Advantages Over Service Principal

✅ **No secrets to manage** - Eliminates Service Principal credentials
✅ **Auto-rotated tokens** - Azure AD tokens are short-lived and automatically refreshed
✅ **Pod-level identity** - Each pod gets its own identity token
✅ **Better audit trail** - Azure AD sign-in logs track all token exchanges
✅ **Zero-trust security** - Federated credential restricts identity to specific namespace/SA
✅ **Simpler rotation** - No credential rotation required

## Security Best Practices

1. **Federated Credential Scope**
   - Automatically scoped to specific namespace and ServiceAccount
   - Subject: `system:serviceaccount:vault-hashicorp:vault-raft-operator-sa`

2. **RBAC Permissions**
   - Managed Identity has minimal permissions (Storage Blob Data Contributor only)
   - Scoped to specific storage account

3. **Vault Token**
   - Use limited-TTL token for Terraform operations
   - Store in secure location (Azure Key Vault, HashiCorp Vault)
   - Revoke after infrastructure setup

4. **State File**
   - Use remote backend (Azure Storage, Terraform Cloud)
   - Enable encryption at rest
   - Restrict access

### Remote Backend Example

```hcl
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate"
    container_name       = "tfstate"
    key                  = "vault-raft-operator-workload-identity.tfstate"
  }
}
```

## Troubleshooting

### Issue: Workload Identity token exchange fails

```bash
# Verify OIDC issuer is enabled
az aks show \
  --resource-group <rg> \
  --name <aks-cluster-name> \
  --query "oidcIssuerProfile.issuerUrl"

# Check federated credential configuration
az identity federated-credential show \
  --identity-name $(terraform output -raw managed_identity_name) \
  --resource-group <rg> \
  --name vault-raft-operator-federated-credential

# Verify pod has correct label
kubectl get pod -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup -o yaml | grep "azure.workload.identity/use"
```

### Issue: 403 Forbidden accessing storage

```bash
# Verify Managed Identity has correct role
az role assignment list \
  --assignee $(terraform output -raw managed_identity_principal_id) \
  --all

# Check if RBAC propagated (wait 5-10 minutes)
az role assignment list \
  --scope /subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.Storage/storageAccounts/<account> \
  --assignee $(terraform output -raw managed_identity_principal_id)
```

### Issue: ServiceAccount annotation missing

```bash
# Verify annotation was applied
kubectl get sa -n vault-hashicorp vault-raft-operator-sa -o jsonpath='{.metadata.annotations}'

# Should show:
# {"azure.workload.identity/client-id":"<client-id>"}
```

## Cleanup

Remove all resources:

```bash
# Destroy Terraform-managed resources
terraform destroy
```

**Warning:** This does NOT delete existing snapshots in Azure Storage.

## Module Structure

```
.
├── main.tf              # Provider configuration
├── variables.tf         # Input variables
├── azure.tf             # Azure resources (Managed Identity, federated credential, storage, RBAC)
├── kubernetes.tf        # Kubernetes resources (namespace, SA with annotations, CronJob)
├── vault.tf             # Vault resources (policy, auth backend, role)
├── outputs.tf           # Output values
├── terraform.tfvars.example  # Example configuration
└── README.md            # This file
```

## Next Steps

- Set up monitoring for CronJob success/failure
- Configure Azure Storage lifecycle policies for retention
- Test restore procedures
- Set up alerts for backup failures
- Document disaster recovery runbook
- Enable Azure AD sign-in logs to audit token exchanges

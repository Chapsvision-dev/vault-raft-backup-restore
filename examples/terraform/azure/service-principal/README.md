# Terraform Module - Azure with Service Principal

Complete Terraform module to deploy Vault Raft Backup & Restore Operator with Azure Service Principal authentication.

## What This Module Creates

### Azure Resources
- Azure AD Application and Service Principal
- Service Principal credentials (auto-rotates annually)
- Storage Container for snapshots
- RBAC role assignment (Storage Blob Data Contributor)

### Kubernetes Resources
- Namespace (`vault-hashicorp`)
- ServiceAccount
- Secret with Service Principal credentials
- CronJob for scheduled backups

### Vault Resources
- Kubernetes auth backend
- Vault policy for Raft operations
- Kubernetes auth backend role

## Prerequisites

1. **Azure**
   - Azure subscription with active credentials
   - Existing Storage Account
   - Permissions to create Service Principals

2. **Kubernetes**
   - Kubernetes cluster (1.23+)
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
     - time ~> 0.12

## Quick Start

### 1. Configure Variables

Copy the example file and update with your values:

```bash
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
```

**Required variables:**
- `azure_subscription_id` - Your Azure subscription ID
- `resource_group_name` - Resource group containing storage account
- `storage_account_name` - Existing storage account name
- `kubernetes_host` - Kubernetes API server endpoint
- `kubernetes_cluster_ca_certificate` - Base64 encoded CA cert
- `kubernetes_token` - Kubernetes authentication token
- `vault_address` - Vault server address (external)
- `vault_token` - Vault admin token

### 2. Get Kubernetes Credentials

```bash
# For AKS
az aks get-credentials --resource-group <rg> --name <cluster-name>

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
# Check CronJob
kubectl get cronjob -n vault-hashicorp vault-raft-backup

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
terraform output manual_backup_command
terraform output service_principal_client_id
```

## Service Principal Credential Rotation

The module automatically rotates Service Principal credentials annually using the `time_rotating` resource. To manually rotate:

```bash
# Trigger rotation by tainting the resource
terraform taint time_rotating.sp_rotation

# Apply to create new credentials
terraform apply

# Old credentials remain valid for a grace period
```

## Restore Operations

This module only sets up backup automation. For restores, you need to manually create a Job.

### Create Restore Job

1. **List snapshots:**
   ```bash
   az storage blob list \
     --account-name $(terraform output -raw storage_account_name) \
     --container-name $(terraform output -raw storage_container_name) \
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
               - name: RESTORE_SOURCE
                 value: "snapshots/2025-01-15-02-00-00.snap"  # UPDATE THIS
               - name: RESTORE_TARGET
                 value: "/data/restored.snap"
             envFrom:
               - secretRef:
                   name: azure-sp-creds
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

## Security Best Practices

1. **Vault Token**
   - Use limited-TTL token for Terraform operations
   - Store in secure location (e.g., HashiCorp Vault, Azure Key Vault)
   - Revoke after infrastructure setup

2. **Kubernetes Token**
   - Use dedicated ServiceAccount for Terraform
   - Limit token duration
   - Rotate periodically

3. **Service Principal**
   - Automatic annual rotation configured
   - Minimal RBAC permissions (Storage Blob Data Contributor only)
   - Credentials stored in Kubernetes Secret

4. **State File**
   - Use remote backend (Azure Storage, Terraform Cloud)
   - Enable encryption at rest
   - Restrict access (contains sensitive credentials)

### Remote Backend Example

```hcl
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate"
    container_name       = "tfstate"
    key                  = "vault-raft-operator.tfstate"
  }
}
```

## Troubleshooting

### Issue: Service Principal permissions

```bash
# Verify role assignment
az role assignment list \
  --assignee $(terraform output -raw service_principal_client_id) \
  --all
```

### Issue: Kubernetes authentication

```bash
# Test kubectl connectivity
kubectl get nodes

# Verify token is valid
kubectl auth can-i get pods --all-namespaces
```

### Issue: Vault authentication

```bash
# Test Vault connectivity
vault status

# Verify token permissions
vault token lookup
```

## Migration to Workload Identity

For better security, migrate to [Workload Identity](../workload-identity/):

1. Apply Workload Identity Terraform module
2. Test backup with new authentication
3. Run `terraform destroy` on this module
4. Manually delete Service Principal if desired

## Cleanup

Remove all resources:

```bash
# Destroy Terraform-managed resources
terraform destroy

# Optionally delete Service Principal manually
az ad sp delete --id $(terraform output -raw service_principal_client_id)
```

**Warning:** This does NOT delete existing snapshots in Azure Storage.

## Module Structure

```
.
├── main.tf              # Provider configuration
├── variables.tf         # Input variables
├── azure.tf             # Azure resources (SP, storage, RBAC)
├── kubernetes.tf        # Kubernetes resources (namespace, SA, CronJob, secret)
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
- Consider migrating to Workload Identity for passwordless auth

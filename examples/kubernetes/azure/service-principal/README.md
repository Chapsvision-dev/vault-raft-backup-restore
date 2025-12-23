# Kubernetes with Azure Service Principal

Deploy using **Azure Service Principal** credentials stored in Kubernetes Secrets for authentication to Azure Storage.

## Prerequisites

- Kubernetes cluster (1.23+)
- Vault deployed with Raft storage backend
- Vault Kubernetes authentication configured
- Azure Storage Account and Container
- Azure Service Principal with Storage Blob Data Contributor role

## Setup Steps

### 1. Create Azure Service Principal

```bash
# Set variables
RESOURCE_GROUP="your-resource-group"
STORAGE_ACCOUNT="yourstorageaccount"
SP_NAME="vault-raft-operator-sp"

# Create Service Principal
az ad sp create-for-rbac \
  --name $SP_NAME \
  --role "Storage Blob Data Contributor" \
  --scopes /subscriptions/YOUR_SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT

# Output will show:
# {
#   "appId": "xxx-xxx-xxx",           # This is AZURE_CLIENT_ID
#   "displayName": "vault-raft-operator-sp",
#   "password": "xxx-xxx",            # This is AZURE_CLIENT_SECRET
#   "tenant": "xxx-xxx-xxx"           # This is AZURE_TENANT_ID
# }
```

Save these values securely - you'll need them in step 3.

### 2. Deploy RBAC Resources

```bash
kubectl apply -f rbac.yaml
```

This creates:
- `vault-hashicorp` namespace
- `vault-raft-operator-sa` ServiceAccount

### 3. Create Azure Credentials Secret

Edit [azure-secret.yaml](azure-secret.yaml) and update:

```yaml
data:
  AZURE_CLIENT_ID: "<base64-encoded-client-id>"
  AZURE_TENANT_ID: "<base64-encoded-tenant-id>"
  AZURE_CLIENT_SECRET: "<base64-encoded-secret>"
```

To base64 encode:
```bash
echo -n 'your-client-id' | base64
echo -n 'your-tenant-id' | base64
echo -n 'your-client-secret' | base64
```

Apply the secret:
```bash
kubectl apply -f azure-secret.yaml
```

### 4. Configure Vault Kubernetes Auth

```bash
# Enable Kubernetes auth backend
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
  kubernetes_host="https://kubernetes.default.svc:443"

# Create the Vault policy
vault policy write raft-backup-restore ../../vault-policy/raft-backup-restore.hcl

# Create a role for the ServiceAccount
vault write auth/kubernetes/role/vault-raft-operator \
  bound_service_account_names=vault-raft-operator-sa \
  bound_service_account_namespaces=vault-hashicorp \
  policies=raft-backup-restore \
  ttl=1h
```

### 5. Deploy Backup CronJob

Edit [backup-cronjob.yaml](backup-cronjob.yaml) to configure:
- `schedule` (default: daily at 2 AM UTC)
- `VAULT_ADDR` (if different from default)
- `AZURE_STORAGE_ACCOUNT` (your storage account name)
- `AZURE_STORAGE_CONTAINER` (your container name)
- `BACKUP_TARGET` (blob prefix, e.g., "snapshots")

```bash
kubectl apply -f backup-cronjob.yaml
```

Verify the CronJob:
```bash
kubectl get cronjob -n vault-hashicorp vault-raft-backup
kubectl describe cronjob -n vault-hashicorp vault-raft-backup
```

### 6. Trigger Manual Backup

```bash
kubectl create job -n vault-hashicorp \
  --from=cronjob/vault-raft-backup \
  vault-raft-backup-manual-$(date +%s)

# Check logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup --follow
```

## Restore Operations

**⚠️ WARNING:** Restore operations will overwrite your Vault cluster state. Only perform restores when:
- You have verified the snapshot integrity
- All Vault cluster members are sealed
- You have communicated the maintenance window to stakeholders

### Manual Restore Process

1. **List available snapshots:**
   ```bash
   az storage blob list \
     --account-name $STORAGE_ACCOUNT \
     --container-name vault-snapshots \
     --query "[].name" -o table
   ```

2. **Edit restore-job.yaml:**
   Update `RESTORE_SOURCE` with your snapshot blob key:
   ```yaml
   env:
     - name: RESTORE_SOURCE
       value: "snapshots/2025-01-15-02-00-00.snap"  # UPDATE THIS
   ```

3. **Seal all Vault nodes:**
   ```bash
   vault operator seal
   ```

4. **Apply the restore Job:**
   ```bash
   kubectl apply -f restore-job.yaml
   ```

5. **Monitor restore:**
   ```bash
   kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-restore --follow
   ```

6. **Unseal Vault after restore:**
   ```bash
   vault operator unseal <key1>
   vault operator unseal <key2>
   vault operator unseal <key3>
   ```

7. **Cleanup Job:**
   ```bash
   kubectl delete job -n vault-hashicorp vault-raft-restore-manual
   ```

## Security Considerations

### Service Principal Security

1. **Rotate credentials regularly** (recommended: every 90 days)
2. **Use minimal RBAC permissions** - Storage Blob Data Contributor only on required storage account
3. **Store secrets in Kubernetes Secrets** (not ConfigMaps or environment variables in manifests)
4. **Consider using SealedSecrets or External Secrets Operator** for GitOps workflows
5. **Enable Azure Storage logging** to audit all access

### Rotating Service Principal Credentials

```bash
# Reset Service Principal credentials
az ad sp credential reset \
  --id $AZURE_CLIENT_ID \
  --append  # Use --append to keep old credential temporarily

# Update Kubernetes secret
kubectl create secret generic azure-sp-creds \
  --from-literal=AZURE_CLIENT_ID=xxx \
  --from-literal=AZURE_TENANT_ID=xxx \
  --from-literal=AZURE_CLIENT_SECRET=xxx \
  --namespace vault-hashicorp \
  --dry-run=client -o yaml | kubectl apply -f -

# Test with manual backup job
kubectl create job -n vault-hashicorp \
  --from=cronjob/vault-raft-backup \
  test-new-creds-$(date +%s)

# If successful, remove old credential
az ad sp credential delete \
  --id $AZURE_CLIENT_ID \
  --key-id <old-key-id>
```

## Migration to Workload Identity

For better security without managing secrets, consider migrating to [Workload Identity](../workload-identity/):

1. Follow Workload Identity setup guide
2. Test backup with new authentication method
3. Update CronJob to remove `envFrom` secret reference
4. Delete Azure SP credentials secret
5. Revoke Service Principal (optional)

**Advantages of Workload Identity:**
- No secrets to manage or rotate
- Short-lived, auto-renewed tokens
- Better audit trail via Azure AD
- Zero trust security model

## Troubleshooting

### Check Secret

```bash
# Verify secret exists
kubectl get secret -n vault-hashicorp azure-sp-creds

# Check secret contents (base64 encoded)
kubectl get secret -n vault-hashicorp azure-sp-creds -o yaml
```

### Common Issues

**Issue: "authentication failed" errors**
- Verify Service Principal credentials are correct
- Check credentials are base64 encoded in secret
- Ensure Service Principal has not expired
- Verify tenant ID is correct

**Issue: "403 Forbidden" accessing storage**
- Verify Service Principal has Storage Blob Data Contributor role
- Check role assignment scope includes your storage account
- Wait 5-10 minutes for RBAC propagation

**Issue: "secret not found"**
- Verify secret name matches in CronJob/Job `envFrom` section
- Check secret is in the same namespace (`vault-hashicorp`)
- Ensure secret was created successfully

## Configuration Reference

### Environment Variables

Set in [backup-cronjob.yaml](backup-cronjob.yaml) and [restore-job.yaml](restore-job.yaml):

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `VAULT_ADDR` | Yes | Vault server address | `http://vault.vault-hashicorp.svc:8200` |
| `VAULT_AUTH_METHOD` | Yes | Authentication method | `kubernetes` |
| `VAULT_K8S_MOUNT` | Yes | Kubernetes auth mount path | `kubernetes` |
| `VAULT_K8S_ROLE` | Yes | Kubernetes auth role name | `vault-raft-operator` |
| `BACKUP_PROVIDER` | Yes | Storage provider | `azure` |
| `AZURE_STORAGE_ACCOUNT` | Yes | Azure storage account name | `mystorageaccount` |
| `AZURE_STORAGE_CONTAINER` | Yes | Container name | `vault-snapshots` |
| `BACKUP_SOURCE` | Backup only | Local snapshot path | `/data/snapshot.snap` |
| `BACKUP_TARGET` | Backup only | Remote blob prefix | `snapshots` |
| `RESTORE_SOURCE` | Restore only | Remote blob key | `snapshots/2025-01-15.snap` |
| `RESTORE_TARGET` | Restore only | Local restore path | `/data/restored.snap` |
| `LOG_LEVEL` | No | Log level | `info` |
| `LOG_FORMAT` | No | Log format | `json` |

Azure SP credentials (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET`) are loaded from the `azure-sp-creds` secret via `envFrom`.

### Resource Limits

Default configuration:

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "256Mi"
    ephemeral-storage: "256Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
    ephemeral-storage: "1Gi"
```

Adjust based on snapshot size and backup frequency.

## Next Steps

- Set up monitoring and alerting for backup job failures
- Configure Azure Storage lifecycle policies for retention
- Test restore procedures in non-production
- Consider migrating to Workload Identity for enhanced security
- Document your disaster recovery runbook

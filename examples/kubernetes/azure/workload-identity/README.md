# Kubernetes with Azure AD Workload Identity

Deploy using **Azure AD Workload Identity** for secure, passwordless authentication to Azure Storage without managing Service Principal secrets.

## Prerequisites

- AKS cluster with [Workload Identity enabled](https://learn.microsoft.com/en-us/azure/aks/workload-identity-deploy-cluster)
- OIDC issuer enabled on AKS cluster
- Azure Managed Identity with Storage Blob Data Contributor role
- Vault deployed with Raft storage backend
- Vault Kubernetes authentication configured

## Architecture

```
┌─────────────────────┐
│   Kubernetes Pod    │
│  (ServiceAccount)   │
└──────────┬──────────┘
           │ Workload Identity
           │ Token Exchange
           ▼
┌─────────────────────┐
│  Azure Managed      │
│    Identity         │
└──────────┬──────────┘
           │ RBAC
           │ Storage Blob Data Contributor
           ▼
┌─────────────────────┐
│  Azure Blob         │
│    Storage          │
└─────────────────────┘
```

## Setup Steps

### 1. Create Azure Managed Identity

```bash
# Set variables
RESOURCE_GROUP="your-resource-group"
LOCATION="westeurope"
IDENTITY_NAME="vault-raft-operator-identity"
STORAGE_ACCOUNT="yourstorageaccount"

# Create managed identity
az identity create \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

# Get identity details
IDENTITY_CLIENT_ID=$(az identity show \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --query clientId -o tsv)

IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

echo "Client ID: $IDENTITY_CLIENT_ID"
echo "Principal ID: $IDENTITY_PRINCIPAL_ID"
```

### 2. Grant Storage Permissions

```bash
# Get storage account resource ID
STORAGE_ACCOUNT_ID=$(az storage account show \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --query id -o tsv)

# Assign Storage Blob Data Contributor role
az role assignment create \
  --assignee $IDENTITY_PRINCIPAL_ID \
  --role "Storage Blob Data Contributor" \
  --scope $STORAGE_ACCOUNT_ID
```

### 3. Configure AKS Workload Identity

```bash
# Get AKS OIDC issuer URL
AKS_CLUSTER="your-aks-cluster"
OIDC_ISSUER=$(az aks show \
  --name $AKS_CLUSTER \
  --resource-group $RESOURCE_GROUP \
  --query oidcIssuerProfile.issuerUrl -o tsv)

# Create federated credential
az identity federated-credential create \
  --name "vault-raft-operator-federated-credential" \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --issuer $OIDC_ISSUER \
  --subject "system:serviceaccount:vault-hashicorp:vault-raft-operator-sa"
```

### 4. Update ServiceAccount

Edit [rbac.yaml](rbac.yaml) and replace `YOUR_MANAGED_IDENTITY_CLIENT_ID` with your `$IDENTITY_CLIENT_ID`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-raft-operator-sa
  namespace: vault-hashicorp
  annotations:
    azure.workload.identity/client-id: "YOUR_MANAGED_IDENTITY_CLIENT_ID"  # Update this
  labels:
    azure.workload.identity/use: "true"
```

Then apply:
```bash
kubectl apply -f rbac.yaml
```

### 5. Configure Vault Kubernetes Auth

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

### 6. Update and Deploy Backup CronJob

Edit [backup-cronjob.yaml](backup-cronjob.yaml) to configure:
- `VAULT_ADDR` (if different from default)
- `AZURE_STORAGE_ACCOUNT` (your storage account name)
- `AZURE_STORAGE_CONTAINER` (your container name)
- `BACKUP_TARGET` (blob prefix, e.g., "snapshots")

```bash
kubectl apply -f backup-cronjob.yaml
```

Verify:
```bash
kubectl get cronjob -n vault-hashicorp vault-raft-backup
kubectl describe cronjob -n vault-hashicorp vault-raft-backup
```

### 7. Trigger Manual Backup

```bash
kubectl create job -n vault-hashicorp \
  --from=cronjob/vault-raft-backup \
  vault-raft-backup-manual-$(date +%s)

# Check logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup --follow
```

## Restore Operations

**⚠️ WARNING:** Restore will overwrite Vault cluster state. Ensure all nodes are sealed first.

### Manual Restore

1. **Find snapshot blob:**
   ```bash
   az storage blob list \
     --account-name $STORAGE_ACCOUNT \
     --container-name vault-snapshots \
     --auth-mode login \
     --query "[].name" -o table
   ```

2. **Edit restore-job.yaml:**
   Update `RESTORE_SOURCE` with your snapshot blob key:
   ```yaml
   - name: RESTORE_SOURCE
     value: "snapshots/2025-01-15-02-00-00.snap"  # Update this
   ```

3. **Seal Vault:**
   ```bash
   vault operator seal
   ```

4. **Apply restore Job:**
   ```bash
   kubectl apply -f restore-job.yaml
   ```

5. **Monitor:**
   ```bash
   kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-restore --follow
   ```

6. **Unseal Vault:**
   ```bash
   vault operator unseal <key1>
   vault operator unseal <key2>
   vault operator unseal <key3>
   ```

## Advantages of Workload Identity

1. **No Secrets** - No Service Principal credentials to manage or rotate
2. **Short-Lived Tokens** - Azure AD tokens automatically refreshed
3. **Pod-Level Identity** - Each pod gets its own identity token
4. **Audit Trail** - Azure AD logs all identity token exchanges
5. **Zero Trust** - Federated credential limits identity to specific namespace/ServiceAccount

## Troubleshooting

### Check Workload Identity Setup

```bash
# Verify ServiceAccount annotations
kubectl get sa -n vault-hashicorp vault-raft-operator-sa -o yaml

# Check pod labels
kubectl get pod -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup -o yaml | grep -A5 labels

# Verify federated credential
az identity federated-credential list \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP
```

### Common Issues

**Issue: "failed to acquire token" errors**
- Verify pod has label `azure.workload.identity/use: "true"`
- Check ServiceAccount annotation with correct client ID
- Verify federated credential subject matches namespace/ServiceAccount name
- Ensure OIDC issuer URL is correct

**Issue: "403 Forbidden" accessing storage**
- Verify Managed Identity has Storage Blob Data Contributor role
- Check role assignment scope (should be at storage account level)
- Wait 5-10 minutes for RBAC propagation

**Issue: Token exchange fails**
- Verify AKS has Workload Identity enabled
- Check OIDC issuer is configured
- Verify federated credential issuer URL matches AKS OIDC URL

## Migration from Service Principal

To migrate from Service Principal authentication:

1. Create Managed Identity and federated credential (steps above)
2. Grant storage permissions to Managed Identity
3. Update ServiceAccount with annotation
4. Remove `envFrom` section from CronJob/Job (no more secrets needed)
5. Delete Azure SP credentials secret: `kubectl delete secret -n vault-hashicorp azure-sp-creds`
6. Test backup job
7. Revoke old Service Principal credentials

## Security Best Practices

1. **Limit federated credential scope** to specific namespace and ServiceAccount
2. **Use least-privilege RBAC** - Storage Blob Data Contributor only on required containers
3. **Enable Azure Storage logging** to audit all access
4. **Rotate nothing** - Workload Identity tokens are short-lived and auto-renewed
5. **Monitor identity usage** in Azure AD sign-in logs

## Next Steps

- Set up monitoring for backup job success/failure
- Configure Azure Storage lifecycle policies for snapshot retention
- Test disaster recovery procedures
- Document your specific restore runbook

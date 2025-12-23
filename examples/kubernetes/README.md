# Kubernetes Deployment Examples

Deploy the Vault Raft Backup & Restore Operator on Kubernetes using CronJobs for scheduled backups and one-time Jobs for restores.

## Authentication Methods

Choose your Azure authentication method:

### 🔐 [Service Principal](azure/service-principal/)
Use Azure Service Principal credentials stored in Kubernetes Secrets.

**Best for:**
- Non-AKS Kubernetes clusters
- Multi-cloud environments
- Quick setup without additional Azure configuration

[View Service Principal Guide →](azure/service-principal/)

### 🎯 [Workload Identity](azure/workload-identity/) (Recommended)
Use Azure AD Workload Identity for passwordless, secret-free authentication.

**Best for:**
- AKS clusters (Azure Kubernetes Service)
- Zero-trust security requirements
- Eliminating secret management overhead

[View Workload Identity Guide →](azure/workload-identity/)

---

## Quick Start

For detailed setup instructions, see the guides for each authentication method:

- **[Service Principal Setup →](azure/service-principal/README.md)** - Step-by-step guide for Service Principal authentication
- **[Workload Identity Setup →](azure/workload-identity/README.md)** - Step-by-step guide for Workload Identity (recommended for AKS)

### Basic Flow

1. **Create RBAC resources** - ServiceAccount and namespace
2. **Configure Azure authentication** - Either Service Principal secret or Workload Identity
3. **Configure Vault Kubernetes auth** - Enable auth backend and create role
4. **Deploy CronJob** - Scheduled backups
5. **Test** - Trigger manual backup and verify

### Quick Commands

```bash
# Apply RBAC (choose your auth method directory)
kubectl apply -f azure/service-principal/rbac.yaml
# OR
kubectl apply -f azure/workload-identity/rbac.yaml

# For Service Principal: create Azure credentials secret
kubectl apply -f azure/service-principal/azure-secret.yaml

# Deploy backup CronJob
kubectl apply -f azure/service-principal/backup-cronjob.yaml
# OR
kubectl apply -f azure/workload-identity/backup-cronjob.yaml

# Trigger manual backup
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
   # Use Azure CLI or portal to find snapshot blob key
   az storage blob list \
     --account-name mystorageaccount \
     --container-name vault-snapshots \
     --query "[].name" -o table
   ```

2. **Edit restore-job.yaml:**
   ```yaml
   env:
     - name: RESTORE_SOURCE
       value: "snapshots/2025-01-15-02-00-00.snap"  # UPDATE THIS
   ```

3. **Seal all Vault nodes:**
   ```bash
   # Seal Vault before restore
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

## Configuration Reference

### Environment Variables

All environment variables are configured in the manifest YAML files under `spec.template.spec.containers[].env`.

#### Vault Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `VAULT_ADDR` | Yes | Vault server address | `http://vault.vault-hashicorp.svc:8200` |
| `VAULT_AUTH_METHOD` | Yes | Authentication method | `kubernetes` |
| `VAULT_K8S_MOUNT` | Yes | Kubernetes auth mount path | `kubernetes` |
| `VAULT_K8S_ROLE` | Yes | Kubernetes auth role name | `vault-raft-operator` |

#### Azure Provider Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `BACKUP_PROVIDER` | Yes | Storage provider | `azure` |
| `AZURE_STORAGE_ACCOUNT` | Yes | Azure storage account name | `mystorageaccount` |
| `AZURE_STORAGE_CONTAINER` | Yes | Container name | `vault-snapshots` |
| `AZURE_CLIENT_ID` | Yes (SP auth) | Service Principal client ID | `xxx-xxx-xxx` |
| `AZURE_TENANT_ID` | Yes (SP auth) | Azure tenant ID | `xxx-xxx-xxx` |
| `AZURE_CLIENT_SECRET` | Yes (SP auth) | Service Principal secret | Set via Secret |

#### Backup Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `BACKUP_SOURCE` | Yes | Local snapshot path | `/data/snapshot.snap` |
| `BACKUP_TARGET` | Yes | Remote blob prefix | `snapshots` |

#### Restore Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `RESTORE_SOURCE` | Yes | Remote blob key | `snapshots/2025-01-15.snap` |
| `RESTORE_TARGET` | Yes | Local restore path | `/data/restored.snap` |

#### Logging Configuration

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `LOG_LEVEL` | No | Log level | `info` |
| `LOG_FORMAT` | No | Log format | `json` |

### Resource Limits

Default resource configuration:

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

Adjust based on your snapshot size and cluster load.

### CronJob Schedule

Default schedule is daily at 2 AM UTC:
```yaml
schedule: "0 2 * * *"
```

Common schedules:
- Every 6 hours: `"0 */6 * * *"`
- Every 12 hours: `"0 */12 * * *"`
- Daily at 3 AM: `"0 3 * * *"`
- Weekly on Sunday at 2 AM: `"0 2 * * 0"`

## Security Best Practices

1. **Use ServiceAccount tokens** for Kubernetes auth (automatic rotation)
2. **Enable Pod Security Standards:**
   ```yaml
   securityContext:
     runAsNonRoot: true
     seccompProfile:
       type: RuntimeDefault
   containerSecurityContext:
     allowPrivilegeEscalation: false
     readOnlyRootFilesystem: true
     capabilities:
       drop: ["ALL"]
   ```

3. **Limit RBAC permissions** to only required Vault paths
4. **Use Azure Service Principal** with minimal blob permissions (read/write on specific container)
5. **Enable audit logging** in Vault to track snapshot operations
6. **Store credentials in Secrets**, never in ConfigMaps or manifests

## Troubleshooting

### Check CronJob Status
```bash
kubectl get cronjob -n vault-hashicorp
kubectl describe cronjob -n vault-hashicorp vault-raft-backup
```

### View Recent Jobs
```bash
kubectl get jobs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup
```

### Check Logs
```bash
# Backup logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-backup --tail=100

# Restore logs
kubectl logs -n vault-hashicorp -l app.kubernetes.io/name=vault-raft-restore --tail=100
```

### Common Issues

**Issue: "permission denied" errors**
- Verify ServiceAccount has correct RBAC role
- Check Vault Kubernetes auth role bindings
- Verify Vault policy allows `sys/storage/raft/snapshot` operations

**Issue: "blob not found" on restore**
- Verify `RESTORE_SOURCE` blob key exists
- Check Azure credentials and permissions
- Use `az storage blob list` to verify blob name

**Issue: Job stuck in "Pending"**
- Check pod events: `kubectl describe pod -n vault-hashicorp <pod-name>`
- Verify resource quotas aren't exceeded
- Check if volume mounts are available

**Issue: "connection refused" to Vault**
- Verify `VAULT_ADDR` is correct
- Check Vault service is running: `kubectl get svc -n vault-hashicorp`
- Verify network policies allow pod-to-pod communication

## Monitoring

### Metrics to Monitor

1. **Job Success Rate:**
   ```bash
   kubectl get jobs -n vault-hashicorp \
     -l app.kubernetes.io/name=vault-raft-backup \
     --sort-by=.status.startTime
   ```

2. **Snapshot Size Over Time:**
   ```bash
   az storage blob list \
     --account-name mystorageaccount \
     --container-name vault-snapshots \
     --query "[].{name:name, size:properties.contentLength}" \
     -o table
   ```

3. **Job Duration:**
   Check logs for timing information:
   ```bash
   kubectl logs -n vault-hashicorp <job-pod> | grep -i "duration\|elapsed"
   ```

### Alerting Recommendations

- Alert on CronJob failures (consecutive failures > 2)
- Alert on snapshot size anomalies (sudden increase/decrease)
- Alert on missing scheduled backups
- Alert on restore operations (for audit trail)

## Cleanup

Remove all resources:
```bash
kubectl delete -f backup-cronjob.yaml
kubectl delete -f restore-job.yaml
kubectl delete -f azure-secret.yaml
kubectl delete -f rbac.yaml
```

## Next Steps

- Consider using [Terraform examples](../terraform/) for full infrastructure automation
- Set up monitoring and alerting for backup jobs
- Test restore procedures in non-production environment
- Document your disaster recovery runbook

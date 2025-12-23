# Docker Compose - Restore Example

This example demonstrates how to restore a Vault Raft snapshot using Docker Compose.

## ⚠️ WARNING

**Restoring a snapshot will:**
- Stop the Vault cluster
- Replace all current data with snapshot data
- Require manual unsealing after restore

**Only perform restores:**
- During planned maintenance windows
- After verifying the snapshot integrity
- With a tested disaster recovery plan

## Prerequisites

- Docker and Docker Compose installed
- Vault cluster accessible and **all nodes stopped except leader**
- Azure Storage Account with the snapshot blob
- Vault token with restore permissions
- **Backup of current state before restore**

## Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your configuration:
   - Set `VAULT_ADDR` to your Vault **leader** node
   - Set `VAULT_TOKEN` with a token that has the `raft-backup-restore` policy
   - Set `RESTORE_SOURCE` to the exact blob key to restore (e.g., `snapshots/2025-01-15T02-00-01Z.snap`)
   - Configure Azure storage credentials

3. **Stop all Vault follower nodes** (keep only the leader running)

4. Run the restore:
   ```bash
   docker compose up
   ```

5. Check the logs for success/errors:
   ```bash
   docker compose logs
   ```

6. **Unseal Vault** after restore:
   ```bash
   vault operator unseal <key1>
   vault operator unseal <key2>
   vault operator unseal <key3>
   ```

7. **Start follower nodes** and rejoin the cluster

## What Happens

1. Container authenticates to Vault using the provided token
2. Downloads snapshot from Azure Blob Storage (`RESTORE_SOURCE`)
3. Saves snapshot temporarily to `/data/restored.snap`
4. Restores snapshot via `POST /v1/sys/storage/raft/snapshot`
5. Vault becomes sealed and requires unsealing
6. Container exits

## Finding the Right Snapshot

List available snapshots in Azure:

```bash
az storage blob list \
  --account-name mystorageaccount \
  --container-name vault-snapshots \
  --prefix snapshots/ \
  --output table
```

Or use Azure Portal to browse the `vault-snapshots` container.

## Troubleshooting

**Restore rejected:**
- Ensure only the leader node is running
- Check follower nodes are stopped: `vault operator raft list-peers`
- Verify snapshot file is valid (not corrupted)

**Download failures:**
- Verify `RESTORE_SOURCE` blob key exists in Azure
- Check Service Principal has "Storage Blob Data Reader" role
- Verify SAS token has read permissions

**Vault won't unseal:**
- Use original unseal keys (from initial `vault operator init`)
- Check Vault logs for errors: `docker logs vault`
- Verify snapshot is from same cluster

## Testing Restore (Recommended)

Before restoring production:

1. Deploy a test Vault cluster
2. Initialize with test keys
3. Restore your snapshot
4. Verify data integrity
5. Document unseal keys and recovery steps

# Docker Compose - Backup Example

This example demonstrates how to run a one-time Vault Raft backup using Docker Compose.

## Prerequisites

- Docker and Docker Compose installed
- Vault cluster accessible from Docker host
- Azure Storage Account with container created
- Vault token with backup permissions

## Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your configuration:
   - Set `VAULT_ADDR` to your Vault server
   - Set `VAULT_TOKEN` with a token that has the `raft-backup-restore` policy
   - Configure Azure storage credentials

3. Run the backup:
   ```bash
   docker compose up
   ```

4. Check the logs:
   ```bash
   docker compose logs
   ```

## What Happens

1. Container starts and authenticates to Vault using the provided token
2. Creates a Raft snapshot via `GET /v1/sys/storage/raft/snapshot`
3. Saves snapshot temporarily to `/data/snapshot.snap`
4. Uploads snapshot to Azure Blob Storage at `snapshots/YYYY-MM-DDTHH-MM-SSZ.snap`
5. Container exits

## Scheduled Backups

To run backups on a schedule, use a cron job on the Docker host:

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * cd /path/to/this/directory && docker compose up
```

Or use a container scheduler like [Ofelia](https://github.com/mcuadros/ofelia).

## Troubleshooting

**Authentication failures:**
- Verify `VAULT_TOKEN` is valid: `vault token lookup`
- Check token has required policy: `vault token capabilities sys/storage/raft/snapshot`

**Azure upload failures:**
- Verify storage account name and container exist
- Check Service Principal has "Storage Blob Data Contributor" role
- Test SAS token validity and permissions

**Container permission errors:**
- Ensure `/data` volume is writable
- Check SELinux/AppArmor policies if applicable

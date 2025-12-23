# Vault Policy for Raft Backup & Restore Operations
# Apply with: vault policy write raft-backup-restore raft-backup-restore.hcl

# GET /v1/sys/storage/raft/snapshot
# Required for creating backups
path "sys/storage/raft/snapshot" {
  capabilities = ["read"]
}

# POST /v1/sys/storage/raft/snapshot
# Required for standard restore operations
path "sys/storage/raft/snapshot" {
  capabilities = ["update"]
}

# POST /v1/sys/storage/raft/snapshot-force
# Required for forced restore operations (bypasses safety checks)
path "sys/storage/raft/snapshot-force" {
  capabilities = ["update"]
}

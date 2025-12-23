# Create Vault policy for Raft backup/restore
resource "vault_policy" "raft_backup_restore" {
  name = "raft-backup-restore"

  policy = <<EOT
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
EOT
}

# Enable Kubernetes auth backend
resource "vault_auth_backend" "kubernetes" {
  type = "kubernetes"
  path = "kubernetes"
}

# Configure Kubernetes auth backend
resource "vault_kubernetes_auth_backend_config" "kubernetes" {
  backend            = vault_auth_backend.kubernetes.path
  kubernetes_host    = var.kubernetes_host
  kubernetes_ca_cert = base64decode(var.kubernetes_cluster_ca_certificate)
}

# Create Kubernetes auth backend role
resource "vault_kubernetes_auth_backend_role" "vault_raft_operator" {
  backend                          = vault_auth_backend.kubernetes.path
  role_name                        = var.vault_k8s_auth_backend_role
  bound_service_account_names      = [kubernetes_service_account.vault_raft_operator.metadata[0].name]
  bound_service_account_namespaces = [kubernetes_namespace.vault_hashicorp.metadata[0].name]
  token_policies                   = [vault_policy.raft_backup_restore.name]
  token_ttl                        = 3600  # 1 hour
}

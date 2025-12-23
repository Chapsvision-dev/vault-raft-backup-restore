output "storage_account_name" {
  description = "Azure Storage Account name"
  value       = data.azurerm_storage_account.vault_snapshots.name
}

output "storage_container_name" {
  description = "Azure Storage Container name"
  value       = azurerm_storage_container.vault_snapshots.name
}

output "managed_identity_name" {
  description = "Azure Managed Identity name"
  value       = azurerm_user_assigned_identity.vault_raft_operator.name
}

output "managed_identity_client_id" {
  description = "Azure Managed Identity client ID"
  value       = azurerm_user_assigned_identity.vault_raft_operator.client_id
}

output "managed_identity_principal_id" {
  description = "Managed Identity principal ID"
  value       = azurerm_user_assigned_identity.vault_raft_operator.principal_id
}

output "managed_identity_id" {
  description = "Managed Identity resource ID"
  value       = azurerm_user_assigned_identity.vault_raft_operator.id
}

output "federated_credential_name" {
  description = "Federated identity credential name"
  value       = azurerm_federated_identity_credential.vault_raft_operator.name
}

output "oidc_issuer_url" {
  description = "AKS OIDC issuer URL"
  value       = data.azurerm_kubernetes_cluster.aks.oidc_issuer_url
}

output "kubernetes_namespace" {
  description = "Kubernetes namespace"
  value       = kubernetes_namespace.vault_hashicorp.metadata[0].name
}

output "kubernetes_service_account" {
  description = "Kubernetes ServiceAccount name"
  value       = kubernetes_service_account.vault_raft_operator.metadata[0].name
}

output "vault_policy_name" {
  description = "Vault policy name"
  value       = vault_policy.raft_backup_restore.name
}

output "vault_k8s_role_name" {
  description = "Vault Kubernetes auth backend role name"
  value       = vault_kubernetes_auth_backend_role.vault_raft_operator.role_name
}

output "cronjob_name" {
  description = "Kubernetes CronJob name"
  value       = kubernetes_cron_job_v1.vault_raft_backup.metadata[0].name
}

output "backup_schedule" {
  description = "Backup CronJob schedule"
  value       = kubernetes_cron_job_v1.vault_raft_backup.spec[0].schedule
}

output "manual_backup_command" {
  description = "Command to trigger a manual backup"
  value       = "kubectl create job -n ${kubernetes_namespace.vault_hashicorp.metadata[0].name} --from=cronjob/${kubernetes_cron_job_v1.vault_raft_backup.metadata[0].name} vault-raft-backup-manual-$(date +%s)"
}

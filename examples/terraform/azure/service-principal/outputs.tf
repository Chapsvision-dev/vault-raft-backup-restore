output "storage_account_name" {
  description = "Azure Storage Account name"
  value       = data.azurerm_storage_account.vault_snapshots.name
}

output "storage_container_name" {
  description = "Azure Storage Container name"
  value       = azurerm_storage_container.vault_snapshots.name
}

output "service_principal_client_id" {
  description = "Azure Service Principal client ID"
  value       = azuread_application.vault_raft_operator.client_id
}

output "service_principal_tenant_id" {
  description = "Azure tenant ID"
  value       = data.azurerm_client_config.current.tenant_id
}

output "service_principal_object_id" {
  description = "Service Principal object ID"
  value       = azuread_service_principal.vault_raft_operator.object_id
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

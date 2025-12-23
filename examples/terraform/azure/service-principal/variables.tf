variable "azure_subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "resource_group_name" {
  description = "Azure resource group name where storage account exists"
  type        = string
}

variable "location" {
  description = "Azure region for resources"
  type        = string
  default     = "West Europe"
}

variable "storage_account_name" {
  description = "Azure storage account name for Vault snapshots"
  type        = string
}

variable "storage_container_name" {
  description = "Azure storage container name for backups"
  type        = string
  default     = "vault-snapshots"
}

variable "service_principal_name" {
  description = "Name for the Azure Service Principal"
  type        = string
  default     = "vault-raft-operator-sp"
}

variable "kubernetes_host" {
  description = "Kubernetes API server endpoint"
  type        = string
}

variable "kubernetes_cluster_ca_certificate" {
  description = "Kubernetes cluster CA certificate (base64 encoded)"
  type        = string
  sensitive   = true
}

variable "kubernetes_token" {
  description = "Kubernetes authentication token"
  type        = string
  sensitive   = true
}

variable "kubernetes_namespace" {
  description = "Kubernetes namespace for Vault Raft operator"
  type        = string
  default     = "vault-hashicorp"
}

variable "vault_address" {
  description = "Vault server address"
  type        = string
}

variable "vault_token" {
  description = "Vault admin token for configuration"
  type        = string
  sensitive   = true
}

variable "vault_k8s_auth_backend_role" {
  description = "Vault Kubernetes auth backend role name"
  type        = string
  default     = "vault-raft-operator"
}

variable "vault_internal_address" {
  description = "Vault address accessible from within Kubernetes cluster"
  type        = string
  default     = "http://vault.vault-hashicorp.svc:8200"
}

variable "backup_schedule" {
  description = "CronJob schedule for backups (cron format)"
  type        = string
  default     = "0 2 * * *"  # Daily at 2 AM UTC
}

variable "backup_target_prefix" {
  description = "Blob prefix for backup snapshots"
  type        = string
  default     = "snapshots"
}

variable "operator_image" {
  description = "Docker image for the operator"
  type        = string
  default     = "chapsvision/vault-raft-backup-operator:latest"
}

variable "log_level" {
  description = "Log level for the operator"
  type        = string
  default     = "info"
}

variable "log_format" {
  description = "Log format (json or text)"
  type        = string
  default     = "json"
}

variable "resource_requests_cpu" {
  description = "CPU resource requests"
  type        = string
  default     = "100m"
}

variable "resource_requests_memory" {
  description = "Memory resource requests"
  type        = string
  default     = "256Mi"
}

variable "resource_limits_cpu" {
  description = "CPU resource limits"
  type        = string
  default     = "500m"
}

variable "resource_limits_memory" {
  description = "Memory resource limits"
  type        = string
  default     = "512Mi"
}

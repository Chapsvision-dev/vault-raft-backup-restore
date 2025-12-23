# Create namespace
resource "kubernetes_namespace" "vault_hashicorp" {
  metadata {
    name = var.kubernetes_namespace
  }
}

# Create ServiceAccount
resource "kubernetes_service_account" "vault_raft_operator" {
  metadata {
    name      = "vault-raft-operator-sa"
    namespace = kubernetes_namespace.vault_hashicorp.metadata[0].name
    labels = {
      "app.kubernetes.io/name"    = "vault-raft-operator"
      "app.kubernetes.io/part-of" = "vault-raft-operator"
    }
  }
}

# Create Secret with Azure Service Principal credentials
resource "kubernetes_secret" "azure_sp_creds" {
  metadata {
    name      = "azure-sp-creds"
    namespace = kubernetes_namespace.vault_hashicorp.metadata[0].name
  }

  data = {
    AZURE_CLIENT_ID     = azuread_application.vault_raft_operator.client_id
    AZURE_TENANT_ID     = data.azurerm_client_config.current.tenant_id
    AZURE_CLIENT_SECRET = azuread_service_principal_password.vault_raft_operator.value
  }

  type = "Opaque"
}

# Create CronJob for scheduled backups
resource "kubernetes_cron_job_v1" "vault_raft_backup" {
  metadata {
    name      = "vault-raft-backup"
    namespace = kubernetes_namespace.vault_hashicorp.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "vault-raft-backup"
      "app.kubernetes.io/part-of"   = "vault-raft-operator"
      "app.kubernetes.io/component" = "backup"
    }
  }

  spec {
    schedule                      = var.backup_schedule
    concurrency_policy            = "Forbid"
    successful_jobs_history_limit = 3
    failed_jobs_history_limit     = 3
    suspend                       = false

    job_template {
      metadata {
        labels = {
          "app.kubernetes.io/name"      = "vault-raft-backup"
          "app.kubernetes.io/part-of"   = "vault-raft-operator"
          "app.kubernetes.io/component" = "backup"
        }
      }

      spec {
        backoff_limit = 2

        template {
          metadata {
            labels = {
              "app.kubernetes.io/name"      = "vault-raft-backup"
              "app.kubernetes.io/part-of"   = "vault-raft-operator"
              "app.kubernetes.io/component" = "backup"
            }
          }

          spec {
            automount_service_account_token = true
            service_account_name            = kubernetes_service_account.vault_raft_operator.metadata[0].name
            restart_policy                  = "OnFailure"

            security_context {
              run_as_non_root = true
              seccomp_profile {
                type = "RuntimeDefault"
              }
            }

            volume {
              name = "tmp-data"
              empty_dir {}
            }

            container {
              name  = "operator"
              image = var.operator_image
              args  = ["backup"]

              security_context {
                allow_privilege_escalation = false
                read_only_root_filesystem  = true
                capabilities {
                  drop = ["ALL"]
                }
              }

              resources {
                requests = {
                  cpu               = var.resource_requests_cpu
                  memory            = var.resource_requests_memory
                  ephemeral-storage = "256Mi"
                }
                limits = {
                  cpu               = var.resource_limits_cpu
                  memory            = var.resource_limits_memory
                  ephemeral-storage = "1Gi"
                }
              }

              volume_mount {
                name       = "tmp-data"
                mount_path = "/data"
              }

              env {
                name  = "VAULT_ADDR"
                value = var.vault_internal_address
              }

              env {
                name  = "VAULT_AUTH_METHOD"
                value = "kubernetes"
              }

              env {
                name  = "VAULT_K8S_MOUNT"
                value = "kubernetes"
              }

              env {
                name  = "VAULT_K8S_ROLE"
                value = var.vault_k8s_auth_backend_role
              }

              env {
                name  = "BACKUP_PROVIDER"
                value = "azure"
              }

              env {
                name  = "AZURE_STORAGE_ACCOUNT"
                value = data.azurerm_storage_account.vault_snapshots.name
              }

              env {
                name  = "AZURE_STORAGE_CONTAINER"
                value = azurerm_storage_container.vault_snapshots.name
              }

              env {
                name  = "BACKUP_SOURCE"
                value = "/data/snapshot.snap"
              }

              env {
                name  = "BACKUP_TARGET"
                value = var.backup_target_prefix
              }

              env {
                name  = "LOG_LEVEL"
                value = var.log_level
              }

              env {
                name  = "LOG_FORMAT"
                value = var.log_format
              }

              env_from {
                secret_ref {
                  name = kubernetes_secret.azure_sp_creds.metadata[0].name
                }
              }
            }
          }
        }
      }
    }
  }

  depends_on = [
    azurerm_role_assignment.storage_blob_contributor,
    vault_kubernetes_auth_backend_role.vault_raft_operator
  ]
}

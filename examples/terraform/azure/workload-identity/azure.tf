# Data source for existing storage account
data "azurerm_storage_account" "vault_snapshots" {
  name                = var.storage_account_name
  resource_group_name = var.resource_group_name
}

# Create storage container for snapshots
resource "azurerm_storage_container" "vault_snapshots" {
  name                  = var.storage_container_name
  storage_account_id    = data.azurerm_storage_account.vault_snapshots.id
  container_access_type = "private"
}

# Get current Azure client config
data "azurerm_client_config" "current" {}

# Get AKS cluster for OIDC issuer
data "azurerm_kubernetes_cluster" "aks" {
  name                = var.aks_cluster_name
  resource_group_name = var.resource_group_name
}

# Create User-Assigned Managed Identity
resource "azurerm_user_assigned_identity" "vault_raft_operator" {
  name                = var.managed_identity_name
  resource_group_name = var.resource_group_name
  location            = var.location
}

# Assign Storage Blob Data Contributor role to Managed Identity
resource "azurerm_role_assignment" "storage_blob_contributor" {
  scope                = data.azurerm_storage_account.vault_snapshots.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.vault_raft_operator.principal_id
}

# Create federated identity credential for Workload Identity
resource "azurerm_federated_identity_credential" "vault_raft_operator" {
  name                = "vault-raft-operator-federated-credential"
  resource_group_name = var.resource_group_name
  parent_id           = azurerm_user_assigned_identity.vault_raft_operator.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = data.azurerm_kubernetes_cluster.aks.oidc_issuer_url
  subject             = "system:serviceaccount:${var.kubernetes_namespace}:vault-raft-operator-sa"
}

# Data source for existing storage account
data "azurerm_storage_account" "vault_snapshots" {
  name                = var.storage_account_name
  resource_group_name = var.resource_group_name
}

# Create storage container for snapshots
resource "azurerm_storage_container" "vault_snapshots" {
  name                 = var.storage_container_name
  storage_account_id   = data.azurerm_storage_account.vault_snapshots.id
  container_access_type = "private"
}

# Get current Azure client config
data "azurerm_client_config" "current" {}

# Create Azure AD Application
resource "azuread_application" "vault_raft_operator" {
  display_name = var.service_principal_name
}

# Create Service Principal
resource "azuread_service_principal" "vault_raft_operator" {
  client_id = azuread_application.vault_raft_operator.client_id
}

# Create Service Principal password
resource "azuread_service_principal_password" "vault_raft_operator" {
  service_principal_id = azuread_service_principal.vault_raft_operator.id
  rotate_when_changed = {
    rotation = time_rotating.sp_rotation.id
  }
}

# Rotation timer for Service Principal credentials
resource "time_rotating" "sp_rotation" {
  rotation_days = 365
}

# Assign Storage Blob Data Contributor role to Service Principal
resource "azurerm_role_assignment" "storage_blob_contributor" {
  scope                = data.azurerm_storage_account.vault_snapshots.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azuread_service_principal.vault_raft_operator.object_id
}

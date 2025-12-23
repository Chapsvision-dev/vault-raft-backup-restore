terraform {
  required_version = ">= 1.5"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.20"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}

provider "kubernetes" {
  host                   = var.kubernetes_host
  cluster_ca_certificate = base64decode(var.kubernetes_cluster_ca_certificate)
  token                  = var.kubernetes_token
}

provider "vault" {
  address = var.vault_address
  token   = var.vault_token
}

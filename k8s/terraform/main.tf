terraform {
  required_version = ">= 1.5"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
  }

  # Для учебного проекта — локальный state-файл.
  # В продакшене заменить на: backend "s3" / "gcs" / "azurerm"
  backend "local" {
    path = "terraform.tfstate"
  }
}

provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = var.kube_context
}

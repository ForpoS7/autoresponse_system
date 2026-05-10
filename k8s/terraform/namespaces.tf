locals {
  common_labels = {
    "app.kubernetes.io/managed-by" = "terraform"
    "environment"                  = var.environment
  }
}

resource "kubernetes_namespace" "autoresponse" {
  metadata {
    name   = "autoresponse"
    labels = merge(local.common_labels, {
      "app.kubernetes.io/part-of" = "autoresponse-system"
    })
  }
}

resource "kubernetes_namespace" "kafka" {
  metadata {
    name   = "kafka"
    labels = merge(local.common_labels, {
      "app.kubernetes.io/part-of" = "autoresponse-system"
    })
  }
}

resource "kubernetes_namespace" "monitoring" {
  metadata {
    name   = "monitoring"
    labels = merge(local.common_labels, {
      "app.kubernetes.io/part-of" = "autoresponse-system"
    })
  }
}

resource "kubernetes_namespace" "argocd" {
  metadata {
    name   = "argocd"
    labels = local.common_labels
  }
}

# Service Account для Go autoapply сервиса
# Нужен для чтения ConfigMap/Secret и записи статусов в будущем
resource "kubernetes_service_account" "autoapply" {
  metadata {
    name      = "autoapply-service"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
    annotations = {
      "app.kubernetes.io/component" = "autoapply"
    }
  }
}

# Service Account для Java aggregate сервиса
resource "kubernetes_service_account" "aggregate" {
  metadata {
    name      = "aggregate-service"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
    annotations = {
      "app.kubernetes.io/component" = "aggregate"
    }
  }
}

# Service Account для Go auth сервиса
resource "kubernetes_service_account" "auth" {
  metadata {
    name      = "auth-service"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
    annotations = {
      "app.kubernetes.io/component" = "auth"
    }
  }
}

# Service Account для Python generate сервиса
resource "kubernetes_service_account" "generate" {
  metadata {
    name      = "generate-service"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
    annotations = {
      "app.kubernetes.io/component" = "generate"
    }
  }
}

# Минимальная роль: читать собственные секреты и configmap
resource "kubernetes_role" "app_reader" {
  metadata {
    name      = "app-config-reader"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }

  rule {
    api_groups     = [""]
    resources      = ["secrets", "configmaps"]
    verbs          = ["get", "list", "watch"]
  }
}

resource "kubernetes_role_binding" "autoapply_reader" {
  metadata {
    name      = "autoapply-config-reader"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = kubernetes_role.app_reader.metadata[0].name
  }
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.autoapply.metadata[0].name
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
  }
}

resource "kubernetes_role_binding" "aggregate_reader" {
  metadata {
    name      = "aggregate-config-reader"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = kubernetes_role.app_reader.metadata[0].name
  }
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.aggregate.metadata[0].name
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
  }
}

# Секреты создаются Terraform один раз при инициализации кластера.
# Значения передаются через terraform.tfvars (не хранятся в git).

resource "kubernetes_secret" "postgres_credentials" {
  metadata {
    name      = "postgres-credentials"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }

  type = "Opaque"

  data = {
    POSTGRES_HOST     = var.postgres_host
    POSTGRES_PORT     = tostring(var.postgres_port)
    POSTGRES_DB       = var.postgres_db
    POSTGRES_USER     = var.postgres_user
    POSTGRES_PASSWORD = var.postgres_password
    # Полная connection string для Spring/GORM
    DATABASE_URL = "postgres://${var.postgres_user}:${var.postgres_password}@${var.postgres_host}:${var.postgres_port}/${var.postgres_db}?sslmode=disable"
  }
}

resource "kubernetes_secret" "jwt_signing_key" {
  metadata {
    name      = "jwt-signing-key"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }

  type = "Opaque"

  data = {
    JWT_SECRET = var.jwt_secret
  }
}

resource "kubernetes_secret" "kafka_config" {
  metadata {
    name      = "kafka-config"
    namespace = kubernetes_namespace.autoresponse.metadata[0].name
    labels    = local.common_labels
  }

  type = "Opaque"

  data = {
    KAFKA_BOOTSTRAP_SERVERS = var.kafka_bootstrap
  }
}

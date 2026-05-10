output "namespaces" {
  description = "Созданные неймспейсы"
  value = [
    kubernetes_namespace.autoresponse.metadata[0].name,
    kubernetes_namespace.kafka.metadata[0].name,
    kubernetes_namespace.monitoring.metadata[0].name,
    kubernetes_namespace.argocd.metadata[0].name,
  ]
}

output "service_accounts" {
  description = "Созданные Service Accounts"
  value = {
    autoapply = "${kubernetes_service_account.autoapply.metadata[0].namespace}/${kubernetes_service_account.autoapply.metadata[0].name}"
    aggregate = "${kubernetes_service_account.aggregate.metadata[0].namespace}/${kubernetes_service_account.aggregate.metadata[0].name}"
    auth      = "${kubernetes_service_account.auth.metadata[0].namespace}/${kubernetes_service_account.auth.metadata[0].name}"
    generate  = "${kubernetes_service_account.generate.metadata[0].namespace}/${kubernetes_service_account.generate.metadata[0].name}"
  }
}

output "secret_names" {
  description = "Имена созданных секретов (значения не выводятся)"
  value = [
    kubernetes_secret.postgres_credentials.metadata[0].name,
    kubernetes_secret.jwt_signing_key.metadata[0].name,
    kubernetes_secret.kafka_config.metadata[0].name,
  ]
}

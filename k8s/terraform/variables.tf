variable "kubeconfig_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Путь к kubeconfig"
}

variable "kube_context" {
  type        = string
  default     = "k3d-autoresponse"
  description = "Имя контекста (k3d-<cluster-name>)"
}

variable "environment" {
  type        = string
  default     = "dev"
  description = "Окружение: dev | staging | prod"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Допустимые значения: dev, staging, prod."
  }
}

variable "postgres_password" {
  type        = string
  sensitive   = true
  description = "Пароль PostgreSQL (не коммитить в git)"
}

variable "jwt_secret" {
  type        = string
  sensitive   = true
  description = "JWT signing secret — минимум 32 символа"
}

variable "postgres_user" {
  type    = string
  default = "postgres"
}

variable "postgres_db" {
  type    = string
  default = "autoresponse"
}

variable "postgres_host" {
  type    = string
  default = "postgres.autoresponse.svc.cluster.local"
}

variable "postgres_port" {
  type    = number
  default = 5432
}

variable "kafka_bootstrap" {
  type        = string
  default     = "autoresponse-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092"
  description = "Bootstrap-адрес Kafka (Strimzi создаёт сервис с таким именем)"
}

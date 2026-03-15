.PHONY: help start stop test clean build java go infra

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

help: ## Показать справку
	@echo "$$(tput bold)HH AutoResponse System - Доступные команды$$(tput sgr0)"
	@echo ""
	@echo "$$(tput bold)Основные команды:$$(tput sgr0)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "$$(tput bold)Пример использования:$$(tput sgr0)"
	@echo "  make infra     # Запустить PostgreSQL + Kafka"
	@echo "  make java      # Запустить Java сервис"
	@echo "  make go        # Запустить Go сервис"
	@echo "  make test      # Тестировать API"

infra: ## Запустить инфраструктуру (PostgreSQL + Kafka)
	@echo "$(YELLOW)Запуск инфраструктуры...$(NC)"
	cd hh_aggregate_service && docker-compose up -d
	@echo "$(GREEN)✓ Инфраструктура запущена$(NC)"
	@echo ""
	@echo "Ожидание запуска PostgreSQL (10 сек)..."
	@sleep 10
	@cd hh_aggregate_service && docker-compose ps

stop: ## Остановить инфраструктуру
	@echo "$(YELLOW)Остановка инфраструктуры...$(NC)"
	cd hh_aggregate_service && docker-compose down
	@echo "$(GREEN)✓ Инфраструктура остановлена$(NC)"

java: ## Запустить Java сервис (hh_aggregate_service)
	@echo "$(YELLOW)Запуск Java сервиса на порту 8080...$(NC)"
	cd hh_aggregate_service && ./gradlew bootRun

go: ## Запустить Go сервис (hh_autoapply_service)
	@echo "$(YELLOW)Запуск Go сервиса на порту 8081...$(NC)"
	cd hh_autoapply_service && go run cmd/main.go

build: ## Собрать Go сервис
	@echo "$(YELLOW)Сборка Go сервиса...$(NC)"
	cd hh_autoapply_service && go build -o bin/hh_autoapply_service ./cmd/main.go
	@echo "$(GREEN)✓ Сборка завершена$(NC)"

install-playwright: ## Установить браузеры Playwright
	@echo "$(YELLOW)Установка браузеров Playwright...$(NC)"
	cd hh_autoapply_service && go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
	@echo "$(GREEN)✓ Браузеры установлены$(NC)"

test: ## Тестировать API Go сервиса
	@echo "$(YELLOW)Тестирование API...$(NC)"
	./test-api.sh

test-java: ## Тестировать API Java сервиса
	@echo "$(YELLOW)Тестирование API Java сервиса...$(NC)"
	./test-api.sh 8080

clean: ## Очистить временные файлы
	@echo "$(YELLOW)Очистка...$(NC)"
	cd hh_autoapply_service && go clean
	rm -rf hh_autoapply_service/bin/
	@echo "$(GREEN)✓ Очистка завершена$(NC)"

logs: ## Показать логи Docker контейнеров
	@echo "$(YELLOW)Логи инфраструктуры...$(NC)"
	cd hh_aggregate_service && docker-compose logs -f

ps: ## Показать статус Docker контейнеров
	@echo "$(YELLOW)Статус контейнеров...$(NC)"
	cd hh_aggregate_service && docker-compose ps

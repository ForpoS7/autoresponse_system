.PHONY: help infra stop auth java go build install-playwright test test-java clean logs ps

GREEN  := \033[0;32m
YELLOW := \033[1;33m
NC     := \033[0m

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Инфраструктура ───────────────────────────────────────────────────────────

infra: ## Запустить всю инфраструктуру (PostgreSQL + Kafka + Auth + nginx)
	@echo "$(YELLOW)Запуск инфраструктуры...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)✓ Инфраструктура запущена$(NC)"
	@echo "Ожидание PostgreSQL (10 сек)..."
	@sleep 10
	docker-compose ps

stop: ## Остановить всю инфраструктуру
	@echo "$(YELLOW)Остановка...$(NC)"
	docker-compose down
	@echo "$(GREEN)✓ Остановлено$(NC)"

# ─── Сервисы (запуск на хосте) ────────────────────────────────────────────────

java: ## Запустить Java сервис (hh_aggregate_service) на :8080
	@echo "$(YELLOW)Запуск Java сервиса на :8080...$(NC)"
	cd hh_aggregate_service && ./gradlew bootRun

go: ## Запустить Go сервис (hh_autoapply_service) на :8081
	@echo "$(YELLOW)Запуск Go сервиса на :8081...$(NC)"
	cd hh_autoapply_service && go run cmd/main.go

auth: ## Запустить Auth сервис локально (вне Docker) на :8082
	@echo "$(YELLOW)Запуск Auth сервиса на :8082...$(NC)"
	cd auth_service && go run cmd/main.go

# ─── Сборка ───────────────────────────────────────────────────────────────────

build: ## Собрать Go и Auth сервисы
	@echo "$(YELLOW)Сборка Go сервисов...$(NC)"
	cd hh_autoapply_service && go build -o bin/hh_autoapply_service ./cmd/main.go
	cd auth_service && go build -o bin/auth_service ./cmd/main.go
	@echo "$(GREEN)✓ Сборка завершена$(NC)"

install-playwright: ## Установить браузеры Playwright
	@echo "$(YELLOW)Установка браузеров Playwright...$(NC)"
	cd hh_autoapply_service && go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
	@echo "$(GREEN)✓ Браузеры установлены$(NC)"

# ─── Тестирование ─────────────────────────────────────────────────────────────

test: ## Тестировать API через nginx Gateway (:80)
	@echo "$(YELLOW)Тестирование через API Gateway...$(NC)"
	./test-api.sh 80

test-java: ## Тестировать Java API напрямую (:8080)
	@echo "$(YELLOW)Тестирование Java сервиса напрямую...$(NC)"
	./test-api.sh 8080

test-auth: ## Проверить Auth сервис
	@echo "$(YELLOW)Тест регистрации через Gateway...$(NC)"
	curl -s -X POST http://localhost/auth/register \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"secret123"}' | cat
	@echo ""

# ─── Обслуживание ─────────────────────────────────────────────────────────────

logs: ## Логи всех Docker контейнеров
	docker-compose logs -f

ps: ## Статус Docker контейнеров
	docker-compose ps

clean: ## Очистить артефакты сборки
	@echo "$(YELLOW)Очистка...$(NC)"
	cd hh_autoapply_service && go clean
	cd auth_service && go clean
	rm -rf hh_autoapply_service/bin/ auth_service/bin/
	@echo "$(GREEN)✓ Очистка завершена$(NC)"

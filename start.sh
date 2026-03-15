#!/bin/bash

# Скрипт быстрого запуска HH AutoResponse System
# Использование: ./start.sh

set -e

echo "========================================="
echo "HH AutoResponse System - Запуск"
echo "========================================="

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Проверка требований
echo -e "\n${YELLOW}[1/6] Проверка требований...${NC}"

# Проверка Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker не установлен${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker установлен${NC}"

# Проверка Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go не установлен${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Go установлен ($(go version))${NC}"

# Проверка Java
if ! command -v java &> /dev/null; then
    echo -e "${RED}✗ Java не установлена${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Java установлена ($(java -version 2>&1 | head -1))${NC}"

# Проверка Gradle
if ! command -v gradle &> /dev/null; then
    echo -e "${YELLOW}! Gradle не найден, попробуем через gradlew${NC}"
fi

# Запуск инфраструктуры
echo -e "\n${YELLOW}[2/6] Запуск инфраструктуры (PostgreSQL + Kafka)...${NC}"
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service
docker-compose up -d

# Ожидание запуска БД
echo -e "${YELLOW}[3/6] Ожидание запуска PostgreSQL...${NC}"
sleep 10

# Проверка статуса контейнеров
echo -e "\n${YELLOW}[4/6] Проверка статуса контейнеров...${NC}"
docker-compose ps

# Установка Playwright
echo -e "\n${YELLOW}[5/6] Установка браузеров Playwright...${NC}"
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# Запуск Java сервиса
echo -e "\n${YELLOW}[6/6] Запуск сервисов...${NC}"

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}Инфраструктура запущена!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "Для запуска Java сервиса выполните:"
echo "  cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service"
echo "  gradle bootRun"
echo ""
echo "Для запуска Go сервиса выполните:"
echo "  cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service"
echo "  go run cmd/main.go"
echo ""
echo "Порты:"
echo "  - Java сервис: http://localhost:8080"
echo "  - Go сервис:   http://localhost:8081"
echo "  - PostgreSQL:  localhost:5444"
echo "  - Kafka:       localhost:9092"
echo ""
echo -e "${GREEN}=========================================${NC}"

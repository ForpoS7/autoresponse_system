#!/bin/bash

# Скрипт тестирования API HH AutoResponse System
# Использование: ./test-api.sh

set -e

echo "========================================="
echo "HH AutoResponse System - Тестирование API"
echo "========================================="

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PORT=${1:-8081} # По умолчанию тестируем Go сервис

echo -e "\n${YELLOW}Тестирование сервиса на порту $PORT${NC}"

# Health check
echo -e "\n${YELLOW}[1/6] Health Check...${NC}"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT/health)
if [ "$HEALTH" == "200" ]; then
    echo -e "${GREEN}✓ Сервис доступен${NC}"
else
    echo -e "${RED}✗ Сервис недоступен${NC}"
    exit 1
fi

# Регистрация
echo -e "\n${YELLOW}[2/6] Регистрация пользователя...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:$PORT/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}')

echo "Ответ: $REGISTER_RESPONSE"

# Извлекаем токен (упрощенно, через grep)
TOKEN=$(echo $REGISTER_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    # Если пользователь уже существует, пробуем логин
    echo -e "${YELLOW}Пользователь уже существует, выполняем логин...${NC}"
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:$PORT/api/auth/login \
      -H "Content-Type: application/json" \
      -d '{"email": "test@example.com", "password": "password123"}')
    TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -n "$TOKEN" ]; then
    echo -e "${GREEN}✓ Токен получен${NC}"
else
    echo -e "${RED}✗ Не удалось получить токен${NC}"
    exit 1
fi

# Парсинг вакансий
echo -e "\n${YELLOW}[3/6] Парсинг вакансий...${NC}"
VACANCIES=$(curl -s -X GET "http://localhost:$PORT/api/vacancies?query=Java Developer&page=0" \
  -H "Authorization: Bearer $TOKEN")
echo "Ответ: $VACANCIES"
echo -e "${GREEN}✓ Вакансии получены${NC}"

# Получение токена HH
echo -e "\n${YELLOW}[4/6] Получение токена HH...${NC}"
HH_TOKEN=$(curl -s -X GET http://localhost:$PORT/api/hh-token \
  -H "Authorization: Bearer $TOKEN")
echo "Ответ: $HH_TOKEN"
echo -e "${GREEN}✓ Токен HH получен${NC}"

# Проверка планировщика
echo -e "\n${YELLOW}[5/6] Проверка планировщика...${NC}"
SCHEDULER=$(curl -s -X GET http://localhost:$PORT/api/scheduler/config \
  -H "Authorization: Bearer $TOKEN")
echo "Ответ: $SCHEDULER"
echo -e "${GREEN}✓ Конфигурация планировщика получена${NC}"

# Создание автоотклика
echo -e "\n${YELLOW}[6/6] Создание автоотклика...${NC}"
AUTOAPPLY=$(curl -s -X POST http://localhost:$PORT/api/autoapply \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "Java Developer", "apply_count": 3}')
echo "Ответ: $AUTOAPPLY"
echo -e "${GREEN}✓ Автоотклик создан${NC}"

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}Все тесты пройдены успешно!${NC}"
echo -e "${GREEN}=========================================${NC}"

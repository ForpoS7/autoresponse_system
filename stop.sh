#!/bin/bash

# Скрипт остановки HH AutoResponse System
# Использование: ./stop.sh

set -e

echo "========================================="
echo "HH AutoResponse System - Остановка"
echo "========================================="

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "\n${YELLOW}Остановка Docker контейнеров...${NC}"
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service
docker-compose down

echo -e "\n${GREEN}✓ Все сервисы остановлены${NC}"
echo ""
echo "Примечание: Java и Go сервисы нужно остановить вручную (Ctrl+C)"
echo ""
echo -e "${GREEN}=========================================${NC}"

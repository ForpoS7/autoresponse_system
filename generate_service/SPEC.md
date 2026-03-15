# Generate Service — Specification

## Общая информация

| Параметр | Значение |
|----------|----------|
| **Язык** | Python 3.x |
| **Фреймворк** | requests + kafka-python |
| **Порт** | Не имеет HTTP сервера (Kafka consumer) |
| **Основное назначение** | Генерация сопроводительных писем с помощью LLM |
| **LLM модель** | Ollama Qwen2.5:7b |

## Функционал

### 1. Генерация сопроводительных писем

- **Потребление запросов** из Kafka топика `vacancy_input`
- **Генерация текста** через Ollama API
- **Публикация результата** в Kafka топик `vacancy_output`

### 2. Интеграция с LLM

- **Ollama API** (`http://localhost:11434/api/generate`)
- **Модель:** `qwen2.5:7b`
- **Промпт-инжиниринг** для коротких писем

## Архитектура

```
┌─────────────────────┐         ┌─────────────────────┐
│  hh_autoapply_svc   │  Kafka  │   generate_service  │
│  (Go, port 8081)    │ ──────> │   (Python)          │
│                     │         │                     │
│  • Отправка запроса │         │  • Получение из     │
│    в vacancy_input  │         │    vacancy_input    │
│  • Получение ответа │         │  • Генерация через  │
│    из vacancy_output│         │    Ollama           │
│                     │         │  • Отправка в       │
│                     │ <────── │    vacancy_output   │
│                     │  Kafka  │                     │
└─────────────────────┘         └─────────────────────┘
```

## Структура проекта

```
generate_service/
└── generate.py    # Основной файл сервиса
```

## Зависимости

```python
import json
import requests
from kafka import KafkaConsumer, KafkaProducer
```

### Установка зависимостей

```bash
pip install kafka-python requests
```

## Конфигурация

```python
KAFKA_BROKER = "localhost:9092"
INPUT_TOPIC = "vacancy_input"
OUTPUT_TOPIC = "vacancy_output"

OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL = "qwen2.5:7b"
```

## Формат сообщений

### Входное сообщение (vacancy_input)

```json
{
  "title": "Java Developer",
  "company": "Яндекс",
  "requirements": "Разработка backend-сервисов на Java, опыт с Spring..."
}
```

### Выходное сообщение (vacancy_output)

```json
{
  "title": "Java Developer",
  "company": "Яндекс",
  "generated_text": "Имею релевантный опыт разработки backend-сервисов на Java с использованием Spring. Работал с высоконагруженными системами и микросервисной архитектурой. Готов применить свои навыки в вашей команде."
}
```

## Промпт для генерации

```python
def build_prompt(data: dict) -> str:
    return f"""
Сгенерируй ОЧЕНЬ короткое сопроводительное письмо для отклика на вакансию.

Входные данные:

Название вакансии: "{data['title']}"
Компания: "{data['company']}"
Описание/требования: "{data['requirements']}"

Жёсткие требования:

2–3 предложения, не больше
Без приветствий и подписей
Без фраз: «с большим интересом», «уверен», «буду рад», «внести вклад»
Текст должен выглядеть как написанный человеком, не HR и не нейросетью
Прямо укажи, что есть релевантный опыт по вакансии {data['title']}
Профессионально, но разговорно
Только финальный текст письма
Никаких комментариев, пояснений или советов
Русский язык
не оставляй в конце системный комментарий с [Ваше имя]
"""
```

## Запуск

### Предварительные требования

1. **Установить Ollama:**
   ```bash
   # macOS
   brew install ollama
   
   # Запуск Ollama
   ollama serve
   ```

2. **Скачать модель:**
   ```bash
   ollama pull qwen2.5:7b
   ```

3. **Установить Python зависимости:**
   ```bash
   pip install kafka-python requests
   ```

### Запуск сервиса

```bash
cd generate_service
python generate.py
```

## Интеграция с hh_autoapply_service

### Поток данных

```
1. hh_autoapply_service получает вакансию из Kafka (vacancies.parsed)
2. hh_autoapply_service отправляет запрос в vacancy_input:
   {
     "title": "Go Developer",
     "company": "Сбер",
     "requirements": "..."
   }
3. generate_service генерирует письмо через Ollama
4. generate_service публикует результат в vacancy_output
5. hh_autoapply_service получает сгенерированное письмо
6. hh_autoapply_service использует письмо для автоотклика
```

### Пример интеграции в Go сервисе

```go
// Отправка запроса на генерацию
func (s *CoverLetterService) Generate(vacancy Vacancy) (string, error) {
    // Публикация в vacancy_input
    msg := VacancyInput{
        Title: vacancy.Title,
        Company: vacancy.Employer,
        Requirements: vacancy.Description,
    }
    s.producer.Send(INPUT_TOPIC, msg)
    
    // Получение из vacancy_output
    result := <-s.outputChannel
    return result.GeneratedText, nil
}
```

## Логирование

```python
print("Service started. Waiting Kafka messages...")

for message in consumer:
    try:
        data = message.value
        print("Received:", data)
        # ...
        print("Sent result to Kafka")
    except Exception as e:
        print("Error:", e)
```

## Требования к сопроводительным письмам

| Требование | Описание |
|------------|----------|
| **Длина** | 2-3 предложения, не больше |
| **Приветствия** | Без приветствий и подписей |
| **Клише** | Без «с большим интересом», «уверен», «буду рад» |
| **Стиль** | Как написано человеком, не HR |
| **Контент** | Указание на релевантный опыт |
| **Тон** | Профессионально, но разговорно |
| **Язык** | Русский |

## Kafka

### Consumer

**Топик:** `vacancy_input`  
**Group ID:** `ollama-service`  
**Формат:** JSON

```python
consumer = KafkaConsumer(
    INPUT_TOPIC,
    bootstrap_servers=KAFKA_BROKER,
    value_deserializer=lambda m: json.loads(m.decode("utf-8")),
    auto_offset_reset="latest",
    group_id="ollama-service"
)
```

### Producer

**Топик:** `vacancy_output`  
**Формат:** JSON

```python
producer = KafkaProducer(
    bootstrap_servers=KAFKA_BROKER,
    value_serializer=lambda v: json.dumps(v, ensure_ascii=False).encode("utf-8")
)
```

## Проверка работы

### 1. Проверка Ollama

```bash
# Проверка доступности
curl http://localhost:11434/api/tags

# Тестовая генерация
ollama run qwen2.5:7b "Привет"
```

### 2. Проверка Kafka

```bash
# Просмотр сообщений vacancy_input
docker exec -it hh-autoresponse-kafka /bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic vacancy_input \
  --from-beginning

# Просмотр сообщений vacancy_output
docker exec -it hh-autoresponse-kafka /bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic vacancy_output \
  --from-beginning
```

## Примечания

1. **Ollama должен быть запущен** перед стартом сервиса
2. **Модель qwen2.5:7b** должна быть загружена заранее
3. **Сервис работает в фоне** как Kafka consumer
4. **Таймаут запроса к Ollama** установлен на 120 секунд
5. **Генерация происходит синхронно** для каждого сообщения

---

**Дата создания:** 2026-03-15  
**Версия спецификации:** 1.0

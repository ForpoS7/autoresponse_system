import json
import requests
from kafka import KafkaConsumer, KafkaProducer

KAFKA_BROKER = "localhost:9092"
INPUT_TOPIC   = "vacancy_input"
OUTPUT_TOPIC  = "vacancy_output"

OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL      = "qwen2.5:7b"

consumer = KafkaConsumer(
    INPUT_TOPIC,
    bootstrap_servers=KAFKA_BROKER,
    value_deserializer=lambda m: json.loads(m.decode("utf-8")),
    auto_offset_reset="latest",
    group_id="ollama-service"
)

producer = KafkaProducer(
    bootstrap_servers=KAFKA_BROKER,
    value_serializer=lambda v: json.dumps(v, ensure_ascii=False).encode("utf-8")
)

def build_prompt(data: dict) -> str:
    return f"""
Сгенерируй ОЧЕНЬ короткое сопроводительное письмо для отклика на вакансию.

Название вакансии: "{data['title']}"
Компания: "{data['company']}"
Описание/требования: "{data['requirements']}"

Жёсткие требования:
- 2–3 предложения, не больше
- Без приветствий и подписей
- Без фраз: «с большим интересом», «уверен», «буду рад», «внести вклад»
- Текст должен выглядеть как написанный человеком
- Прямо укажи релевантный опыт по вакансии {data['title']}
- Профессионально, но разговорно
- Только финальный текст письма
- Никаких комментариев и пояснений
- Русский язык
"""

def ask_ollama(prompt: str) -> str:
    response = requests.post(
        OLLAMA_URL,
        json={"model": MODEL, "prompt": prompt, "stream": False},
        timeout=120
    )
    response.raise_for_status()
    return response.json()["response"].strip()

print("generate_service started. Waiting for vacancy_input messages...")

for message in consumer:
    try:
        data = message.value
        print(f"Received: correlationId={data.get('correlationId')} title={data.get('title')}")

        prompt = build_prompt(data)
        result_text = ask_ollama(prompt)

        # correlationId обязательно прокидываем обратно —
        # Go-сервис использует его для маршрутизации к нужной горутине
        output = {
            "correlationId":  data.get("correlationId", ""),
            "generated_text": result_text,
        }

        producer.send(OUTPUT_TOPIC, output)
        producer.flush()
        print(f"Sent to vacancy_output: correlationId={output['correlationId']}")

    except Exception as e:
        print(f"Error: {e}")

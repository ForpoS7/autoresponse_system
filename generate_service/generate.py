import json
import requests
from kafka import KafkaConsumer, KafkaProducer

KAFKA_BROKER = "localhost:9092"
INPUT_TOPIC = "vacancy_input"
OUTPUT_TOPIC = "vacancy_output"

OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL = "qwen2.5:7b"

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

def ask_ollama(prompt: str) -> str:
    response = requests.post(
        OLLAMA_URL,
        json={
            "model": MODEL,
            "prompt": prompt,
            "stream": False
        },
        timeout=120
    )

    response.raise_for_status()
    return response.json()["response"].strip()

print("Service started. Waiting Kafka messages...")

for message in consumer:
    try:
        data = message.value
        print("Received:", data)

        prompt = build_prompt(data)

        result_text = ask_ollama(prompt)

        output = {
            "title": data["title"],
            "company": data["company"],
            "generated_text": result_text
        }

        producer.send(OUTPUT_TOPIC, output)
        producer.flush()

        print("Sent result to Kafka")

    except Exception as e:
        print("Error:", e)
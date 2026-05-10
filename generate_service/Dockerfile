FROM python:3.11-slim
WORKDIR /app
RUN pip install --no-cache-dir \
    kafka-python==2.0.2 \
    requests==2.31.0 \
    python-dotenv==1.0.0
COPY generate.py .
CMD ["python", "-u", "generate.py"]

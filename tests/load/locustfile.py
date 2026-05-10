"""
HH AutoResponse System — Locust load test
Имитирует реалистичный поток пользователей через API Gateway (Istio IngressGateway).

Два типа пользователей:
  BrowseUser  (weight=3) — читает вакансии, лёгкая нагрузка без Kafka-событий
  ApplyUser   (weight=1) — отправляет автоотклики, запускает полный EDA-флоу:
                           parse.requested → vacancies.parsed → vacancy_input → vacancy_output

Запуск (локально против k3d):
  locust -f locustfile.py --host http://localhost:30080 --users 20 --spawn-rate 2

Запуск в k8s (headless):
  locust -f locustfile.py --host http://istio-ingressgateway.istio-ingress \\
         --users 50 --spawn-rate 5 --run-time 5m --headless --csv /results/locust
"""

import random
import logging
from locust import HttpUser, task, between, events

log = logging.getLogger("locust.autoresponse")

# Тестовые пользователи — регистрируются при первом on_start
_TEST_USERS = [
    {"email": f"loadtest{i}@autoresponse.local", "password": "LoadTest1234!"}
    for i in range(100)
]


class _AuthMixin:
    """Получает JWT при старте воркера; повторяет при истечении."""

    token: str = ""

    def on_start(self):  # noqa: D102
        self._acquire_token()

    def _acquire_token(self):
        user = random.choice(_TEST_USERS)
        with self.client.post(
            "/auth/login", json=user, catch_response=True, name="/auth/login"
        ) as resp:
            if resp.status_code == 200:
                self.token = resp.json().get("token", "")
                resp.success()
                return
            # Пользователь ещё не зарегистрирован
            resp.success()

        with self.client.post(
            "/auth/register", json=user, catch_response=True, name="/auth/register"
        ) as resp:
            if resp.status_code in (200, 201):
                self.token = resp.json().get("token", "")
            resp.success()

    def _headers(self):
        return {"Authorization": f"Bearer {self.token}"}

    def _handle(self, resp, name=""):
        """Помечает 401 как «нужен новый токен», не как ошибку."""
        if resp.status_code == 401:
            resp.success()
            self._acquire_token()
        elif resp.status_code == 429:
            # Rate-limit — ожидаемое поведение, не ошибка
            resp.success()


# ---------------------------------------------------------------------------
# Тип 1: пользователь-браузер (вес 3 — большинство трафика)
# ---------------------------------------------------------------------------

class BrowseUser(_AuthMixin, HttpUser):
    """Читает вакансии. Не генерирует события в Kafka."""

    weight = 3
    wait_time = between(1, 4)

    @task(5)
    def get_vacancies(self):
        with self.client.get(
            "/api/vacancies", headers=self._headers(),
            catch_response=True, name="/api/vacancies"
        ) as r:
            self._handle(r)

    @task(2)
    def get_hh_token_status(self):
        with self.client.get(
            "/api/hh-token", headers=self._headers(),
            catch_response=True, name="/api/hh-token"
        ) as r:
            self._handle(r)

    @task(1)
    def get_scheduler_status(self):
        with self.client.get(
            "/api/scheduler", headers=self._headers(),
            catch_response=True, name="/api/scheduler"
        ) as r:
            self._handle(r)

    @task(1)
    def auth_validate(self):
        """Имитирует сервис-к-сервису валидацию токена."""
        with self.client.post(
            "/auth/validate",
            json={"token": self.token},
            catch_response=True, name="/auth/validate"
        ) as r:
            self._handle(r)


# ---------------------------------------------------------------------------
# Тип 2: пользователь-аппликант (вес 1 — генерирует события в Kafka)
# ---------------------------------------------------------------------------

class ApplyUser(_AuthMixin, HttpUser):
    """
    Отправляет автоотклики. Каждый POST /api/autoapply запускает цепочку:
      Go → parse.requested → Java (Playwright) → vacancies.parsed →
      Go → vacancy_input → Python (Ollama) → vacancy_output → HH.ru отклик
    """

    weight = 1
    wait_time = between(4, 10)

    _job_ids: list

    def on_start(self):
        self._job_ids = []
        super().on_start()

    @task(2)
    def post_autoapply(self):
        payload = {
            "keywords": random.choice(["golang developer", "python backend",
                                       "java spring", "devops kubernetes"]),
            "city": random.choice(["Москва", "Санкт-Петербург", "Новосибирск"]),
            "salary_min": random.randrange(100_000, 250_000, 25_000),
            "experience": random.choice(["noExperience", "between1And3", "between3And6"]),
        }
        with self.client.post(
            "/api/autoapply", json=payload, headers=self._headers(),
            catch_response=True, name="/api/autoapply [POST]"
        ) as r:
            self._handle(r)
            if r.status_code in (200, 201, 202):
                data = r.json()
                job_id = data.get("id") or data.get("jobId") or data.get("job_id")
                if job_id:
                    self._job_ids.append(str(job_id))
                    if len(self._job_ids) > 20:
                        self._job_ids.pop(0)

    @task(4)
    def poll_autoapply_status(self):
        """Имитирует периодический опрос статуса — нагружает кеш Redis."""
        if not self._job_ids:
            return
        job_id = random.choice(self._job_ids)
        with self.client.get(
            f"/api/autoapply/{job_id}", headers=self._headers(),
            catch_response=True, name="/api/autoapply/[id]"
        ) as r:
            self._handle(r)

    @task(1)
    def list_archive(self):
        prefix = random.choice(["logs", "vacancies"])
        with self.client.get(
            f"/api/archive/list?prefix={prefix}", headers=self._headers(),
            catch_response=True, name="/api/archive/list"
        ) as r:
            self._handle(r)

    @task(1)
    def manual_archive(self):
        """Редкий тяжёлый запрос — архивация в MinIO."""
        with self.client.post(
            "/api/archive/run", headers=self._headers(),
            catch_response=True, name="/api/archive/run"
        ) as r:
            self._handle(r)


# ---------------------------------------------------------------------------
# Хуки для логирования в stdout (видны в pod логах)
# ---------------------------------------------------------------------------

@events.request.add_listener
def on_request(request_type, name, response_time, response_length,
               response, context, exception, **_kw):
    if exception:
        log.warning("FAIL %s %s — %s", request_type, name, exception)
    elif response and response.status_code >= 500:
        log.warning("HTTP %d  %s %s", response.status_code, request_type, name)

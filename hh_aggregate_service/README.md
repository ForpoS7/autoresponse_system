# HH.ru Parser (Java + Playwright)

Простой парсер вакансий HH.ru на Java с использованием Playwright.

## Структура проекта

```
hh_aggregate_service/
├── src/main/java/by/icemens/hh_aggregate_service/
│   ├── config/
│   │   └── Settings.java        # Настройки
│   ├── model/
│   │   └── Vacancy.java         # Модель вакансии
│   ├── service/
│   │   ├── BrowserManager.java  # Менеджер браузера
│   │   └── VacancySearchService.java  # Сервис поиска
│   ├── Login.java               # Авторизация
│   └── VacancyParserDemo.java   # Демонстрация парсинга
├── build.gradle
└── README.md
```

## Установка

### 1. Требования

- Java 17 или выше
- Gradle 8.x

### 2. Сборка проекта

```bash
cd hh_aggregate_service
gradle build
```

### 3. Установка браузеров Playwright

```bash
# Windows (PowerShell)
pwsh bin\install-playwright.ps1

# Или через Java
gradle run --args="install-browsers"
```

## Использование

### Шаг 1: Авторизация

Перед первым запуском нужно авторизоваться на HH.ru:

```bash
gradle run -DmainClass=by.icemens.hh_aggregate_service.Login
```

Или через Java:

```bash
gradle build
java -cp build/classes/java/main;build/resources/main by.icemens.hh_aggregate_service.Login
```

**Что происходит:**
1. Откроется браузер Chromium
2. Войдите в свой аккаунт HH.ru
3. После успешного входа нажмите Enter в терминале
4. Сессия сохранится в `~/.hh_autoresponse/hh_session.json`

### Шаг 2: Парсинг вакансий

Запуск с параметрами по умолчанию:

```bash
gradle run
```

Свои параметры (запрос и страница):

```bash
gradle run --args="Python Developer 0"
```

Или через Java:

```bash
java -cp build/classes/java/main;build/resources/main by.icemens.hh_aggregate_service.VacancyParserDemo "Java Developer" 0
```

## Пример вывода

```
============================================================
           Парсинг вакансий HH.ru
============================================================

Запрос: Java Developer
Страница: 0

[OK] Найдено вакансий: 20

1. Java Developer
   Компания: Яндекс
   URL: https://hh.ru/vacancy/12345678

2. Senior Java Developer
   Компания: Сбер
   URL: https://hh.ru/vacancy/87654321

...
```

## Настройки

Файл: `src/main/java/by/icemens/hh_aggregate_service/config/Settings.java`

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| `defaultSearchText` | Java Developer | Запрос поиска по умолчанию |
| `areaCode` | 113 | Код региона (113 = Россия) |
| `browserHeadless` | true | Режим без отображения браузера |
| `pageTimeout` | 30000 | Таймаут загрузки страницы (мс) |

## Сессия

Сессия сохраняется в:
- **Windows:** `C:\Users\<user>\.hh_autoresponse\hh_session.json`
- **Linux/macOS:** `~/.hh_autoresponse/hh_session.json`

Для обновления сессии повторно запустите `Login.main()`.

## Возможности

- ✅ Поиск вакансий с указанием запроса и страницы
- ✅ Парсинг заголовка, компании, URL вакансии
- ✅ Проверка на капчу
- ✅ Сохранение сессии браузера
- ✅ Headless режим (по умолчанию)

## Ограничения

- ❌ Нет отклика на вакансии (можно добавить)
- ❌ Нет обработки пагинации (только одна страница за запрос)
- ❌ Нет сохранения результатов в файл
- ❌ Нет работы с зарплатой и регионом (можно добавить)

## Troubleshooting

### Session file not found

```
[ERROR] Сессия не найдена!
Запустите Login.main() для авторизации.
```

**Решение:** Запустите `Login.main()` для авторизации.

### Captcha detected

```
[ERROR] Сработала защита от ботов (captcha)
```

**Решение:**
1. Отключите headless режим в `Settings.java`: `browserHeadless = false`
2. Обновите сессию через `Login.main()`

### Playwright browser not found

```
Exception: Executable doesn't exist at ...
```

**Решение:** Установите браузеры Playwright:

```bash
# Windows
pwsh bin\install-playwright.ps1

# Linux/macOS
playwright install chromium
```

## Расширение функциональности

Для добавления новых возможностей:

1. **Парсинг зарплаты:** Добавьте поля `salaryFrom`, `salaryTo`, `currency` в `Vacancy.java` и парсинг в `VacancySearchService.java`
2. **Отклик на вакансию:** Создайте `VacancyApplyService.java` по аналогии с Python версией
3. **Сохранение в JSON:** Добавьте запись результатов в файл в `VacancyParserDemo.java`
4. **Пагинация:** Добавьте цикл по страницам в `VacancyParserDemo.java`

## Лицензия

MIT

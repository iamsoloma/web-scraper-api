# Web Scraper API

Web Scraper API на Go с использованием [chromedp](https://github.com/chromedp/chromedp) для извлечения данных с веб-страниц.

## Возможности

- Извлечение заголовка страницы
- Полный скриншот страницы (base64)
- HTML-код страницы
- Конвертация страницы в Markdown (GitHub Flavored)
- Текст страницы
- Сбор изображений со страницы
- Сбор ссылок со страницы (с исключением внутренних якорей)
- Проверка `robots.txt` перед скрейпингом
- Санитизация HTML (удаление `script`/`style` и inline-обработчиков)
- Асинхронная обработка задач через очередь на базе PostgreSQL
- Хранение результатов в S3-совместимом хранилище

## Архитектура

Запрос на скрейпинг не выполняется синхронно: API создаёт задачу в очереди PostgreSQL и возвращает её идентификатор. Фоновый воркер забирает задачи из очереди, выполняет скрейпинг и сохраняет результат в S3-совместимое хранилище. Статус и результат задачи можно получить по идентификатору.

```
POST /scrape ──▶ PostgreSQL (очередь задач)
                     │
                     ▼
              Воркер (chromedp)
                     │
                     ▼
        S3-хранилище (результат) ◀── GET /scrape/{id}
```

## Структура проекта

```
├── api/          # HTTP-сервер, обработчики, middleware логирования
├── client/       # Go-клиент для работы с API
├── config/       # Конфигурация приложения (config.yaml)
├── examples/     # Пример использования Go-клиента
├── queue/        # Очередь задач на PostgreSQL
├── scrape/       # Скрейпинг (chromedp), парсер robots.txt, типы
├── storage/      # S3-совместимое хранилище результатов
├── utils/        # Утилиты (санитизация HTML)
├── .bruno/       # Коллекция запросов для Bruno
├── config.yaml   # Файл конфигурации
├── docker-compose.yml  # PostgreSQL и SeaweedFS для локальной разработки
└── Dockerfile    # Образ приложения с Chromium
```

## Требования

- Go 1.26 или выше
- Google Chrome или Chromium (для локального запуска)
- Docker и Docker Compose (для инфраструктуры)

## Запуск

### 1. Инфраструктура

Поднимите PostgreSQL (очередь задач) и SeaweedFS (S3-хранилище):

```bash
docker compose up -d
```

### 2. Приложение

Клонируйте репозиторий и установите зависимости:

```bash
git clone https://sourcecraft.dev/egorsolomahin1/web-scraper-api.git
cd web-scraper-api
go mod tidy
```

Запустите сервер:

```bash
go run main.go
```

Сервер будет доступен по адресу `http://localhost:8080`.

### Запуск через Docker

```bash
docker build -t web-scraper-api .
docker run -p 8080:8080 --network host web-scraper-api
```

Образ включает все необходимые зависимости, включая Chromium.

## Использование API

### Проверка состояния

```bash
curl http://localhost:8080/health
```

Ответ:

```json
{
  "Status": "Ok",
  "CurrentTime": "...",
  "Uptime": "...",
  "Version": "0.0.7",
  "UserAgent": "OpinionBot"
}
```

### Создание задачи скрейпинга

Отправьте `POST` запрос к эндпоинту `/scrape`:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "screenshot": true,
    "html": true,
    "markdown": true,
    "text": true,
    "images": true,
    "links": true
  }' \
  http://localhost:8080/scrape
```

Ответ — идентификатор задачи:

```json
{
  "taskID": 1
}
```

### Получение результата

Получите статус и результат задачи по идентификатору:

```bash
curl http://localhost:8080/scrape/1
```

Пока задача обрабатывается, статус будет `pending` или `processing`. Когда статус станет `done`, в поле `result` появится результат скрейпинга:

```json
{
  "id": 1,
  "task": "scrape",
  "payload": { "url": "https://example.com", ... },
  "status": "done",
  "priority": 0,
  "attempts": 0,
  "created_at": "...",
  "last_attempt_at": "...",
  "result": {
    "title": "Заголовок страницы",
    "screenshot": "base64-encoded-screenshot",
    "html": "HTML страницы",
    "markdown": "Markdown страницы",
    "text": "Текст страницы",
    "images": {
      "подпись": "ссылка"
    },
    "links": {
      "подпись": "ссылка"
    },
    "date": "Дата обращения в UTC"
  }
}
```

### Параметры запроса

| Поле        | Тип    | Описание                              |
|-------------|--------|---------------------------------------|
| `url`       | string | URL страницы для скрейпинга (обязательно) |
| `screenshot`| bool   | Сделать полный скриншот страницы      |
| `html`      | bool   | Вернуть HTML-код страницы             |
| `markdown`  | bool   | Вернуть страницу в формате Markdown   |
| `text`      | bool   | Вернуть текст страницы                |
| `images`    | bool   | Собрать изображения со страницы       |
| `links`     | bool   | Собрать ссылки со страницы            |

## Go-клиент

Для удобной работы с API можно использовать клиент из пакета `client`:

```go
client := client.NewClient("http://localhost:8080")

page, err := client.Scrape(&scrape.ScrapeRequest{
    URL:        "https://en.wikipedia.org/wiki/War_and_Peace",
    Screenshot: true,
    Markdown:   true,
})
```

Полный пример — в файле [examples/scrapePage.go](examples/scrapePage.go).

## Конфигурация

Приложение читает конфигурацию из файла `config.yaml`:

```yaml
listenAddr: "0.0.0.0:8080"
userAgent: "OpinionBot"
database:
  domain: "localhost"
  port: "5432"
  user: "main"
  password: "qwerty"
  dbname: "tasks"
storage:
  bucket: "tasks"
  endpoint: "http://localhost:8333"
  region: "us-east-1"
  AccessKeyID: "main"
  SecretAccessKey: "qwerty"
```

Параметры можно переопределить через переменные окружения (например, `PORT`, `USER_AGENT`).

## Тестирование запросов

В каталоге [.bruno](.bruno) находится готовая коллекция запросов для [Bruno](https://www.usebruno.com/): `health`, `scrape` и `scrapeResult`.

## Зависимости

- [chromedp](https://github.com/chromedp/chromedp) — автоматизация Chrome/Chromium
- [pgx](https://github.com/jackc/pgx) — драйвер PostgreSQL
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) — работа с S3-хранилищем
- [html-to-markdown](https://github.com/firecrawl/html-to-markdown) — конвертация HTML в Markdown
- [cleanenv](https://github.com/ilyakaznacheev/cleanenv) — загрузка конфигурации
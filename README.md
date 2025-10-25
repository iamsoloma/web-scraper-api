# Web Scraper API

Web Scraper API на Golang с использованием chromedp для извлечения данных со страниц.

## Описание

Этот API позволяет извлекать информацию с веб-страниц, включая:
- Название страницы
- Скриншот страницы
- Текст страницы
- Изображения со страницы
- Ссылки на другие страницы

## Установка

1. Убедитесь, что у вас установлен Go 1.19 или выше
2. Установите Google Chrome или Chromium
3. Клонируйте репозиторий:
   ```
   git clone https://github.com/egorsolomahin1/web-scraper-api.git
   cd web-scraper-api
   ```
4. Установите зависимости:
   ```
   go mod tidy
   ```

## Запуск

### Локальный запуск
```
go run server.go
```

Сервер будет доступен по адресу `http://localhost:8080`

### Запуск через Docker
```
docker build -t web-scraper-api .
docker run -p 8080:8080 web-scraper-api
```

### Запуск через docker-compose
```
docker-compose up
```

## Использование

Отправьте GET запрос к эндпоинту `/scrape` с параметром `url`:

```
curl "http://localhost:8080/scrape?url=https://example.com"
```

Ответ будет в формате JSON:
```json
{
  "title": "Название страницы",
  "screenshot": "base64-encoded-screenshot",
  "text": "Текст страницы",
  "images": ["ссылка1", "ссылка2", ...],
  "links": ["ссылка1", "ссылка2", ...]
}
```

## Зависимости

- [chromedp](https://github.com/chromedp/chromedp) - для автоматизации Chrome/Chromium

## Docker

Dockerfile включает все необходимые зависимости для запуска приложения в контейнере, включая Chromium.
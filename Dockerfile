FROM golang:1.19-alpine AS builder

# Установка зависимостей
RUN apk add --no-cache git

# Установка Chrome
RUN apk add --no-cache chromium

WORKDIR /app

# Копируем go mod и sum файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Финальный этап
FROM alpine:latest

# Установка Chrome для runtime
RUN apk add --no-cache chromium

WORKDIR /root/

# Копируем бинарный файл из builder этапа
COPY --from=builder /app/main .

# Открываем порт
EXPOSE 8080

# Запуск приложения
CMD ["./main"]
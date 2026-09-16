FROM golang:1.27.1-alpine3.24 AS builder

WORKDIR /build

COPY . .

RUN go build -o /build/main


FROM alpine:3.24.1

RUN apk add --no-cache chromium

WORKDIR /usr/bin

# Копируем бинарный файл из builder этапа
COPY --from=builder /build/main .

# Открываем порт
EXPOSE 8080

# Запуск приложения
CMD ["./main"]
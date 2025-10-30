FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY . .

RUN go build -o /build/main


FROM alpine:3.22.2

RUN apk add --no-cache chromium

WORKDIR /usr/bin

# Копируем бинарный файл из builder этапа
COPY --from=builder /build/main .

# Открываем порт
EXPOSE 8080

# Запуск приложения
CMD ["./main"]
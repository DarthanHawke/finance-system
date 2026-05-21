FROM golang:1.26.3-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates build-base

# Создание рабочей директории
WORKDIR /src

# Копируем модули
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /src/bin/external-payment-service ./cmd/external-payment-service

FROM alpine:3.23

# Копируем бинарник и env
COPY --from=builder /src/bin/external-payment-service /usr/local/bin/
COPY --from=builder /src/.env.example .

# Настройки окружения
ENV APP_ENV=example \
    PORT=8081

# Открываем порт
EXPOSE $PORT

CMD ["external-payment-service"]
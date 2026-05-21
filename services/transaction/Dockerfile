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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /src/bin/transaction-service ./cmd/transaction-service

# Миграции
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /src/bin/migrate ./cmd/migrate

FROM alpine:3.23

# Копируем бинарник, миграции и env
COPY --from=builder /src/bin/transaction-service /usr/local/bin/
COPY --from=builder /src/bin/migrate /usr/local/bin/
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/.env.example .
COPY --from=builder /src/iso8583.example.json .


# Настройки окружения
ENV APP_ENV=example \
    PORT=50058

# Открываем порт
EXPOSE $PORT

ENTRYPOINT [ "/bin/sh", "-c", \
    "echo 'Running migrations...' && \
    migrate && \
    echo 'Starting transaction service...' && \
    transaction-service" \
]
FROM golang:1.23-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /src/bin/notification-service ./cmd/notification-service

FROM alpine:3.21

# Копируем бинарник
COPY --from=builder /src/bin/notification-service /usr/local/bin/

# Настройки окружения
ENV APP_ENV=development \
    PORT=8082

# Открываем порт
EXPOSE $PORT

CMD ["notification-service"]
package main

import (
	"os"
	app "transaction-service/internal/app"
	"transaction-service/internal/config"

	"log"
)

const (
	example     = "example"
	development = "development"
	production  = "production"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = example // значение по умолчанию (например, production)
	}

	// Загружаем конфиг
	cfg, err := config.LoadConfig(env)
	if err != nil {
		log.Fatalf("Failed to load config %v", err)
		return
	}

	// Создаем сервер
	server, err := app.NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Запускаем сервер
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

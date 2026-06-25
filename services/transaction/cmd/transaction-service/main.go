package main

import (
	"log/slog"
	"os"
	app "transaction-service/internal/app"
	"transaction-service/internal/config"
	"transaction-service/internal/lib/logger"
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
		slog.Error("Failed to load config", "error", err)
		return
	}

	log := logger.New(logger.Config{
		Env:        cfg.Env,
		Level:      slog.LevelInfo,
		LogFile:    cfg.Logs.LogFile,
		MaxSize:    cfg.Logs.MaxSize,
		MaxBackups: cfg.Logs.MaxBackups,
		MaxAge:     cfg.Logs.MaxAge,
		AddSource:  cfg.Logs.AddSource,
	})

	// Создаем сервер
	server, err := app.NewApp(cfg, log)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
	}

	// Запускаем сервер
	if err := server.Run(); err != nil {
		slog.Error("Server error", "error", err)
	}
}

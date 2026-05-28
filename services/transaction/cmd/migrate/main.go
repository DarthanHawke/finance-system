package main

import (
	"log/slog"
	"os"
	"transaction-service/internal/config"
	"transaction-service/internal/lib/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	example     = "example"
	development = "development"
	production  = "production"
)

// RunMigrations применяет все pending миграции
func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = example
	}
	slog.Info("ENV applied", "env", env)

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

	// Инициализируем мигратор
	mgrt, err := migrate.New("file://./migrations", cfg.DataBase.DSN())
	if err != nil {
		log.Error("migrate init failed", "error", err)
		return
	}
	defer mgrt.Close()

	// Применяем миграции
	if err := mgrt.Up(); err != nil && err != migrate.ErrNoChange {
		log.Error("migrate up failed", "error", err)
	}

}

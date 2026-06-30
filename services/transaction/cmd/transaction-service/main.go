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
		env = example
	}

	// Загружаем конфиг
	cfg, err := config.LoadConfig(env)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
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

	log.Info("Configuration:",
		"env", cfg.Env,
		"grpc_port", cfg.GRPCServer.Port,
		"db_host", cfg.DataBase.Host,
		"db_name", cfg.DataBase.Name,
		"redis_addr", cfg.Redis.Addr,
		"kafka_brokers", cfg.Kafka.Brokers,
		"kafka_concurrency", cfg.Kafka.Concurrency,
		"kafka_consumer_group", cfg.Kafka.ConsumerGroupID,
		"kafka_producer_topic", cfg.Kafka.ProducerTopic,
		"kafka_dlq_topic", cfg.Kafka.DLQTopic,
	)

	// Создаем сервер
	server, err := app.NewApp(cfg, log)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Запускаем сервер
	if err := server.Run(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpcapp "account-service/internal/app/grpc"
	config "account-service/internal/config"
	iban "account-service/internal/lib/iban"
	postgres "account-service/internal/repository/postgres"
	redis "account-service/internal/repository/redis"
	service "account-service/internal/service/account"
	consumer "account-service/internal/transport/kafka/consumer"
	processor "account-service/internal/transport/kafka/outbox"
	producer "account-service/internal/transport/kafka/producer"

	"go.uber.org/zap"
)

// App управляет всеми компонентами сервиса
type App struct {
	logger          *zap.Logger
	config          *config.Configuration
	postgres        *postgres.Database
	redis           *redis.Client
	grpcApp         *grpcapp.App
	consumer        *consumer.Consumer
	producer        *producer.Producer
	outboxProcessor *processor.OutboxProcessor
}

// NewApp создает все компоненты и настраивает зависимости
func NewApp(cfg *config.Configuration) (*App, error) {
	logger, err := createLogger(cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	logger.Info("initializing server",
		zap.String("env", cfg.Env),
		zap.Int("port", cfg.GRPCServer.Port),
	)

	ibanManager, err := iban.NewIBANGenerator("ru", "1337", 25)
	if err != nil {
		logger.Error("Failed to init iban", zap.Error(err))
		return nil, fmt.Errorf("ailed to init iban: %w", err)
	}

	dataBase, err := postgres.NewDatabase(cfg.DataBase.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	logger.Info("database connected")

	redisClient, err := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		dataBase.Close()
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	logger.Info("Redis connection success", zap.String("Addres", cfg.Redis.Addr))

	deduplicator := redis.NewDeduplicator(redisClient, 24*time.Hour)

	accountRepository := postgres.NewAccountRepository(dataBase, logger)
	eventRepository := postgres.NewEventRepository(dataBase, logger)

	accountService := service.NewAccountService(
		accountRepository,
		ibanManager,
		logger,
	)

	balanceService := service.NewBalanceService(
		accountRepository,
		eventRepository,
		logger,
	)

	producer := producer.NewProducer(
		&producer.Config{
			Brokers:      splitBrokers(cfg.Kafka.Brokers),
			Topic:        cfg.Kafka.ProducerTopic,
			BatchSize:    cfg.Kafka.ProducerBatchSize,
			BatchTimeout: time.Duration(cfg.Kafka.ProducerBatchTimeout) * time.Millisecond,
			RequiredAcks: cfg.Kafka.ProducerRequiredAcks,
			MaxAttempts:  cfg.Kafka.ProducerMaxAttempts,
			WriteTimeout: time.Duration(cfg.Kafka.ProducerWriteTimeout) * time.Millisecond,
		},
		&producer.DLQConfig{
			Brokers:      splitBrokers(cfg.Kafka.Brokers),
			Topic:        cfg.Kafka.DLQTopic,
			BatchSize:    cfg.Kafka.DLQBatchSize,
			BatchTimeout: time.Duration(cfg.Kafka.DLQBatchTimeout) * time.Millisecond,
			MaxAttempts:  cfg.Kafka.DLQMaxAttempts,
		},
		&producer.RetryConfig{
			MaxAttempts: cfg.Kafka.RetryMaxAttempts,
			InitialWait: time.Duration(cfg.Kafka.RetryInitialWait) * time.Millisecond,
			MaxWait:     time.Duration(cfg.Kafka.RetryMaxWait) * time.Millisecond,
			Multiplier:  cfg.Kafka.RetryMultiplier,
		},
		logger,
	)

	consumer := consumer.NewConsumer(
		splitTopics(cfg.Kafka.ConsumerTopics),
		&consumer.Config{
			Brokers:          splitBrokers(cfg.Kafka.Brokers),
			GroupID:          cfg.Kafka.ConsumerGroupID,
			MinBytes:         cfg.Kafka.ConsumerMinBytes,
			MaxBytes:         cfg.Kafka.ConsumerMaxBytes,
			MaxWait:          time.Duration(cfg.Kafka.ConsumerMaxWait) * time.Millisecond,
			CommitInterval:   time.Duration(cfg.Kafka.ConsumerCommitInterval) * time.Millisecond,
			SessionTimeout:   time.Duration(cfg.Kafka.ConsumerSessionTimeout) * time.Millisecond,
			RebalanceTimeout: time.Duration(cfg.Kafka.ConsumerRebalanceTimeout) * time.Millisecond,
			StartOffset:      cfg.Kafka.ConsumerStartOffset,
			Concurrency:      cfg.Kafka.ConsumerConcurrency,
		},
		balanceService,
		deduplicator,
		logger,
	)

	outboxProcessor := processor.NewOutboxProcessor(
		eventRepository,
		producer,
		logger,
		&processor.Config{
			BatchSize:    100,
			HandlePeriod: 1 * time.Second,
		},
	)

	grpcApp := grpcapp.New(
		logger,
		cfg.GRPCServer.Port,
		accountService,
	)
	return &App{
		logger:          logger,
		config:          cfg,
		postgres:        dataBase,
		redis:           redisClient,
		grpcApp:         grpcApp,
		consumer:        consumer,
		producer:        producer,
		outboxProcessor: outboxProcessor,
	}, nil
}

// Run запускает все компоненты сервера
func (app *App) Run() error {
	defer app.shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.logger.Info("starting server")

	// запускаем gRPC сервер
	go app.grpcApp.MustRun()

	// запускаем Kafka consumer
	app.consumer.Start(ctx)

	// запускаем outbox processor
	app.outboxProcessor.StartProcessEvents(ctx)

	app.logger.Info("server started successfully")

	// ждем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		app.logger.Info("received shutdown signal")
	case <-ctx.Done():
		app.logger.Info("context cancelled")
	}

	// останавливаем gRPC
	app.grpcApp.Stop()
	app.logger.Info("grpc server stopped")

	// в сулчае нормального завершения сразу отправляем отмену для consumer и outbox,
	// чтоб не висели в таймауте в ожидании defer
	cancel()

	// останавливаем consumer
	if err := app.consumer.Stop(); err != nil {
		app.logger.Error("failed to stop consumer", zap.Error(err))
	}

	app.logger.Info("server stopped gracefully")
	return nil
}

func (app *App) shutdown() {
	app.logger.Info("shutting down resources")

	if app.producer != nil {
		if err := app.producer.Close(); err != nil {
			app.logger.Error("failed to close producer", zap.Error(err))
		}
	}

	if app.redis != nil {
		if err := app.redis.Close(); err != nil {
			app.logger.Error("failed to close redis", zap.Error(err))
		}
	}

	if app.postgres != nil {
		if err := app.postgres.Close(); err != nil {
			app.logger.Error("failed to close database", zap.Error(err))
		}
	}

	app.logger.Info("resources released")
}

func splitBrokers(brokers string) []string {
	return strings.Split(brokers, ",")
}

func splitTopics(topics string) []string {
	parts := strings.Split(topics, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func createLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

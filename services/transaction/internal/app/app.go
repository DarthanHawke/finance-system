package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpc "transaction-service/internal/app/grpc"
	config "transaction-service/internal/config"
	iso8583 "transaction-service/internal/lib/iso8583"
	stan "transaction-service/internal/lib/stan"
	postgres "transaction-service/internal/repository/postgres"
	redis "transaction-service/internal/repository/redis"
	service "transaction-service/internal/service/transaction"
	consumer "transaction-service/internal/transport/kafka/consumer"
	processor "transaction-service/internal/transport/kafka/outbox"
	producer "transaction-service/internal/transport/kafka/producer"
)

// App управляет всеми компонентами сервиса
type App struct {
	logger          *slog.Logger
	postgres        *postgres.Database
	redis           *redis.Client
	grpc            *grpc.App
	consumer        *consumer.Consumer
	producer        *producer.Producer
	outboxProcessor *processor.OutboxProcessor
	stanManager     *stan.STAN
}

// NewApp создает все компоненты и настраивает зависимости
func NewApp(cfg *config.Configuration, iso8583cfg *config.ISO8583Config, log *slog.Logger) (*App, error) {
	log.Info("initializing server", "env", cfg.Env, "port", cfg.GRPCServer.Port)

	dataBase, err := postgres.NewDatabase(cfg.DataBase.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	log.Info("database connected")

	redisClient, err := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		dataBase.Close()
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	log.Info("Redis connection success", "address", cfg.Redis.Addr)

	cache := redis.NewCache(redisClient, 10*time.Minute)
	deduplicator := redis.NewDeduplicator(redisClient, 24*time.Hour)

	stanManager := stan.NewSTAN()
	iso8583Manager := iso8583.NewISO8583(iso8583cfg)

	transactionRepository := postgres.NewTransactionRepository(dataBase, cache)
	eventRepository := postgres.NewEventRepository(dataBase)

	operationService := service.NewOperationService(
		transactionRepository,
	)

	transactionService := service.NewTransactionService(
		transactionRepository,
		eventRepository,
		iso8583Manager,
		stanManager,
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
		log,
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
		transactionService,
		deduplicator,
		log,
	)

	outboxProcessor := processor.NewOutboxProcessor(
		eventRepository,
		producer,
		log,
		&processor.Config{
			BatchSize:    cfg.Kafka.ProcessorBatchSize,
			HandlePeriod: time.Duration(cfg.Kafka.ProcessorHandlePeriod) * time.Millisecond,
			Concurrency:  cfg.Kafka.Concurrency,
		},
	)

	grpc := grpc.New(
		log,
		cfg.GRPCServer.Port,
		operationService,
	)
	return &App{
		logger:          log,
		postgres:        dataBase,
		redis:           redisClient,
		grpc:            grpc,
		consumer:        consumer,
		producer:        producer,
		outboxProcessor: outboxProcessor,
		stanManager:     stanManager,
	}, nil
}

// Run запускает все компоненты сервера
func (app *App) Run() error {
	defer app.shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.logger.Info("starting server")

	// запускаем gRPC сервер
	go app.grpc.MustRun()

	// запускаем Kafka consumer
	app.consumer.Run(ctx)

	// запускаем outbox processor
	outboxDone := make(chan struct{})
	go func() {
		defer close(outboxDone)
		app.outboxProcessor.Run(ctx)
	}()

	// запускаем планировщик сброса счётчиков STAN
	stanDone := make(chan struct{})
	go func() {
		defer close(stanDone)
		app.runSTANResetScheduler(ctx)
	}()

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
	app.grpc.Stop()
	app.logger.Info("grpc server stopped")

	// в сулчае нормального завершения сразу отправляем отмену для consumer и outbox,
	// чтоб не висели в таймауте в ожидании defer
	cancel()

	// ждем завершения outbox processor
	<-outboxDone
	app.logger.Info("outbox processor stopped")

	// ждем завершения STAN scheduler
	<-stanDone
	app.logger.Info("stan scheduler stopped")

	// останавливаем consumer
	if err := app.consumer.Stop(); err != nil {
		app.logger.Error("failed to stop consumer", "error", err)
	}

	app.logger.Info("server stopped gracefully")
	return nil
}

func (app *App) shutdown() {
	app.logger.Info("shutting down resources")

	if app.producer != nil {
		if err := app.producer.Close(); err != nil {
			app.logger.Error("failed to close producer", "error", err)
		}
	}

	if app.redis != nil {
		if err := app.redis.Close(); err != nil {
			app.logger.Error("failed to close redis", "error", err)
		}
	}

	if app.postgres != nil {
		if err := app.postgres.Close(); err != nil {
			app.logger.Error("failed to close database", "error", err)
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

// runSTANResetScheduler сбрасывает счётчики STAN каждый день в полночь
func (app *App) runSTANResetScheduler(ctx context.Context) {
	app.logger.Info("starting STAN reset scheduler")

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

		select {
		case <-time.After(next.Sub(now)):
			app.stanManager.ResetCounters() // нужен доступ к stanManager
			app.logger.Info("STAN counters reset")
		case <-ctx.Done():
			app.logger.Info("STAN reset scheduler stopped")
			return
		}
	}
}

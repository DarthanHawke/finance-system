package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	config "transaction-service/internal/config"
	tracing "transaction-service/internal/lib/tracing"
	postgres "transaction-service/internal/repository/postgres"
	redis "transaction-service/internal/repository/redis"
	saga "transaction-service/internal/saga"
	consumer "transaction-service/internal/transport/kafka/consumer"
	handlers "transaction-service/internal/transport/kafka/handlers"
	processor "transaction-service/internal/transport/kafka/outbox"
	producer "transaction-service/internal/transport/kafka/producer"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

// App управляет всеми компонентами сервиса
type App struct {
	logger          *slog.Logger
	postgres        *postgres.Database
	redis           *redis.Client
	consumer        *consumer.Consumer
	producer        *producer.Producer
	outboxProcessor *processor.OutboxProcessor
	tracerProvider  *trace.TracerProvider
	metricsServer   *http.Server
}

// NewApp создает все компоненты и настраивает зависимости
func NewApp(cfg *config.Configuration, log *slog.Logger) (*App, error) {
	log.Info("initializing server", "env", cfg.Env, "port", cfg.GRPCServer.Port)

	tracer, err := tracing.InitTracer(
		context.Background(),
		"transaction-service",
		cfg.Tracing.OTLPEndpoint,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

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

	deduplicator := redis.NewDeduplicator(redisClient, 24*time.Hour)

	transactionRepository := postgres.NewTransactionRepository(dataBase)
	sagaRepository := postgres.NewSagaRepository(dataBase)
	eventRepository := postgres.NewEventRepository(dataBase)

	sagaCoordinator := postgres.NewSagaCoordinator(dataBase, transactionRepository, sagaRepository, eventRepository)
	sagaOrchestrator := saga.NewOrchestrator(transactionRepository, sagaCoordinator, saga.Register())
	eventHandlers := handlers.New(sagaOrchestrator)

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
		eventHandlers,
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

	metric := &http.Server{
		Addr:    ":9090",
		Handler: promhttp.Handler(),
	}

	return &App{
		logger:          log,
		postgres:        dataBase,
		redis:           redisClient,
		consumer:        consumer,
		producer:        producer,
		outboxProcessor: outboxProcessor,
		tracerProvider:  tracer,
		metricsServer:   metric,
	}, nil
}

// Run запускает все компоненты сервера
func (app *App) Run() error {
	defer app.shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.logger.Info("starting server")

	// запускаем http сервер для prometheus
	go app.runMetricServer()

	// запускаем Kafka consumer
	app.consumer.Run(ctx)

	// запускаем outbox processor
	outboxDone := make(chan struct{})
	go func() {
		defer close(outboxDone)
		app.outboxProcessor.Run(ctx)
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

	app.stopMetricServer()
	app.logger.Info("http prometheus server stopped")

	// в сулчае нормального завершения сразу отправляем отмену для consumer и outbox,
	// чтоб не висели в таймауте в ожидании defer
	cancel()

	// ждем завершения outbox processor
	<-outboxDone
	app.logger.Info("outbox processor stopped")

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

	if app.tracerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.tracerProvider.Shutdown(ctx); err != nil {
			app.logger.Error("failed to shutdown tracer", "error", err)
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

func (app *App) runMetricServer() {
	app.logger.Info("starting metrics server", "port", 9090)
	if err := app.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.logger.Error("metrics server failed", "error", err)
	}
}

func (app *App) stopMetricServer() {
	if app.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.metricsServer.Shutdown(ctx); err != nil {
			app.logger.Error("failed to shutdown metrics server", "error", err)
		}
	}
}

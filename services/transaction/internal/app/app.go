package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpcapp "transaction-service/internal/app/grpc"
	cache "transaction-service/internal/cache"
	config "transaction-service/internal/config"
	iso8583 "transaction-service/internal/lib/iso8583"
	stan "transaction-service/internal/lib/stan"
	repository "transaction-service/internal/repository"
	processor "transaction-service/internal/service/event"
	service "transaction-service/internal/service/transaction"
	consumer "transaction-service/internal/transport/kafka/consumer"
	producer "transaction-service/internal/transport/kafka/producer"

	"go.uber.org/zap"
)

// App управляет всеми компонентами сервиса
type App struct {
	logger *zap.Logger
	config *config.Configuration

	dataBase *repository.Database

	// Транспорт
	grpcApp  *grpcapp.App
	consumer *consumer.Consumer
	producer *producer.Producer

	// Процессы
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

	dataBase, err := repository.NewDatabase(cfg.DataBase.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	logger.Info("database connected")

	redisCache := cache.NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, 10*time.Minute)
	defer redisCache.Close()
	logger.Info("Redis connection success", zap.String("Addres", cfg.Redis.Addr))

	iso8583config, err := config.LoadISO8583Config(cfg.ISO8583ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load ISO8583 config: %w", err)
	}
	stanManager := stan.NewSTAN()
	iso8583Manager := iso8583.NewISO8583(iso8583config, stanManager)

	transactionRepository := repository.NewTransactionRepository(dataBase, redisCache, logger)
	eventRepository := repository.NewEventRepository(dataBase, logger)

	operationService := service.NewOperationService(
		transactionRepository,
		logger,
	)

	transactionService := service.NewTransactionService(
		transactionRepository,
		eventRepository,
		iso8583Manager,
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
		transactionService,
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

	tlsConfig, err := createTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create tls config: %w", err)
	}

	grpcApp := grpcapp.New(
		logger,
		cfg.GRPCServer.Port,
		tlsConfig,
		operationService,
	)
	return &App{
		logger:          logger,
		config:          cfg,
		dataBase:        dataBase,
		grpcApp:         grpcApp,
		consumer:        consumer,
		producer:        producer,
		outboxProcessor: outboxProcessor,
	}, nil
}

// Run запускает все компоненты сервера
func (app *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.logger.Info("starting server")

	// Запускаем gRPC сервер
	go app.grpcApp.MustRun()

	// Запускаем Kafka consumer
	app.consumer.Start(ctx)

	// Запускаем outbox processor
	app.outboxProcessor.StartProcessEvents(ctx)

	app.logger.Info("server started successfully")

	// Ждем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.logger.Info("received shutdown signal")

	// Отменяем контекст для остановки consumer и outbox
	cancel()

	// Останавливаем gRPC сервер
	app.grpcApp.Stop()
	app.logger.Info("grpc server stopped")

	// Ждем завершения consumer (с таймаутом)
	if err := app.consumer.Stop(); err != nil {
		app.logger.Error("failed to stop consumer", zap.Error(err))
	}

	// Закрываем producer
	if err := app.producer.Close(); err != nil {
		app.logger.Error("failed to close producer", zap.Error(err))
	}

	// Закрываем базу данных
	if err := app.dataBase.Close(); err != nil {
		app.logger.Error("failed to close database", zap.Error(err))
	}

	app.logger.Info("server stopped gracefully")
	return nil
}

// Вспомогательные функции

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

func createTLSConfig(cfg *config.Configuration) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(cfg.CA)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("append ca certs")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}, nil
}

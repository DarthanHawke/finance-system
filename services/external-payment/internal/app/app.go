package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"external-payment-service/internal/config"
	"external-payment-service/internal/service"
	httphandler "external-payment-service/internal/transport/http"
	consumer "external-payment-service/internal/transport/kafka/consumer"
	producer "external-payment-service/internal/transport/kafka/producer"

	"go.uber.org/zap"
)

type App struct {
	logger   *zap.Logger
	config   *config.Configuration
	consumer *consumer.Consumer
	producer *producer.Producer
	httpSrv  *http.Server
}

func NewApp(cfg *config.Configuration) (*App, error) {
	logger, err := createLogger(cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	logger.Info("initializing external payment service",
		zap.String("env", cfg.Env),
		zap.Int("http_port", cfg.HTTPServer.Port),
	)

	// Producer
	producer := producer.NewProducer(
		&producer.Config{
			Brokers:      splitBrokers(cfg.Kafka.Brokers),
			BatchSize:    cfg.Kafka.ProducerBatchSize,
			BatchTimeout: time.Duration(cfg.Kafka.ProducerBatchTimeout) * time.Millisecond,
			MaxAttempts:  cfg.Kafka.ProducerMaxAttempts,
		},
		&producer.RetryConfig{
			MaxAttempts: cfg.Kafka.RetryMaxAttempts,
			InitialWait: time.Duration(cfg.Kafka.RetryInitialWait) * time.Millisecond,
			MaxWait:     time.Duration(cfg.Kafka.RetryMaxWait) * time.Millisecond,
			Multiplier:  cfg.Kafka.RetryMultiplier,
		},
		logger,
	)

	// Service
	svc := service.NewExternalPaymentService(producer, logger)

	// Consumer
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
		},
		svc,
		logger,
	)

	// HTTP сервер
	httpHandler := httphandler.NewExternalPaymentHandler(svc, logger)
	mux := http.NewServeMux()
	httpHandler.RegisterRoutes(mux)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler: mux,
	}

	return &App{
		logger:   logger,
		config:   cfg,
		consumer: consumer,
		producer: producer,
		httpSrv:  httpSrv,
	}, nil
}

func (app *App) Run() error {
	defer app.shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.logger.Info("starting server")

	// Kafka consumer
	app.consumer.Start(ctx)

	// HTTP сервер
	go func() {
		app.logger.Info("starting HTTP server", zap.String("addr", app.httpSrv.Addr))
		if err := app.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	app.logger.Info("server started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		app.logger.Info("received shutdown signal")
	case <-ctx.Done():
		app.logger.Info("context cancelled")
	}

	// Останавливаем HTTP
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := app.httpSrv.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("HTTP shutdown error", zap.Error(err))
	}
	app.logger.Info("HTTP server stopped")

	cancel()

	// Останавливаем consumer
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
	app.logger.Info("resources released")
}

func createLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
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

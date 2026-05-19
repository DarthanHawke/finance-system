package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"api-gateway-service/internal/config"
	redisclient "api-gateway-service/internal/repository/redis"
	accountclient "api-gateway-service/internal/transport/grpc/client/account"
	transactionclient "api-gateway-service/internal/transport/grpc/client/transaction"
	accounthandler "api-gateway-service/internal/transport/http/handlers/account"
	transactionhandler "api-gateway-service/internal/transport/http/handlers/transaction"
	kafkaproducer "api-gateway-service/internal/transport/kafka/producer"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// App управляет всеми компонентами сервиса
type App struct {
	logger   *zap.Logger
	config   *config.Configuration
	server   *http.Server
	producer *kafkaproducer.Producer
	redis    *redisclient.Client
}

// NewApp создает все компоненты и настраивает зависимости
func NewApp(cfg *config.Configuration) (*App, error) {
	logger, err := createLogger(cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	logger.Info("initializing api-gateway",
		zap.String("env", cfg.Env),
		zap.Int("port", cfg.HTTPServer.Port),
	)

	tlsConfig, err := createTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create tls config: %w", err)
	}

	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	logger.Info("redis connected", zap.String("addr", cfg.Redis.Addr))

	accountClient, err := accountclient.NewAccountClient(
		context.Background(),
		logger,
		cfg.AccountClient.Address,
		cfg.AccountClient.Timeout,
		cfg.AccountClient.RetriesCount,
		tlsConfig,
	)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("create account client: %w", err)
	}
	logger.Info("account grpc client created")

	transactionClient, err := transactionclient.NewTransactionClient(
		context.Background(),
		logger,
		cfg.TransactionClient.Address,
		cfg.TransactionClient.Timeout,
		cfg.TransactionClient.RetriesCount,
		tlsConfig,
	)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("create transaction client: %w", err)
	}
	logger.Info("transaction grpc client created")

	producer := kafkaproducer.NewProducer(
		&kafkaproducer.Config{
			Brokers:      splitBrokers(cfg.Kafka.Brokers),
			Topic:        cfg.Kafka.ProducerTopic,
			BatchSize:    cfg.Kafka.ProducerBatchSize,
			BatchTimeout: time.Duration(cfg.Kafka.ProducerBatchTimeout) * time.Millisecond,
			RequiredAcks: cfg.Kafka.ProducerRequiredAcks,
			MaxAttempts:  cfg.Kafka.ProducerMaxAttempts,
			WriteTimeout: time.Duration(cfg.Kafka.ProducerWriteTimeout) * time.Millisecond,
		},
		&kafkaproducer.RetryConfig{
			MaxAttempts: cfg.Kafka.RetryMaxAttempts,
			InitialWait: time.Duration(cfg.Kafka.RetryInitialWait) * time.Millisecond,
			MaxWait:     time.Duration(cfg.Kafka.RetryMaxWait) * time.Millisecond,
			Multiplier:  cfg.Kafka.RetryMultiplier,
		},
		logger,
	)
	logger.Info("kafka producer created")

	accountHandler := accounthandler.NewAccountHandler(accountClient)

	deduplicator := redisclient.NewDeduplicator(redisClient, 24*time.Hour)
	transactionHandler := transactionhandler.NewTransactionHandler(
		transactionClient,
		producer,
		deduplicator,
		logger,
	)

	router := chi.NewRouter()
	router.Mount("/account", accountHandler.Routes())
	router.Mount("/transaction", transactionHandler.Routes())

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		logger:   logger,
		config:   cfg,
		server:   httpServer,
		producer: producer,
		redis:    redisClient,
	}, nil
}

// Run запускает HTTP сервер и ждет сигнала для graceful shutdown
func (app *App) Run() error {
	defer app.shutdown()

	// Запускаем HTTP сервер в горутине
	go func() {
		app.logger.Info("starting http server", zap.Int("port", app.config.HTTPServer.Port))
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Fatal("http server failed", zap.Error(err))
		}
	}()

	app.logger.Info("api-gateway started successfully")

	// Ждем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	app.logger.Info("received shutdown signal")

	// Даем 10 секунд на завершение активных запросов
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.server.Shutdown(ctx); err != nil {
		app.logger.Error("http server forced to shutdown", zap.Error(err))
		return err
	}

	app.logger.Info("http server stopped gracefully")
	return nil
}

// shutdown освобождает все ресурсы
func (app *App) shutdown() {
	app.logger.Info("shutting down resources")

	if app.producer != nil {
		if err := app.producer.Close(); err != nil {
			app.logger.Error("failed to close kafka producer", zap.Error(err))
		}
	}

	if app.redis != nil {
		if err := app.redis.Close(); err != nil {
			app.logger.Error("failed to close redis", zap.Error(err))
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

func createTLSConfig(cfg *config.Configuration) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLS.TLSCert, cfg.TLS.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(cfg.TLS.CA)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("append ca certs")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ClientCAs:    certPool,
	}, nil
}

func splitBrokers(brokers string) []string {
	return strings.Split(brokers, ",")
}

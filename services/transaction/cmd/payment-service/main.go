package main

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"
	"time"
	transactionapp "transaction-service/internal/app"
	"transaction-service/internal/cache"
	"transaction-service/internal/config"
	"transaction-service/internal/events"
	"transaction-service/internal/lib/logger"
	"transaction-service/internal/storage"

	"go.uber.org/zap"
)

const (
	example     = "example"
	development = "development"
	production  = "production"
)

func main() {
	// Подключаем логгер Zap в настраиваемой конфигурации (>=LevelInfo выводит в консоль, >=DebugLevel в файл)
	log := logger.SetupLogger()
	defer log.Sync()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = example // значение по умолчанию (например, production)
	}
	log.Info("ENV applied", zap.String("env", env))

	// Загружаем конфиг
	cfg, err := config.LoadConfig(env)
	if err != nil {
		log.Error("Failed to load config", zap.Error(err))
		return
	}

	// Загрузка сертификата сервера и приватного ключа
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		log.Error("failed to load key pair: %s", zap.Error(err))
	}
	// Создание пула сертификатов и добавление CA
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(cfg.CA)
	if err != nil {
		log.Error("could not read ca certificate: %s", zap.Error(err))
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Error("failed to append ca certs")
	}

	// Создание TLS конфига
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	// Подключаемся к БД
	db, err := storage.NewDatabase(cfg.DataBase.DSN())
	if err != nil {
		log.Error("DB connection failed", zap.Error(err), zap.String("dsn", cfg.DataBase.DSN()))
		return
	}
	log.Info("DB connection success", zap.String("dsn", cfg.DataBase.DSN()))
	defer db.CloseConnect()

	// Подключаем Redis
	redisCache := cache.NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, 10*time.Minute)
	defer redisCache.Close()
	log.Info("Redis connection success", zap.String("Addres", cfg.Redis.Addr))

	// Подключаем RabbitMQ
	rabbitConn, err := events.NewRabbitMQEventSender(cfg.RabbitMQ.URL, cfg.RabbitMQ.QueueName, log)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitConn.Close()
	log.Info("RabbitMQ connection success", zap.String("URL", cfg.RabbitMQ.URL))

	// Канал для graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// инициализируем сервер
	application := transactionapp.New(log, cfg.GRPSServer.Port, tlsConfig, cfg.DataBase.DSN(), redisCache, rabbitConn)

	// запускаем gRPC сервер
	go application.GRPCServer.MustRun()

	// Ожидание сигнала завершения
	<-done

	// Graceful shutdown
	application.GRPCServer.Stop()
	log.Info("Gracefully stopped")
}

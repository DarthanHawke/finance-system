package main

import (
	billingapp "billing-service/internal/app"
	"billing-service/internal/config"
	"billing-service/internal/lib/iban"
	"billing-service/internal/lib/jwt"
	"billing-service/internal/lib/logger"
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"

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

	// JWT
	jwtManager, err := jwt.NewTokenValidator(
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.Issuer,
		cfg.JWTPublicKeyPath,
	)
	if err != nil {
		log.Error("Failed to init JWT", zap.Error(err))
		return
	}

	ibanManager, err := iban.NewIBANGenerator("ru", "1337", 25)
	if err != nil {
		log.Error("Failed to init iban", zap.Error(err))
		return
	}
	// Канал для graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	systemUsers, err := config.LoadSystemUsers(env)
	if err != nil {
		log.Error("System users not found", zap.Error(err))
	}

	// инициализируем сервер
	application := billingapp.New(
		log,
		cfg.GRPSServer.Port,
		tlsConfig,
		cfg.DataBase.DSN(),
		&cfg.SSOClient,
		&cfg.PaymentClient,
		&cfg.Redis,
		jwtManager,
		ibanManager,
		systemUsers,
	)

	// запускаем gRPC сервер
	go application.GRPCServer.MustRun()

	// Ожидание сигнала завершения
	<-done

	// Graceful shutdown
	application.GRPCServer.Stop()
	log.Info("Gracefully stopped")
}

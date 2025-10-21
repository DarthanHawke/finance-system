package main

import (
	clientapp "client-service/internal/app"
	"client-service/internal/config"
	"context"
	"time"

	"client-service/internal/logger"
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

// @title Transaction API
// @version 1.0
// @description API для платежной системы
// @host localhost
// @BasePath /
// @schemes https
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name access_token
// @description Аутентификация через куки (токен автоматически отправляется сервером)
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

	// Канал для graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	application := clientapp.New(
		log,
		cfg.Port,
		tlsConfig,
		&cfg.BillingClient,
	)

	go application.HTTPServer.MustRun()

	// Ожидание сигнала завершения
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := application.HTTPServer.Stop(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
	log.Info("Gracefully stopped")
}

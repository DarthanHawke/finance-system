package paymentapp

import (
	"crypto/tls"
	grpcapp "payment-service/internal/app/grpc"
	"payment-service/internal/cache"
	"payment-service/internal/events"
	"payment-service/internal/service/payment"
	"payment-service/internal/storage"

	"go.uber.org/zap"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	logger *zap.Logger,
	grpcPort int,
	tlsConfig *tls.Config,
	storagePath string,
	cache *cache.RedisCache,
	eventSender *events.RabbitMQEventSender,
) *App {
	// Подключаемся к БД
	dataBase, err := storage.NewDatabase(storagePath)
	if err != nil {
		panic(err)
	}

	paymentDataBase := storage.NewPaymentDataBase(dataBase, cache)
	paymentService := payment.NewPaymentService(paymentDataBase, eventSender, logger)

	_ = dataBase
	gRPCApp := grpcapp.New(logger, grpcPort, tlsConfig, paymentService)

	return &App{
		GRPCServer: gRPCApp,
	}
}

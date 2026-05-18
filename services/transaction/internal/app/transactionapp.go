package transactionapp

import (
	"crypto/tls"
	grpcapp "transaction-service/internal/app/grpc"
	"transaction-service/internal/cache"
	"transaction-service/internal/repository"
	transaction "transaction-service/internal/service/transaction"

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
) *App {
	// Подключаемся к БД
	dataBase, err := repository.NewDatabase(storagePath)
	if err != nil {
		panic(err)
	}

	transactionDataBase := repository.NewTransactionRepository(dataBase, cache, logger)
	operationService := transaction.NewOperationService(transactionDataBase, logger)

	_ = dataBase
	gRPCApp := grpcapp.New(logger, grpcPort, tlsConfig, operationService)

	return &App{
		GRPCServer: gRPCApp,
	}
}

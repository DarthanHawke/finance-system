package transactionapp

/*
import (
	"crypto/tls"
	grpcapp "transaction-service/internal/app/grpc"
	"transaction-service/internal/cache"
	"transaction-service/internal/events"
	"transaction-service/internal/service/transaction"
	"transaction-service/internal/storage"

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

	transactionDataBase := storage.NewTransactionDataBase(dataBase, cache)
	transactionService := transaction.NewTransactionService(transactionDataBase, eventSender, logger)

	_ = dataBase
	gRPCApp := grpcapp.New(logger, grpcPort, tlsConfig, transactionService)

	return &App{
		GRPCServer: gRPCApp,
	}
}
*/

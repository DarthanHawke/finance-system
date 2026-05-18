package ssoapp

/* legacy code
import (
	"crypto/tls"
	grpcapp "rebac-service/internal/app/grpc"
	"rebac-service/internal/service/role"
	"rebac-service/internal/storage"

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
) *App {
	// Подключаемся к БД
	dataBase, err := storage.NewDatabase(storagePath)
	if err != nil {
		panic(err)
	}
	roleDataBase := storage.NewRoleDataBase(dataBase)
	roleService := role.NewRoleService(logger, roleDataBase, roleDataBase)

	_ = dataBase
	gRPCApp := grpcapp.New(logger, grpcPort, tlsConfig, roleService)

	return &App{
		GRPCServer: gRPCApp,
	}
}
*/

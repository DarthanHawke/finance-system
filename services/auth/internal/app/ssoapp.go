package ssoapp

/* legacy code
import (
	grpcapp "auth-service/internal/app/grpc"
	"auth-service/internal/lib/hash"
	"auth-service/internal/lib/jwt"
	"auth-service/internal/service/session"
	"auth-service/internal/service/user"
	"auth-service/internal/storage"
	"crypto/tls"

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
	jwtManager *jwt.TokenGenerator,
	hasher *hash.Argon2Hasher,
) *App {
	// Подключаемся к БД
	dataBase, err := storage.NewDatabase(storagePath)
	if err != nil {
		panic(err)
	}
	sessionDataBase := storage.NewSessionDataBase(dataBase)
	userDataBase := storage.NewUserDataBase(dataBase)
	sessionService := session.NewSessionService(logger, sessionDataBase, sessionDataBase, jwtManager, hasher)
	userService := user.NewUserService(logger, userDataBase, userDataBase, userDataBase, hasher)

	_ = dataBase
	gRPCApp := grpcapp.New(logger, grpcPort, tlsConfig, sessionService, userService)

	return &App{
		GRPCServer: gRPCApp,
	}
}
*/

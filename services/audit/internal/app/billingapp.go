package billingapp

/* legacy code
import (
	grpcapp "billing-service/internal/app/grpc"
	"billing-service/internal/cache"
	transactionclient "billing-service/internal/clients/grpc/transaction"
	initialization "billing-service/internal/initialization/role"

	roleclient "billing-service/internal/clients/grpc/sso/role"
	sessionclient "billing-service/internal/clients/grpc/sso/session"
	userclient "billing-service/internal/clients/grpc/sso/user"
	"billing-service/internal/config"
	"billing-service/internal/grpc/interceptor"
	"billing-service/internal/lib/iban"
	"billing-service/internal/lib/jwt"
	"billing-service/internal/service/account"
	"billing-service/internal/service/auth"
	"billing-service/internal/service/role"
	"billing-service/internal/service/transaction"
	"billing-service/internal/service/user"
	"billing-service/internal/storage"
	"context"
	"crypto/tls"
	"time"

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
	ssoClientConfig *config.SSOClient,
	transactionClientConfig *config.TransactionClient,
	redisConfig *config.Redis,
	jwtManager *jwt.TokenValidator,
	ibanManager *iban.IbanGenerator,
	systemUsers *config.SystemUsers,
) *App {
	// Подключаемся к БД
	dataBase, err := storage.NewDatabase(storagePath)
	if err != nil {
		panic(err)
	}
	// Подключаем Redis
	redisCache := cache.NewRedisCache(redisConfig.Addr, redisConfig.Password, redisConfig.DB, 10*time.Minute)

	accountDataBase := storage.NewAccountRepository(dataBase, redisCache)
	transactionDataBase := storage.NewTransactionRepository(dataBase, redisCache)

	sessionClient, _ := sessionclient.NewSessionClient(
		context.Background(),
		logger,
		ssoClientConfig.Address,
		ssoClientConfig.Timeout,
		ssoClientConfig.RetriesCount,
		tlsConfig,
	)
	userClient, _ := userclient.NewUserClient(
		context.Background(),
		logger,
		ssoClientConfig.Address,
		ssoClientConfig.Timeout,
		ssoClientConfig.RetriesCount,
		tlsConfig,
	)
	roleClient, _ := roleclient.NewRoleClient(
		context.Background(),
		logger,
		ssoClientConfig.Address,
		ssoClientConfig.Timeout,
		ssoClientConfig.RetriesCount,
		tlsConfig,
	)
	transactionClient, _ := transactionclient.NewTransactionClient(
		context.Background(),
		logger,
		transactionClientConfig.Address,
		transactionClientConfig.Timeout,
		transactionClientConfig.RetriesCount,
		tlsConfig,
	)

	authInterceptor := interceptor.NewAuthInterceptor(jwtManager, redisCache)
	authService := auth.NewAuthService(userClient, sessionClient, roleClient, redisCache, logger)
	transactionService := transaction.NewTransactionService(accountDataBase, transactionClient, transactionDataBase, roleClient, ibanManager, logger)
	accountService := account.NewAccountService(accountDataBase, roleClient, ibanManager, logger)
	userService := user.NewUserService(userClient, roleClient, logger)
	roleService := role.NewRoleService(roleClient, logger)
	initializer := initialization.NewInitializer(roleClient, userClient, logger)
	initializer.InitSystem(context.Background(), systemUsers.SystemUsers)

	gRPCApp := grpcapp.New(
		logger, grpcPort, tlsConfig, authInterceptor,
		authService, transactionService, accountService, userService, roleService,
	)

	return &App{
		GRPCServer: gRPCApp,
	}
}
*/

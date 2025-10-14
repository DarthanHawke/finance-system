package billingapp

import (
	grpcapp "billing-service/internal/app/grpc"
	"billing-service/internal/cache"
	paymentclient "billing-service/internal/clients/grpc/payment"
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
	"billing-service/internal/service/payment"
	"billing-service/internal/service/role"
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
	paymentClientConfig *config.PaymentClient,
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
	paymentDataBase := storage.NewPaymentRepository(dataBase, redisCache)

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
	paymentClient, _ := paymentclient.NewPaymentClient(
		context.Background(),
		logger,
		paymentClientConfig.Address,
		paymentClientConfig.Timeout,
		paymentClientConfig.RetriesCount,
		tlsConfig,
	)

	authInterceptor := interceptor.NewAuthInterceptor(jwtManager, redisCache)
	authService := auth.NewAuthService(userClient, sessionClient, roleClient, redisCache, logger)
	paymentService := payment.NewPaymentService(accountDataBase, paymentClient, paymentDataBase, roleClient, ibanManager, logger)
	accountService := account.NewAccountService(accountDataBase, roleClient, ibanManager, logger)
	userService := user.NewUserService(userClient, roleClient, logger)
	roleService := role.NewRoleService(roleClient, logger)
	initializer := initialization.NewInitializer(roleClient, userClient, logger)
	initializer.InitSystem(context.Background(), systemUsers.SystemUsers)

	gRPCApp := grpcapp.New(
		logger, grpcPort, tlsConfig, authInterceptor,
		authService, paymentService, accountService, userService, roleService,
	)

	return &App{
		GRPCServer: gRPCApp,
	}
}

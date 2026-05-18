package app

/*
import (
	httpapp "client-service/internal/app/http"
	accountclient "client-service/internal/clients/grpc/billing/account"
	authclient "client-service/internal/clients/grpc/billing/auth"
	roleclient "client-service/internal/clients/grpc/billing/role"
	transactionclient "client-service/internal/clients/grpc/billing/transaction"
	userclient "client-service/internal/clients/grpc/billing/user"

	"client-service/internal/config"

	"client-service/internal/service/account"
	"client-service/internal/service/auth"
	"client-service/internal/service/role"
	"client-service/internal/service/transaction"
	"client-service/internal/service/user"

	"context"
	"crypto/tls"

	"go.uber.org/zap"
)

type App struct {
	HTTPServer *httpapp.HTTPApp
}

func New(
	logger *zap.Logger,
	httpPort int,
	tlsConfig *tls.Config,
	billingCfg *config.BillingClient,
) *App {
	// Инициализация клиентов
	authClient, _ := authclient.NewAuthClient(
		context.Background(),
		logger,
		billingCfg.Address,
		billingCfg.Timeout,
		billingCfg.RetriesCount,
		tlsConfig,
	)
	transactionClient, _ := transactionclient.NewTransactionClient(
		context.Background(),
		logger,
		billingCfg.Address,
		billingCfg.Timeout,
		billingCfg.RetriesCount,
		tlsConfig,
	)
	accountClient, _ := accountclient.NewAccountClient(
		context.Background(),
		logger,
		billingCfg.Address,
		billingCfg.Timeout,
		billingCfg.RetriesCount,
		tlsConfig,
	)
	userClient, _ := userclient.NewUserClient(
		context.Background(),
		logger,
		billingCfg.Address,
		billingCfg.Timeout,
		billingCfg.RetriesCount,
		tlsConfig,
	)
	roleClient, _ := roleclient.NewRoleClient(
		context.Background(),
		logger,
		billingCfg.Address,
		billingCfg.Timeout,
		billingCfg.RetriesCount,
		tlsConfig,
	)

	authService := auth.NewAuthService(authClient, logger)
	transactionService := transaction.NewTransactionService(transactionClient, logger)
	accountService := account.NewAccountService(accountClient, logger)

	userService := user.NewUserService(userClient, logger)
	roleService := role.NewRoleService(roleClient, logger)

	httpServer := httpapp.New(logger, httpPort, authService, transactionService, accountService, userService, roleService)
	return &App{
		HTTPServer: httpServer,
	}
}
*/

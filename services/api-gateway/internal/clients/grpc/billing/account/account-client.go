package account

import (
	"client-service/internal/clients/grpc/interceptor"
	"client-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type AccountClient struct {
	accountApi blnggrpc.AccountServiceClient
	logger     *zap.Logger
}

func NewAccountClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*AccountClient, error) {
	const op = "clients.grpc.billing.account.NewAccountClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "billing-service",
		Certificates: tlsConfig.Certificates,
		RootCAs:      tlsConfig.ClientCAs,
	})

	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptor.InterceptorLogger(logger), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &AccountClient{
		accountApi: blnggrpc.NewAccountServiceClient(cc),
		logger:     logger,
	}, nil
}

// CreateAccount создает новый аккаунт с указанной валютой и именем
func (c *AccountClient) CreateAccount(ctx context.Context, currency, name string) (string, error) {
	const op = "clients.grpc.billing.account.CreateAccount"

	response, err := c.accountApi.CreateAccount(ctx, &blnggrpc.CreateAccountRequest{
		Currency: currency,
		Name:     name,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return response.GetAccountId(), nil
}

// GetAccount возвращает информацию об аккаунте
func (c *AccountClient) GetAccount(ctx context.Context, accountID string) (*models.CurrencyAccount, error) {
	const op = "clients.grpc.billing.account.GetAccount"

	response, err := c.accountApi.GetAccount(ctx, &blnggrpc.GetAccountRequest{
		AccountId: accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoAccount := response.GetAccount()
	return &models.CurrencyAccount{
		ID:            protoAccount.GetId(),
		Name:          protoAccount.GetName(),
		UserID:        uuid.MustParse(protoAccount.GetUserId().GetValue()),
		Currency:      protoAccount.GetCurrency(),
		Balance:       protoAccount.GetBalance(),
		BlockedAmount: protoAccount.GetBlockedAmount(),
		CreatedAt:     protoAccount.GetCreatedAt().AsTime(),
		UpdatedAt:     protoAccount.GetUpdatedAt().AsTime(),
	}, nil
}

// GetUserAccounts возвращает список аккаунтов пользователя
func (c *AccountClient) GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]models.CurrencyAccount, error) {
	const op = "clients.grpc.billing.account.GetUserAccounts"

	response, err := c.accountApi.GetUserAccounts(ctx, &blnggrpc.GetUserAccountsRequest{
		UserId: &blnggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	accounts := make([]models.CurrencyAccount, 0, len(response.GetAccounts()))
	for _, account := range response.GetAccounts() {
		accounts = append(accounts, models.CurrencyAccount{
			ID:            account.GetId(),
			Name:          account.GetName(),
			UserID:        uuid.MustParse(account.GetUserId().GetValue()),
			Currency:      account.GetCurrency(),
			Balance:       account.GetBalance(),
			BlockedAmount: account.GetBlockedAmount(),
			CreatedAt:     account.GetCreatedAt().AsTime(),
			UpdatedAt:     account.GetUpdatedAt().AsTime(),
		})
	}
	return accounts, nil
}

// GetBalance получает баланс по указанному account_id
func (c *AccountClient) GetBalance(ctx context.Context, accountID string) (float64, error) {
	const op = "clients.grpc.billing.account.GetBalance"

	response, err := c.accountApi.GetBalance(ctx, &blnggrpc.GetBalanceRequest{
		AccountId: accountID,
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return response.GetBalance(), nil
}

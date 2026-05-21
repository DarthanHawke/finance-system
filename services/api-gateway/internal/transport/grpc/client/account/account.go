package client

import (
	"api-gateway-service/internal/models"
	"api-gateway-service/internal/transport/grpc/interceptor"
	"context"
	"fmt"
	"time"

	accgrpc "github.com/DarthanHawke/protos-finance-system/gen/go/account"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type AccountClient struct {
	accountApi accgrpc.AccountServiceClient
	logger     *zap.Logger
}

func NewAccountClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*AccountClient, error) {
	const op = "transport.grpc.client.account.NewAccountClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptor.InterceptorLogger(logger), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &AccountClient{
		accountApi: accgrpc.NewAccountServiceClient(cc),
		logger:     logger,
	}, nil
}

// CreateAccount создает новый счет с указанной валютой и именем
func (c *AccountClient) CreateAccount(ctx context.Context, req *models.CreateAccountRequest) error {
	const op = "transport.grpc.client.account.CreateAccount"

	_, err := c.accountApi.CreateAccount(ctx, &accgrpc.CreateAccountRequest{
		Name:     req.Name,
		UserId:   &accgrpc.UUID{Value: req.UserID.String()},
		Currency: req.Currency,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetAccount возвращает информацию о счете
func (c *AccountClient) GetAccount(ctx context.Context, req *models.GetAccountRequest) (models.GetAccountResponse, error) {
	const op = "transport.grpc.client.account.GetAccount"

	response, err := c.accountApi.GetAccount(ctx, &accgrpc.GetAccountRequest{
		Code: req.Code,
	})
	if err != nil {
		return models.GetAccountResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	account := response.GetAccount()
	return models.GetAccountResponse{
		Account: &models.Account{
			Code:           account.GetCode(),
			Name:           account.GetName(),
			UserID:         uuid.MustParse(account.GetUserId().GetValue()),
			Currency:       account.GetCurrency(),
			Balance:        account.GetBalance(),
			FrozenBalance:  account.GetFrozenBalance(),
			ReserveBalance: account.GetReserveBalance(),
			Status:         account.GetStatus(),
			CreatedAt:      account.GetCreatedAt().AsTime(),
			UpdatedAt:      account.GetUpdatedAt().AsTime(),
		},
	}, nil
}

// GetAccounts возвращает список счетов пользователя
func (c *AccountClient) GetAccounts(ctx context.Context, req *models.GetAccountsRequest) (models.GetAccountsResponse, error) {
	const op = "transport.grpc.client.account.GetAccounts"

	response, err := c.accountApi.GetAccounts(ctx, &accgrpc.GetAccountsRequest{
		UserId: &accgrpc.UUID{Value: req.UserID.String()},
		Limit:  int32(req.Limit),
		Offcet: int32(req.Offset),
	})
	if err != nil {
		return models.GetAccountsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	accounts := make([]models.Account, 0, len(response.GetAccounts()))
	for _, account := range response.GetAccounts() {
		accounts = append(accounts, models.Account{
			Code:           account.GetCode(),
			Name:           account.GetName(),
			UserID:         uuid.MustParse(account.GetUserId().GetValue()),
			Currency:       account.GetCurrency(),
			Balance:        account.GetBalance(),
			FrozenBalance:  account.GetFrozenBalance(),
			ReserveBalance: account.GetReserveBalance(),
			Status:         account.GetStatus(),
			CreatedAt:      account.GetCreatedAt().AsTime(),
			UpdatedAt:      account.GetUpdatedAt().AsTime(),
		})
	}
	return models.GetAccountsResponse{
		Accounts: accounts,
	}, nil
}

// BlockAccount блокирует счет
func (c *AccountClient) BlockAccount(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "transport.grpc.client.account.BlockAccount"

	_, err := c.accountApi.BlockAccount(ctx, &accgrpc.BlockAccountRequest{
		Code: req.Code,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CloseAccount закрывает счет
func (c *AccountClient) CloseAccount(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "transport.grpc.client.account.CloseAccount"

	_, err := c.accountApi.CloseAccount(ctx, &accgrpc.CloseAccountRequest{
		Code: req.Code,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

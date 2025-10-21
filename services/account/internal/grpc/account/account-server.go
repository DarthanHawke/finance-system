package account

import (
	"account-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Account interface {
	CreateAccount(ctx context.Context, req models.CreateAccountRequest) error
	GetAccount(ctx context.Context, req models.GetAccountRequest) (models.GetAccountResponse, error)
	GetAccounts(ctx context.Context, req models.GetAccountsRequest) (models.GetAccountsResponse, error)
	BlockAccount(ctx context.Context, req models.UpdateAccountRequest) error
	CloseAccount(ctx context.Context, req models.UpdateAccountRequest) error
}

type AccountServerAPI struct {
	blnggrpc.UnimplementedAccountServiceServer
	account Account
}

func NewAccountServer(gRPC *grpc.Server, account Account) {
	blnggrpc.RegisterAccountServiceServer(gRPC, &AccountServerAPI{account: account})
}

func (s *AccountServerAPI) CreateAccount(ctx context.Context, req *blnggrpc.CreateAccountRequest) (*blnggrpc.CreateAccountResponse, error) {
	if req.GetCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	accountID, err := s.account.CreateAccount(ctx, req.GetCurrency(), req.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create account")
	}

	return &blnggrpc.CreateAccountResponse{AccountId: accountID}, nil
}

func (s *AccountServerAPI) GetAccount(ctx context.Context, req *blnggrpc.GetAccountRequest) (*blnggrpc.GetAccountResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	account, err := s.account.GetAccount(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get account")
	}

	return &blnggrpc.GetAccountResponse{
		Account: &blnggrpc.CurrencyAccount{
			Id:            account.AccountCode,
			Name:          account.Name,
			UserId:        &blnggrpc.UUID{Value: account.UserID.String()},
			Currency:      account.Currency,
			Balance:       account.Balance,
			BlockedAmount: account.BlockedAmount,
			CreatedAt:     timestamppb.New(account.CreatedAt),
			UpdatedAt:     timestamppb.New(account.UpdatedAt),
		},
	}, nil
}

func (s *AccountServerAPI) GetUserAccounts(
	ctx context.Context,
	req *blnggrpc.GetUserAccountsRequest,
) (*blnggrpc.GetUserAccountsResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetUserId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse accounts")
		}
	}

	accounts, err := s.account.GetUserAccounts(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get accounts")
	}

	protoAccounts := make([]*blnggrpc.CurrencyAccount, 0, len(accounts))
	for _, account := range accounts {
		protoAccounts = append(protoAccounts, &blnggrpc.CurrencyAccount{
			Id:            account.AccountCode,
			Name:          account.Name,
			UserId:        &blnggrpc.UUID{Value: account.UserID.String()},
			Currency:      account.Currency,
			Balance:       account.Balance,
			BlockedAmount: account.BlockedAmount,
			CreatedAt:     timestamppb.New(account.CreatedAt),
			UpdatedAt:     timestamppb.New(account.UpdatedAt),
		})
	}

	return &blnggrpc.GetUserAccountsResponse{Accounts: protoAccounts}, nil
}

func (s *AccountServerAPI) GetBalance(ctx context.Context, req *blnggrpc.GetBalanceRequest) (*blnggrpc.GetBalanceResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	balance, err := s.account.GetBalance(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get balance")
	}

	return &blnggrpc.GetBalanceResponse{Balance: balance}, nil
}

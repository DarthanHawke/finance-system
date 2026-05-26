package account

import (
	"account-service/internal/models"
	"context"

	accgrpc "github.com/DarthanHawke/protos-finance-system/gen/go/account/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Account interface {
	CreateAccount(ctx context.Context, req *models.CreateAccountRequest) error
	GetAccount(ctx context.Context, req *models.GetAccountRequest) (models.GetAccountResponse, error)
	GetAccounts(ctx context.Context, req *models.GetAccountsRequest) (models.GetAccountsResponse, error)
	BlockAccount(ctx context.Context, req *models.UpdateAccountRequest) error
	CloseAccount(ctx context.Context, req *models.UpdateAccountRequest) error
}

type AccountServerAPI struct {
	accgrpc.UnimplementedAccountServiceServer
	account Account
}

func NewAccountServer(gRPC *grpc.Server, account Account) {
	accgrpc.RegisterAccountServiceServer(gRPC, &AccountServerAPI{account: account})
}

func (s *AccountServerAPI) CreateAccount(
	ctx context.Context,
	req *accgrpc.CreateAccountRequest,
) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.GetUserId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID format: %v", err)
	}
	if req.GetCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	err = s.account.CreateAccount(ctx, &models.CreateAccountRequest{
		Name:     req.GetName(),
		UserID:   userID,
		Currency: req.GetCurrency(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create account")
	}

	return &emptypb.Empty{}, nil
}

func (s *AccountServerAPI) GetAccount(
	ctx context.Context,
	req *accgrpc.GetAccountRequest,
) (*accgrpc.GetAccountResponse, error) {
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "account code is required")
	}

	account, err := s.account.GetAccount(ctx, &models.GetAccountRequest{
		Code: req.GetCode(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get account")
	}

	return &accgrpc.GetAccountResponse{
		Account: &accgrpc.Account{
			Code:           account.Code,
			Name:           account.Name,
			UserId:         &accgrpc.UUID{Value: account.UserID.String()},
			Currency:       account.Currency,
			Balance:        account.Balance,
			FrozenBalance:  account.FrozenBalance,
			ReserveBalance: account.ReserveBalance,
			Status:         account.Status,
			CreatedAt:      timestamppb.New(account.CreatedAt),
			UpdatedAt:      timestamppb.New(account.UpdatedAt),
		},
	}, nil
}

func (s *AccountServerAPI) GetAccounts(
	ctx context.Context,
	req *accgrpc.GetAccountsRequest,
) (*accgrpc.GetAccountsResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetUserId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID format: %v", err)
		}
	}

	accounts, err := s.account.GetAccounts(ctx, &models.GetAccountsRequest{
		UserID: userID,
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffcet()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get accounts")
	}

	protoAccounts := make([]*accgrpc.Account, 0, len(accounts.Accounts))
	for _, account := range accounts.Accounts {
		protoAccounts = append(protoAccounts, &accgrpc.Account{
			Code:           account.Code,
			Name:           account.Name,
			UserId:         &accgrpc.UUID{Value: account.UserID.String()},
			Currency:       account.Currency,
			Balance:        account.Balance,
			FrozenBalance:  account.FrozenBalance,
			ReserveBalance: account.ReserveBalance,
			Status:         account.Status,
			CreatedAt:      timestamppb.New(account.CreatedAt),
			UpdatedAt:      timestamppb.New(account.UpdatedAt),
		})
	}

	return &accgrpc.GetAccountsResponse{Accounts: protoAccounts}, nil
}

func (s *AccountServerAPI) BlockAccount(
	ctx context.Context,
	req *accgrpc.BlockAccountRequest,
) (*emptypb.Empty, error) {
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "account code is required")
	}

	err := s.account.BlockAccount(ctx, &models.UpdateAccountRequest{
		Code: req.GetCode(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to block account")
	}
	return &emptypb.Empty{}, nil
}

func (s *AccountServerAPI) CloseAccount(
	ctx context.Context,
	req *accgrpc.CloseAccountRequest,
) (*emptypb.Empty, error) {
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "account code is required")
	}

	err := s.account.CloseAccount(ctx, &models.UpdateAccountRequest{
		Code: req.GetCode(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to close account")
	}
	return &emptypb.Empty{}, nil
}

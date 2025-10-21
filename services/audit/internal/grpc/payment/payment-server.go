package transaction

import (
	"billing-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	nilAmount = 0.0
)

type Transaction interface {
	Transfer(ctx context.Context, sender, receiver string, amount float64, description string) (uuid.UUID, error)
	GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error)
	GetAllTransaction(ctx context.Context, targetID uuid.UUID) ([]models.Transaction, error)
	UpdateStatusTransaction(ctx context.Context, transactionID uuid.UUID, status string) error
	CancelTransaction(ctx context.Context, transactionID uuid.UUID) error
	Deposit(ctx context.Context, accountID string, amount float64, description string) (uuid.UUID, error)
	ConvertCurrency(ctx context.Context, fromAccountID, toAccountID string, amount float64, description string) (uuid.UUID, error)
	GetOperationHistory(ctx context.Context, accountID string, limit, offset int) ([]models.BalanceOperation, error)
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

type TransactionServerAPI struct {
	blnggrpc.UnimplementedTransactionServiceServer
	transaction Transaction
}

func NewTransactionServer(gRPC *grpc.Server, transaction Transaction) {
	blnggrpc.RegisterTransactionServiceServer(gRPC, &TransactionServerAPI{transaction: transaction})
}

func (s *TransactionServerAPI) Transfer(
	ctx context.Context,
	req *blnggrpc.TransferRequest,
) (*blnggrpc.TransferResponse, error) {
	if float64(req.GetAmount()) == nilAmount {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	if req.GetSender() == "" || req.GetReceiver() == "" {
		return nil, status.Error(codes.InvalidArgument, "sender and receiver are required")
	}

	transactionID, err := s.transaction.Transfer(
		ctx,
		req.GetSender(),
		req.GetReceiver(),
		float64(req.GetAmount()),
		req.GetDescription(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create transaction")
	}

	return &blnggrpc.TransferResponse{TransactionId: &blnggrpc.UUID{Value: transactionID.String()}}, nil
}

func (s *TransactionServerAPI) GetTransaction(
	ctx context.Context,
	req *blnggrpc.GetTransactionRequest,
) (*blnggrpc.GetTransactionResponse, error) {
	transactionID, err := uuid.Parse(req.GetTransactionId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	transaction, err := s.transaction.GetTransaction(ctx, transactionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}

	return &blnggrpc.GetTransactionResponse{
		Transaction: &blnggrpc.Transaction{
			Id:          &blnggrpc.UUID{Value: transaction.ID.String()},
			Sender:      transaction.Sender,
			Receiver:    transaction.Receiver,
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Status:      transaction.Status,
			Description: &transaction.Description,
			CreatedAt:   timestamppb.New(transaction.CreatedAt),
			UpdatedAt:   timestamppb.New(transaction.UpdatedAt),
		},
	}, nil
}

func (s *TransactionServerAPI) GetAllTransaction(
	ctx context.Context,
	req *blnggrpc.GetAllTransactionRequest,
) (*blnggrpc.GetAllTransactionResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetUserId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	var paymants []models.Transaction
	paymants, err = s.transaction.GetAllTransaction(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}

	protoTransactions := make([]*blnggrpc.Transaction, 0, len(paymants))
	for _, p := range paymants {
		protoTransactions = append(protoTransactions, &blnggrpc.Transaction{
			Id:          &blnggrpc.UUID{Value: p.ID.String()},
			Sender:      p.Sender,
			Receiver:    p.Receiver,
			Amount:      p.Amount,
			Currency:    p.Currency,
			Status:      p.Status,
			Description: &p.Description,
			CreatedAt:   timestamppb.New(p.CreatedAt),
			UpdatedAt:   timestamppb.New(p.UpdatedAt),
		})
	}

	return &blnggrpc.GetAllTransactionResponse{Transaction: protoTransactions}, nil
}

func (s *TransactionServerAPI) UpdateStatusTransaction(
	ctx context.Context,
	req *blnggrpc.UpdateStatusTransactionRequest,
) (*emptypb.Empty, error) {
	transactionID, err := uuid.Parse(req.GetTransactionId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.transaction.UpdateStatusTransaction(ctx, transactionID, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update transaction")
	}

	return &emptypb.Empty{}, nil
}

func (s *TransactionServerAPI) CancelTransaction(
	ctx context.Context,
	req *blnggrpc.CancelTransactionRequest,
) (*emptypb.Empty, error) {
	transactionID, err := uuid.Parse(req.GetTransactionId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.transaction.CancelTransaction(ctx, transactionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel transaction")
	}

	return &emptypb.Empty{}, nil
}

func (s *TransactionServerAPI) Deposit(ctx context.Context, req *blnggrpc.DepositRequest) (*blnggrpc.DepositResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	transactionID, err := s.transaction.Deposit(ctx, req.GetAccountId(), req.GetAmount(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to deposit")
	}

	return &blnggrpc.DepositResponse{TransactionId: &blnggrpc.UUID{Value: transactionID.String()}}, nil
}

func (s *TransactionServerAPI) ConvertCurrency(ctx context.Context, req *blnggrpc.ConvertCurrencyRequest) (*blnggrpc.ConvertCurrencyResponse, error) {
	if req.GetFromAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetToAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	operationID, err := s.transaction.ConvertCurrency(ctx, req.GetFromAccountId(), req.GetToAccountId(), req.GetAmount(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert currency")
	}

	return &blnggrpc.ConvertCurrencyResponse{OperationId: &blnggrpc.UUID{Value: operationID.String()}}, nil
}

func (s *TransactionServerAPI) GetCurrencyRate(
	ctx context.Context,
	req *blnggrpc.GetCurrencyRateRequest,
) (*blnggrpc.GetCurrencyRateResponse, error) {
	if req.GetFromCurrency() == "" || req.GetToCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "both currencies are required")
	}

	rate, err := s.transaction.GetCurrencyRate(ctx, req.GetFromCurrency(), req.GetToCurrency())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get currency rate")
	}

	return &blnggrpc.GetCurrencyRateResponse{Rate: rate}, nil
}

func (s *TransactionServerAPI) GetOperationHistory(ctx context.Context, req *blnggrpc.GetOperationHistoryRequest) (*blnggrpc.GetOperationHistoryResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 10
	}
	offset := int(req.GetOffset())

	operations, err := s.transaction.GetOperationHistory(ctx, req.GetAccountId(), limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get operation history")
	}

	protoOperations := make([]*blnggrpc.BalanceOperation, 0, len(operations))
	for _, operation := range operations {
		protoOperations = append(protoOperations, &blnggrpc.BalanceOperation{
			Id:            &blnggrpc.UUID{Value: operation.ID.String()},
			OperationId:   &blnggrpc.UUID{Value: operation.OperationID.String()},
			TransactionId: &blnggrpc.UUID{Value: operation.TransactionID.String()},
			AccountId:     operation.AccountCode,
			Currency:      operation.Currency,
			Amount:        operation.Amount,
			OperationType: operation.OperationType,
			Status:        operation.Status,
			Description:   operation.Description,
			CreatedAt:     timestamppb.New(operation.CreatedAt),
			ProcessedAt:   timestamppb.New(operation.ProcessedAt),
		})
	}

	return &blnggrpc.GetOperationHistoryResponse{Operations: protoOperations}, nil
}

func (s *TransactionServerAPI) UpdateCurrencyRate(ctx context.Context, req *blnggrpc.UpdateCurrencyRateRequest) (*emptypb.Empty, error) {
	if req.GetFromCurrency() == "" || req.GetToCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "both currencies are required")
	}
	if req.GetRate() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "rate must be positive")
	}

	if err := s.transaction.UpdateCurrencyRate(ctx, req.GetFromCurrency(), req.GetToCurrency(), req.GetRate()); err != nil {
		return nil, status.Error(codes.Internal, "failed to update currency rate")
	}

	return &emptypb.Empty{}, nil
}

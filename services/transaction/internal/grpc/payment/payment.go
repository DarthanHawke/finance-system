package transaction

import (
	"context"
	"transaction-service/internal/models"

	//pmtgrpc "github.com/DarthanHawke/protos-finance-system/gen/go/transaction"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Transaction interface {
	Create(ctx context.Context,
		sender, receiver string,
		amount float64,
		currency, description string,
	) (uuid.UUID, error)
	Get(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error)
	UpdateStatus(ctx context.Context, transactionID uuid.UUID, status string) error
	Cancel(ctx context.Context, transactionID uuid.UUID) error
}

type TransactionServerAPI struct {
	pmtgrpc.UnimplementedTransactionServiceServer
	transaction Transaction
}

func NewTransactionServer(
	gRPC *grpc.Server,
	transaction Transaction,
) {
	pmtgrpc.RegisterTransactionServiceServer(gRPC, &TransactionServerAPI{transaction: transaction})
}

func (s *TransactionServerAPI) CreateTransaction(
	ctx context.Context,
	req *pmtgrpc.CreateTransactionRequest,
) (*pmtgrpc.CreateTransactionResponse, error) {
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}
	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.Sender == "" || req.Receiver == "" {
		return nil, status.Error(codes.InvalidArgument, "sender and receiver are required")
	}

	transactionID, err := s.transaction.Create(ctx, req.GetSender(), req.GetReceiver(), float64(req.GetAmount()), req.GetCurrency(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create transaction")
	}

	return &pmtgrpc.CreateTransactionResponse{Id: &pmtgrpc.UUID{Value: transactionID.String()}}, nil
}

func (s *TransactionServerAPI) GetTransaction(
	ctx context.Context,
	req *pmtgrpc.GetTransactionRequest,
) (*pmtgrpc.GetTransactionResponse, error) {
	transactionID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	transaction, err := s.transaction.Get(ctx, transactionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get transaction")
	}
	return &pmtgrpc.GetTransactionResponse{
		Transaction: &pmtgrpc.Transaction{
			Id:          &pmtgrpc.UUID{Value: transaction.ID.String()},
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

func (s *TransactionServerAPI) UpdateStatusTransaction(
	ctx context.Context,
	req *pmtgrpc.UpdateStatusTransactionRequest,
) (*emptypb.Empty, error) {
	transactionID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.transaction.UpdateStatus(ctx, transactionID, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update transaction")
	}

	return &emptypb.Empty{}, nil
}

func (s *TransactionServerAPI) CancelTransaction(
	ctx context.Context,
	req *pmtgrpc.CancelTransactionRequest,
) (*emptypb.Empty, error) {
	transactionID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.transaction.Cancel(ctx, transactionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel transaction")
	}

	return &emptypb.Empty{}, nil
}

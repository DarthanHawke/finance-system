package operation

import (
	"context"
	"transaction-service/internal/models"

	trngrpc "github.com/DarthanHawke/protos-finance-system/gen/go/transaction"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Transaction interface {
	GetTransaction(
		ctx context.Context,
		req *models.GetTransactionRequest,
	) (models.GetTransactionResponse, error)
	GetTransactions(
		ctx context.Context,
		req *models.GetTransactionsRequest,
	) (models.GetTransactionsResponse, error)
}

type TransactionServerAPI struct {
	trngrpc.UnimplementedTransactionServiceServer
	transaction Transaction
}

func NewTransactionServer(
	gRPC *grpc.Server,
	transaction Transaction,
) {
	trngrpc.RegisterTransactionServiceServer(gRPC, &TransactionServerAPI{transaction: transaction})
}

func (s *TransactionServerAPI) GetTransaction(
	ctx context.Context,
	req *trngrpc.GetTransactionRequest,
) (*trngrpc.GetTransactionResponse, error) {
	transactionID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	transaction, err := s.transaction.GetTransaction(ctx, &models.GetTransactionRequest{
		ID: transactionID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}
	return &trngrpc.GetTransactionResponse{
		Transaction: &trngrpc.Transaction{
			Id:                   &trngrpc.UUID{Value: transaction.ID.String()},
			Type:                 transaction.Type,
			Amount:               transaction.Amount,
			Currency:             transaction.Currency,
			SenderType:           transaction.SenderType,
			SenderAccountCode:    transaction.SenderAccountCode,
			SenderPhone:          transaction.SenderPhone,
			SenderCardNumber:     transaction.SenderCardNumber,
			RecipientType:        transaction.RecipientType,
			RecipientAccountCode: transaction.RecipientAccountCode,
			RecipientPhone:       transaction.RecipientPhone,
			RecipientCardNumber:  transaction.RecipientCardNumber,
			ProcessingCode:       transaction.ProcessingCode,
			Stan:                 transaction.Stan,
			AuthorizationCode:    transaction.AuthorizationCode,
			Description:          transaction.Description,
			Status:               transaction.Status,
			CreatedAt:            timestamppb.New(transaction.CreatedAt),
			UpdatedAt:            timestamppb.New(transaction.UpdatedAt),
		},
	}, nil
}

func (s *TransactionServerAPI) GetTransactions(
	ctx context.Context,
	req *trngrpc.GetTransactionsRequest,
) (*trngrpc.GetTransactionsResponse, error) {
	transactions, err := s.transaction.GetTransactions(ctx, &models.GetTransactionsRequest{
		AccountCode: req.GetAccountCode(),
		Limit:       int(req.GetLimit()),
		Offset:      int(req.GetOffcet()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}

	protoTransactions := make([]*trngrpc.Transaction, 0, len(transactions.Transactions))
	for _, transaction := range transactions.Transactions {
		protoTransactions = append(protoTransactions, &trngrpc.Transaction{
			Id:                   &trngrpc.UUID{Value: transaction.ID.String()},
			Type:                 transaction.Type,
			Amount:               transaction.Amount,
			Currency:             transaction.Currency,
			SenderType:           transaction.SenderType,
			SenderAccountCode:    transaction.SenderAccountCode,
			SenderPhone:          transaction.SenderPhone,
			SenderCardNumber:     transaction.SenderCardNumber,
			RecipientType:        transaction.RecipientType,
			RecipientAccountCode: transaction.RecipientAccountCode,
			RecipientPhone:       transaction.RecipientPhone,
			RecipientCardNumber:  transaction.RecipientCardNumber,
			ProcessingCode:       transaction.ProcessingCode,
			Stan:                 transaction.Stan,
			AuthorizationCode:    transaction.AuthorizationCode,
			Description:          transaction.Description,
			Status:               transaction.Status,
			CreatedAt:            timestamppb.New(transaction.CreatedAt),
			UpdatedAt:            timestamppb.New(transaction.UpdatedAt),
		})
	}
	return &trngrpc.GetTransactionsResponse{Transaction: protoTransactions}, nil

}

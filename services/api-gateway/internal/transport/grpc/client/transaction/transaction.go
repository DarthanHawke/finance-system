package client

import (
	"api-gateway-service/internal/models"
	"api-gateway-service/internal/transport/grpc/interceptor"
	"context"
	"fmt"
	"time"

	trngrpc "github.com/DarthanHawke/protos-finance-system/gen/go/transaction"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type TransactionClient struct {
	transactionApi trngrpc.TransactionServiceClient
	logger         *zap.Logger
}

func NewTransactionClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*TransactionClient, error) {
	const op = "transport.grpc.client.operation.NewTransactionClient"

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

	return &TransactionClient{
		transactionApi: trngrpc.NewTransactionServiceClient(cc),
		logger:         logger,
	}, nil
}

// GetTransaction получает информацию о платеже
func (c *TransactionClient) GetTransaction(
	ctx context.Context,
	req *models.GetTransactionRequest,
) (models.GetTransactionResponse, error) {
	const op = "transport.grpc.client.operation.GetTransaction"

	response, err := c.transactionApi.GetTransaction(ctx, &trngrpc.GetTransactionRequest{
		Id: &trngrpc.UUID{Value: req.ID.String()},
	})
	if err != nil {
		return models.GetTransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	transaction := response.GetTransaction()
	return models.GetTransactionResponse{
		Transaction: &models.Transaction{
			ID:                   uuid.MustParse(transaction.GetId().GetValue()),
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
			CreatedAt:            transaction.CreatedAt.AsTime(),
			UpdatedAt:            transaction.UpdatedAt.AsTime(),
		},
	}, nil
}

// GetTransactions получает информацию о всех платежах
func (c *TransactionClient) GetTransactions(
	ctx context.Context,
	req *models.GetTransactionsRequest,
) (models.GetTransactionsResponse, error) {
	const op = "transport.grpc.client.operation.GetTransactions"

	response, err := c.transactionApi.GetTransactions(ctx, &trngrpc.GetTransactionsRequest{
		AccountCode: req.AccountCode,
		Limit:       int32(req.Limit),
		Offcet:      int32(req.Offset),
	})
	if err != nil {
		return models.GetTransactionsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	transactions := make([]models.Transaction, 0, len(response.GetTransaction()))
	for _, transaction := range response.GetTransaction() {
		transactions = append(transactions, models.Transaction{
			ID:                   uuid.MustParse(transaction.GetId().GetValue()),
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
			CreatedAt:            transaction.CreatedAt.AsTime(),
			UpdatedAt:            transaction.UpdatedAt.AsTime(),
		})
	}
	return models.GetTransactionsResponse{
		Transactions: transactions,
	}, nil
}

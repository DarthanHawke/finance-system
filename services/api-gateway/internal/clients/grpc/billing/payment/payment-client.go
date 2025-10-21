package transaction

import (
	"client-service/internal/clients/grpc/interceptor"
	"client-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	blggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type TransactionClient struct {
	transactionApi blggrpc.TransactionServiceClient
	logger         *zap.Logger
}

func NewTransactionClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*TransactionClient, error) {
	const op = "clients.grpc.billing.transaction.NewTransactionClient"

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

	return &TransactionClient{
		transactionApi: blggrpc.NewTransactionServiceClient(cc),
		logger:         logger,
	}, nil
}

// CreateTransaction создает новый платеж
func (c *TransactionClient) Transfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.transaction.Transfer"

	response, err := c.transactionApi.Transfer(ctx, &blggrpc.TransferRequest{
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Description: &description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	transactionId, err := uuid.Parse(response.GetTransactionId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactionId, nil
}

// Deposit пополняет баланс на указанную сумму
func (c *TransactionClient) Deposit(
	ctx context.Context,
	accountID string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.transaction.Deposit"

	response, err := c.transactionApi.Deposit(ctx, &blggrpc.DepositRequest{
		AccountId:   accountID,
		Amount:      amount,
		Description: description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	transactionID, err := uuid.Parse(response.GetTransactionId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactionID, nil
}

// GetTransaction получает информацию о платеже
func (c *TransactionClient) GetTransaction(
	ctx context.Context,
	transactionID uuid.UUID,
) (*models.Transaction, error) {
	const op = "clients.grpc.billing.transaction.GetTransaction"

	response, err := c.transactionApi.GetTransaction(ctx, &blggrpc.GetTransactionRequest{
		TransactionId: &blggrpc.UUID{Value: transactionID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoTransaction := response.GetTransaction()
	return &models.Transaction{
		ID:          uuid.MustParse(protoTransaction.GetId().GetValue()),
		Sender:      protoTransaction.Sender,
		Receiver:    protoTransaction.Receiver,
		Amount:      protoTransaction.Amount,
		Currency:    protoTransaction.Currency,
		Status:      protoTransaction.Status,
		Description: *protoTransaction.Description,
		CreatedAt:   protoTransaction.CreatedAt.AsTime(),
		UpdatedAt:   protoTransaction.UpdatedAt.AsTime(),
	}, nil
}

// GetAllTransaction получает информацию о всех платежах
func (c *TransactionClient) GetAllTransaction(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Transaction, error) {
	const op = "clients.grpc.billing.transaction.GetTransaction"

	response, err := c.transactionApi.GetAllTransaction(ctx, &blggrpc.GetAllTransactionRequest{
		UserId: &blggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	transactions := make([]models.Transaction, 0, len(response.GetTransaction()))
	for _, transaction := range response.GetTransaction() {
		transactions = append(transactions, models.Transaction{
			ID:          uuid.MustParse(transaction.GetId().GetValue()),
			Sender:      transaction.Sender,
			Receiver:    transaction.Receiver,
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Status:      transaction.Status,
			Description: *transaction.Description,
			CreatedAt:   transaction.CreatedAt.AsTime(),
			UpdatedAt:   transaction.UpdatedAt.AsTime(),
		})
	}
	return transactions, nil
}

// UpdateStatusTransaction отменяет платеж
func (c *TransactionClient) UpdateStatusTransaction(ctx context.Context,
	transactionID uuid.UUID,
	status string,
) error {
	const op = "clients.grpc.billing.transaction.UpdateStatusTransaction"

	_, err := c.transactionApi.UpdateStatusTransaction(ctx, &blggrpc.UpdateStatusTransactionRequest{
		TransactionId: &blggrpc.UUID{Value: transactionID.String()},
		Status:        status,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CancelTransaction отменяет платеж
func (c *TransactionClient) CancelTransaction(ctx context.Context, transactionID uuid.UUID) error {
	const op = "clients.grpc.billing.transaction.CancelTransaction"

	_, err := c.transactionApi.CancelTransaction(ctx, &blggrpc.CancelTransactionRequest{
		TransactionId: &blggrpc.UUID{Value: transactionID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует валюту между счетами
func (c *TransactionClient) ConvertCurrency(
	ctx context.Context,
	fromAccountID, toAccountID string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.transaction.ConvertCurrency"

	response, err := c.transactionApi.ConvertCurrency(ctx, &blggrpc.ConvertCurrencyRequest{
		FromAccountId: fromAccountID,
		ToAccountId:   toAccountID,
		Amount:        amount,
		Description:   description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	operationID, err := uuid.Parse(response.GetOperationId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// GetOperationHistory получает историю операций по счету
func (c *TransactionClient) GetOperationHistory(
	ctx context.Context,
	accountID string,
	limit, offset int,
) ([]models.BalanceOperation, error) {
	const op = "clients.grpc.billing.transaction.GetOperationHistory"

	response, err := c.transactionApi.GetOperationHistory(ctx, &blggrpc.GetOperationHistoryRequest{
		AccountId: accountID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	operations := make([]models.BalanceOperation, 0, len(response.GetOperations()))
	for _, operation := range response.GetOperations() {
		operations = append(operations, models.BalanceOperation{
			ID:            uuid.MustParse(operation.GetId().GetValue()),
			OperationID:   uuid.MustParse(operation.GetOperationId().GetValue()),
			AccountID:     operation.GetAccountId(),
			TransactionID: uuid.MustParse(operation.GetTransactionId().GetValue()),
			Currency:      operation.GetCurrency(),
			Amount:        operation.GetAmount(),
			OperationType: operation.GetOperationType(),
			Status:        operation.GetStatus(),
			Description:   operation.GetDescription(),
			CreatedAt:     operation.GetCreatedAt().AsTime(),
			ProcessedAt:   operation.GetProcessedAt().AsTime(),
		})
	}

	return operations, nil
}

// UpdateCurrencyRate обновляет курс валют
func (c *TransactionClient) UpdateCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
	rate float64,
) error {
	const op = "clients.grpc.billing.transaction.UpdateCurrencyRate"

	_, err := c.transactionApi.UpdateCurrencyRate(ctx, &blggrpc.UpdateCurrencyRateRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rate,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate получает текущий курс обмена между валютами
func (c *TransactionClient) GetCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
) (float64, error) {
	const op = "clients.grpc.billing.transaction.GetCurrencyRate"

	response, err := c.transactionApi.GetCurrencyRate(ctx, &blggrpc.GetCurrencyRateRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return response.GetRate(), nil
}

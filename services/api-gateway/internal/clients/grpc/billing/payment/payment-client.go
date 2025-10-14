package payment

import (
	"client-service/internal/clients/grpc/interceptor"
	"client-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	blggrpc "github.com/DarthanHawke/protos-payment-system/gen/go/billing"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type PaymentClient struct {
	paymentApi blggrpc.PaymentServiceClient
	logger     *zap.Logger
}

func NewPaymentClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*PaymentClient, error) {
	const op = "clients.grpc.billing.payment.NewPaymentClient"

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

	return &PaymentClient{
		paymentApi: blggrpc.NewPaymentServiceClient(cc),
		logger:     logger,
	}, nil
}

// CreatePayment создает новый платеж
func (c *PaymentClient) Transfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.payment.Transfer"

	response, err := c.paymentApi.Transfer(ctx, &blggrpc.TransferRequest{
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Description: &description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	paymentId, err := uuid.Parse(response.GetPaymentId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return paymentId, nil
}

// Deposit пополняет баланс на указанную сумму
func (c *PaymentClient) Deposit(
	ctx context.Context,
	accountID string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.payment.Deposit"

	response, err := c.paymentApi.Deposit(ctx, &blggrpc.DepositRequest{
		AccountId:   accountID,
		Amount:      amount,
		Description: description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	paymentID, err := uuid.Parse(response.GetPaymentId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return paymentID, nil
}

// GetPayment получает информацию о платеже
func (c *PaymentClient) GetPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) (*models.Payment, error) {
	const op = "clients.grpc.billing.payment.GetPayment"

	response, err := c.paymentApi.GetPayment(ctx, &blggrpc.GetPaymentRequest{
		PaymentId: &blggrpc.UUID{Value: paymentID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoPayment := response.GetPayment()
	return &models.Payment{
		ID:          uuid.MustParse(protoPayment.GetId().GetValue()),
		Sender:      protoPayment.Sender,
		Receiver:    protoPayment.Receiver,
		Amount:      protoPayment.Amount,
		Currency:    protoPayment.Currency,
		Status:      protoPayment.Status,
		Description: *protoPayment.Description,
		CreatedAt:   protoPayment.CreatedAt.AsTime(),
		UpdatedAt:   protoPayment.UpdatedAt.AsTime(),
	}, nil
}

// GetAllPayment получает информацию о всех платежах
func (c *PaymentClient) GetAllPayment(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Payment, error) {
	const op = "clients.grpc.billing.payment.GetPayment"

	response, err := c.paymentApi.GetAllPayment(ctx, &blggrpc.GetAllPaymentRequest{
		UserId: &blggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	payments := make([]models.Payment, 0, len(response.GetPayment()))
	for _, payment := range response.GetPayment() {
		payments = append(payments, models.Payment{
			ID:          uuid.MustParse(payment.GetId().GetValue()),
			Sender:      payment.Sender,
			Receiver:    payment.Receiver,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
			Status:      payment.Status,
			Description: *payment.Description,
			CreatedAt:   payment.CreatedAt.AsTime(),
			UpdatedAt:   payment.UpdatedAt.AsTime(),
		})
	}
	return payments, nil
}

// UpdateStatusPayment отменяет платеж
func (c *PaymentClient) UpdateStatusPayment(ctx context.Context,
	paymentID uuid.UUID,
	status string,
) error {
	const op = "clients.grpc.billing.payment.UpdateStatusPayment"

	_, err := c.paymentApi.UpdateStatusPayment(ctx, &blggrpc.UpdateStatusPaymentRequest{
		PaymentId: &blggrpc.UUID{Value: paymentID.String()},
		Status:    status,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CancelPayment отменяет платеж
func (c *PaymentClient) CancelPayment(ctx context.Context, paymentID uuid.UUID) error {
	const op = "clients.grpc.billing.payment.CancelPayment"

	_, err := c.paymentApi.CancelPayment(ctx, &blggrpc.CancelPaymentRequest{
		PaymentId: &blggrpc.UUID{Value: paymentID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует валюту между счетами
func (c *PaymentClient) ConvertCurrency(
	ctx context.Context,
	fromAccountID, toAccountID string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.billing.payment.ConvertCurrency"

	response, err := c.paymentApi.ConvertCurrency(ctx, &blggrpc.ConvertCurrencyRequest{
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
func (c *PaymentClient) GetOperationHistory(
	ctx context.Context,
	accountID string,
	limit, offset int,
) ([]models.BalanceOperation, error) {
	const op = "clients.grpc.billing.payment.GetOperationHistory"

	response, err := c.paymentApi.GetOperationHistory(ctx, &blggrpc.GetOperationHistoryRequest{
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
			PaymentID:     uuid.MustParse(operation.GetPaymentId().GetValue()),
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
func (c *PaymentClient) UpdateCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
	rate float64,
) error {
	const op = "clients.grpc.billing.payment.UpdateCurrencyRate"

	_, err := c.paymentApi.UpdateCurrencyRate(ctx, &blggrpc.UpdateCurrencyRateRequest{
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
func (c *PaymentClient) GetCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
) (float64, error) {
	const op = "clients.grpc.billing.payment.GetCurrencyRate"

	response, err := c.paymentApi.GetCurrencyRate(ctx, &blggrpc.GetCurrencyRateRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return response.GetRate(), nil
}

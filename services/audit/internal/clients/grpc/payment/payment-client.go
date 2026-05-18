package transaction

/* legacy code
import (
	"billing-service/internal/clients/grpc/interceptor"
	"billing-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pmtgrpc "github.com/DarthanHawke/protos-finance-system/gen/go/transaction"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type TransactionClient struct {
	transactionApi pmtgrpc.TransactionServiceClient
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
	const op = "clients.grpc.transaction.NewTransactionClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "transaction-service",
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
		transactionApi: pmtgrpc.NewTransactionServiceClient(cc),
		logger:         logger,
	}, nil
}

// CreateTransaction создает новый платеж
func (c *TransactionClient) CreateTransaction(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	currency, description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.transaction.CreateTransaction"

	response, err := c.transactionApi.CreateTransaction(ctx, &pmtgrpc.CreateTransactionRequest{
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Currency:    currency,
		Description: &description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	transactionId, err := uuid.Parse(response.GetId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactionId, nil
}

// GetTransaction получает информацию о платеже
func (c *TransactionClient) GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error) {
	const op = "clients.grpc.transaction.GetTransaction"

	response, err := c.transactionApi.GetTransaction(ctx, &pmtgrpc.GetTransactionRequest{
		Id: &pmtgrpc.UUID{Value: transactionID.String()},
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

// UpdateStatusTransaction обновляет статус платежа
func (c *TransactionClient) UpdateStatusTransaction(ctx context.Context, transactionID uuid.UUID, status string) error {
	const op = "clients.grpc.transaction.UpdateStatusTransaction"

	_, err := c.transactionApi.UpdateStatusTransaction(ctx, &pmtgrpc.UpdateStatusTransactionRequest{
		Id:     &pmtgrpc.UUID{Value: transactionID.String()},
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CancelTransaction отменяет платеж
func (c *TransactionClient) CancelTransaction(ctx context.Context, transactionID uuid.UUID) error {
	const op = "clients.grpc.transaction.CancelTransaction"

	_, err := c.transactionApi.CancelTransaction(ctx, &pmtgrpc.CancelTransactionRequest{
		Id: &pmtgrpc.UUID{Value: transactionID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
*/

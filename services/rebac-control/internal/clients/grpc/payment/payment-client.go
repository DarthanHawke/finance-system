package payment

import (
	"billing-service/internal/clients/grpc/interceptor"
	"billing-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pmtgrpc "github.com/DarthanHawke/protos-payment-system/gen/go/payment"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type PaymentClient struct {
	paymentApi pmtgrpc.PaymentServiceClient
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
	const op = "clients.grpc.payment.NewPaymentClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "payment-service",
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
		paymentApi: pmtgrpc.NewPaymentServiceClient(cc),
		logger:     logger,
	}, nil
}

// CreatePayment создает новый платеж
func (c *PaymentClient) CreatePayment(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	currency, description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.payment.CreatePayment"

	response, err := c.paymentApi.CreatePayment(ctx, &pmtgrpc.CreatePaymentRequest{
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Currency:    currency,
		Description: &description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	paymentId, err := uuid.Parse(response.GetId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return paymentId, nil
}

// GetPayment получает информацию о платеже
func (c *PaymentClient) GetPayment(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error) {
	const op = "clients.grpc.payment.GetPayment"

	response, err := c.paymentApi.GetPayment(ctx, &pmtgrpc.GetPaymentRequest{
		Id: &pmtgrpc.UUID{Value: paymentID.String()},
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

// UpdateStatusPayment обновляет статус платежа
func (c *PaymentClient) UpdateStatusPayment(ctx context.Context, paymentID uuid.UUID, status string) error {
	const op = "clients.grpc.payment.UpdateStatusPayment"

	_, err := c.paymentApi.UpdateStatusPayment(ctx, &pmtgrpc.UpdateStatusPaymentRequest{
		Id:     &pmtgrpc.UUID{Value: paymentID.String()},
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CancelPayment отменяет платеж
func (c *PaymentClient) CancelPayment(ctx context.Context, paymentID uuid.UUID) error {
	const op = "clients.grpc.payment.CancelPayment"

	_, err := c.paymentApi.CancelPayment(ctx, &pmtgrpc.CancelPaymentRequest{
		Id: &pmtgrpc.UUID{Value: paymentID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

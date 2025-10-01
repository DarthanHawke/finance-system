package payment

import (
	"context"

	blnggrpc "github.com/DarthanHawke/protos-payment-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Payment interface {
	ConvertCurrency(ctx context.Context, fromAccountID, toAccountID string, amount float64, description string) (uuid.UUID, error)
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

type PaymentServerAPI struct {
	blnggrpc.UnimplementedPaymentServiceServer
	payment Payment
}

func NewPaymentServer(gRPC *grpc.Server, payment Payment) {
	blnggrpc.RegisterPaymentServiceServer(gRPC, &PaymentServerAPI{payment: payment})
}

func (s *PaymentServerAPI) ConvertCurrency(ctx context.Context, req *blnggrpc.ConvertCurrencyRequest) (*blnggrpc.ConvertCurrencyResponse, error) {
	if req.GetFromAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetToAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	operationID, err := s.payment.ConvertCurrency(ctx, req.GetFromAccountId(), req.GetToAccountId(), req.GetAmount(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert currency")
	}

	return &blnggrpc.ConvertCurrencyResponse{OperationId: &blnggrpc.UUID{Value: operationID.String()}}, nil
}

func (s *PaymentServerAPI) GetCurrencyRate(
	ctx context.Context,
	req *blnggrpc.GetCurrencyRateRequest,
) (*blnggrpc.GetCurrencyRateResponse, error) {
	if req.GetFromCurrency() == "" || req.GetToCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "both currencies are required")
	}

	rate, err := s.payment.GetCurrencyRate(ctx, req.GetFromCurrency(), req.GetToCurrency())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get currency rate")
	}

	return &blnggrpc.GetCurrencyRateResponse{Rate: rate}, nil
}

func (s *PaymentServerAPI) UpdateCurrencyRate(ctx context.Context, req *blnggrpc.UpdateCurrencyRateRequest) (*emptypb.Empty, error) {
	if req.GetFromCurrency() == "" || req.GetToCurrency() == "" {
		return nil, status.Error(codes.InvalidArgument, "both currencies are required")
	}
	if req.GetRate() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "rate must be positive")
	}

	if err := s.payment.UpdateCurrencyRate(ctx, req.GetFromCurrency(), req.GetToCurrency(), req.GetRate()); err != nil {
		return nil, status.Error(codes.Internal, "failed to update currency rate")
	}

	return &emptypb.Empty{}, nil
}

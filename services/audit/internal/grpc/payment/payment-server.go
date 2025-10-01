package payment

import (
	"billing-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-payment-system/gen/go/billing"
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

type Payment interface {
	Transfer(ctx context.Context, sender, receiver string, amount float64, description string) (uuid.UUID, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error)
	GetAllPayment(ctx context.Context, targetID uuid.UUID) ([]models.Payment, error)
	UpdateStatusPayment(ctx context.Context, paymentID uuid.UUID, status string) error
	CancelPayment(ctx context.Context, paymentID uuid.UUID) error
	Deposit(ctx context.Context, accountID string, amount float64, description string) (uuid.UUID, error)
	ConvertCurrency(ctx context.Context, fromAccountID, toAccountID string, amount float64, description string) (uuid.UUID, error)
	GetOperationHistory(ctx context.Context, accountID string, limit, offset int) ([]models.BalanceOperation, error)
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

func (s *PaymentServerAPI) Transfer(
	ctx context.Context,
	req *blnggrpc.TransferRequest,
) (*blnggrpc.TransferResponse, error) {
	if float64(req.GetAmount()) == nilAmount {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	if req.GetSender() == "" || req.GetReceiver() == "" {
		return nil, status.Error(codes.InvalidArgument, "sender and receiver are required")
	}

	paymentID, err := s.payment.Transfer(
		ctx,
		req.GetSender(),
		req.GetReceiver(),
		float64(req.GetAmount()),
		req.GetDescription(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create payment")
	}

	return &blnggrpc.TransferResponse{PaymentId: &blnggrpc.UUID{Value: paymentID.String()}}, nil
}

func (s *PaymentServerAPI) GetPayment(
	ctx context.Context,
	req *blnggrpc.GetPaymentRequest,
) (*blnggrpc.GetPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.GetPaymentId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	payment, err := s.payment.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}

	return &blnggrpc.GetPaymentResponse{
		Payment: &blnggrpc.Payment{
			Id:          &blnggrpc.UUID{Value: payment.ID.String()},
			Sender:      payment.Sender,
			Receiver:    payment.Receiver,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
			Status:      payment.Status,
			Description: &payment.Description,
			CreatedAt:   timestamppb.New(payment.CreatedAt),
			UpdatedAt:   timestamppb.New(payment.UpdatedAt),
		},
	}, nil
}

func (s *PaymentServerAPI) GetAllPayment(
	ctx context.Context,
	req *blnggrpc.GetAllPaymentRequest,
) (*blnggrpc.GetAllPaymentResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetUserId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	var paymants []models.Payment
	paymants, err = s.payment.GetAllPayment(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}

	protoPayments := make([]*blnggrpc.Payment, 0, len(paymants))
	for _, p := range paymants {
		protoPayments = append(protoPayments, &blnggrpc.Payment{
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

	return &blnggrpc.GetAllPaymentResponse{Payment: protoPayments}, nil
}

func (s *PaymentServerAPI) UpdateStatusPayment(
	ctx context.Context,
	req *blnggrpc.UpdateStatusPaymentRequest,
) (*emptypb.Empty, error) {
	paymentID, err := uuid.Parse(req.GetPaymentId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.payment.UpdateStatusPayment(ctx, paymentID, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update payment")
	}

	return &emptypb.Empty{}, nil
}

func (s *PaymentServerAPI) CancelPayment(
	ctx context.Context,
	req *blnggrpc.CancelPaymentRequest,
) (*emptypb.Empty, error) {
	paymentID, err := uuid.Parse(req.GetPaymentId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.payment.CancelPayment(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel payment")
	}

	return &emptypb.Empty{}, nil
}

func (s *PaymentServerAPI) Deposit(ctx context.Context, req *blnggrpc.DepositRequest) (*blnggrpc.DepositResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	paymentID, err := s.payment.Deposit(ctx, req.GetAccountId(), req.GetAmount(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to deposit")
	}

	return &blnggrpc.DepositResponse{PaymentId: &blnggrpc.UUID{Value: paymentID.String()}}, nil
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

func (s *PaymentServerAPI) GetOperationHistory(ctx context.Context, req *blnggrpc.GetOperationHistoryRequest) (*blnggrpc.GetOperationHistoryResponse, error) {
	if req.GetAccountId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid account ID")
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 10
	}
	offset := int(req.GetOffset())

	operations, err := s.payment.GetOperationHistory(ctx, req.GetAccountId(), limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get operation history")
	}

	protoOperations := make([]*blnggrpc.BalanceOperation, 0, len(operations))
	for _, operation := range operations {
		protoOperations = append(protoOperations, &blnggrpc.BalanceOperation{
			Id:            &blnggrpc.UUID{Value: operation.ID.String()},
			OperationId:   &blnggrpc.UUID{Value: operation.OperationID.String()},
			PaymentId:     &blnggrpc.UUID{Value: operation.PaymentID.String()},
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

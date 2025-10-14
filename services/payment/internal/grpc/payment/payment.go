package payment

import (
	"context"
	"payment-service/internal/models"

	pmtgrpc "github.com/DarthanHawke/protos-payment-system/gen/go/payment"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Payment interface {
	Create(ctx context.Context,
		sender, receiver string,
		amount float64,
		currency, description string,
	) (uuid.UUID, error)
	Get(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error)
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string) error
	Cancel(ctx context.Context, paymentID uuid.UUID) error
}

type PaymentServerAPI struct {
	pmtgrpc.UnimplementedPaymentServiceServer
	payment Payment
}

func NewPaymentServer(
	gRPC *grpc.Server,
	payment Payment,
) {
	pmtgrpc.RegisterPaymentServiceServer(gRPC, &PaymentServerAPI{payment: payment})
}

func (s *PaymentServerAPI) CreatePayment(
	ctx context.Context,
	req *pmtgrpc.CreatePaymentRequest,
) (*pmtgrpc.CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}
	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if req.Sender == "" || req.Receiver == "" {
		return nil, status.Error(codes.InvalidArgument, "sender and receiver are required")
	}

	paymentID, err := s.payment.Create(ctx, req.GetSender(), req.GetReceiver(), float64(req.GetAmount()), req.GetCurrency(), req.GetDescription())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create payment")
	}

	return &pmtgrpc.CreatePaymentResponse{Id: &pmtgrpc.UUID{Value: paymentID.String()}}, nil
}

func (s *PaymentServerAPI) GetPayment(
	ctx context.Context,
	req *pmtgrpc.GetPaymentRequest,
) (*pmtgrpc.GetPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	payment, err := s.payment.Get(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}
	return &pmtgrpc.GetPaymentResponse{
		Payment: &pmtgrpc.Payment{
			Id:          &pmtgrpc.UUID{Value: payment.ID.String()},
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

func (s *PaymentServerAPI) UpdateStatusPayment(
	ctx context.Context,
	req *pmtgrpc.UpdateStatusPaymentRequest,
) (*emptypb.Empty, error) {
	paymentID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.payment.UpdateStatus(ctx, paymentID, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update payment")
	}

	return &emptypb.Empty{}, nil
}

func (s *PaymentServerAPI) CancelPayment(
	ctx context.Context,
	req *pmtgrpc.CancelPaymentRequest,
) (*emptypb.Empty, error) {
	paymentID, err := uuid.Parse(req.GetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID format: %v", err)
	}

	err = s.payment.Cancel(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to cancel payment")
	}

	return &emptypb.Empty{}, nil
}

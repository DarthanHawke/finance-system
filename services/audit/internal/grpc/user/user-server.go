package user

import (
	"billing-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	nilStr = ""
)

type User interface {
	GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error)
	GetProfile(ctx context.Context, targetID uuid.UUID) (*models.User, error)
	UpdateName(ctx context.Context, targetID uuid.UUID, fulName string) error
	UpdateEmail(ctx context.Context, targetID uuid.UUID, email string) error
	UpdatePassword(ctx context.Context, password string) error
}

type UserServerAPI struct {
	blnggrpc.UnimplementedUserServiceServer
	user User
}

func NewUserServer(gRPC *grpc.Server, user User) {
	blnggrpc.RegisterUserServiceServer(gRPC, &UserServerAPI{user: user})
}

func (s *UserServerAPI) GetAllUsers(
	ctx context.Context,
	req *blnggrpc.GetAllUsersRequest,
) (*blnggrpc.GetAllUsersResponse, error) {
	// Валидация параметров пагинации
	if req.GetLimit() < 0 || req.GetOffset() < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid pagination parameters")
	}

	limit := int(req.GetLimit())
	offset := int(req.GetOffset())

	users, err := s.user.GetAllUsers(ctx, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get users list")
	}

	// Конвертация в protobuf сообщения
	protoUsers := make([]*blnggrpc.User, 0, len(users))
	for _, user := range users {
		protoUsers = append(protoUsers, &blnggrpc.User{
			Id:        &blnggrpc.UUID{Value: user.ID.String()},
			Email:     user.Email,
			FullName:  user.FullName,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		})
	}

	return &blnggrpc.GetAllUsersResponse{Users: protoUsers}, nil
}

func (s *UserServerAPI) GetProfile(
	ctx context.Context,
	req *blnggrpc.GetProfileRequest,
) (*blnggrpc.GetProfileResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	user, err := s.user.GetProfile(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &blnggrpc.GetProfileResponse{
		User: &blnggrpc.User{
			Id:        &blnggrpc.UUID{Value: user.ID.String()},
			Email:     user.Email,
			FullName:  user.FullName,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

func (s *UserServerAPI) UpdateName(
	ctx context.Context,
	req *blnggrpc.UpdateNameRequest,
) (*emptypb.Empty, error) {
	if req.Name == nilStr {
		return nil, status.Error(codes.InvalidArgument, "incorrect data")
	}

	var userID uuid.UUID
	var err error
	userIDStr := req.GetId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	err = s.user.UpdateName(ctx, userID, req.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to change user")
	}

	return &emptypb.Empty{}, nil
}

func (s *UserServerAPI) UpdateEmail(
	ctx context.Context,
	req *blnggrpc.UpdateEmailRequest,
) (*emptypb.Empty, error) {
	if req.Email == nilStr {
		return nil, status.Error(codes.InvalidArgument, "incorrect data")
	}

	var userID uuid.UUID
	var err error
	userIDStr := req.GetId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	err = s.user.UpdateEmail(ctx, userID, req.GetEmail())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to change user")
	}

	return &emptypb.Empty{}, nil
}

func (s *UserServerAPI) UpdatePassword(
	ctx context.Context,
	req *blnggrpc.UpdatePasswordRequest,
) (*emptypb.Empty, error) {
	if req.Password == nilStr {
		return nil, status.Error(codes.InvalidArgument, "incorrect data")
	}

	err := s.user.UpdatePassword(ctx, req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to change user")
	}

	return &emptypb.Empty{}, nil
}

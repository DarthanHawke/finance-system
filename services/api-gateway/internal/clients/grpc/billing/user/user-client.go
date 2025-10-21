package user

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

type UserClient struct {
	userApi blggrpc.UserServiceClient
	logger  *zap.Logger
}

func NewUserClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*UserClient, error) {
	const op = "clients.grpc.billing.user.NewUserClient"

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

	return &UserClient{
		userApi: blggrpc.NewUserServiceClient(cc),
		logger:  logger,
	}, nil
}

func (c *UserClient) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error) {
	const op = "clients.grpc.billing.User.GetAllUsers"

	response, err := c.userApi.GetAllUsers(ctx, &blggrpc.GetAllUsersRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoUsers := response.GetUsers()
	users := make([]models.User, 0, len(protoUsers))
	for _, user := range protoUsers {
		users = append(users, models.User{
			ID:        uuid.MustParse(user.GetId().GetValue()),
			Email:     user.Email,
			FullName:  user.FullName,
			CreatedAt: user.CreatedAt.AsTime(),
			UpdatedAt: user.UpdatedAt.AsTime(),
		})
	}

	return users, nil
}

func (c *UserClient) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	const op = "clients.grpc.billing.User.GetProfile"

	response, err := c.userApi.GetProfile(ctx, &blggrpc.GetProfileRequest{
		Id: &blggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return &models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	protoUser := response.GetUser()
	return &models.User{
		ID:        uuid.MustParse(protoUser.GetId().GetValue()),
		Email:     protoUser.Email,
		FullName:  protoUser.FullName,
		CreatedAt: protoUser.CreatedAt.AsTime(),
		UpdatedAt: protoUser.UpdatedAt.AsTime(),
	}, nil
}

func (c *UserClient) UpdateName(ctx context.Context, userID uuid.UUID, fulName string) error {
	const op = "clients.grpc.billing.User.UpdateName"

	_, err := c.userApi.UpdateName(ctx, &blggrpc.UpdateNameRequest{
		Id:   &blggrpc.UUID{Value: userID.String()},
		Name: fulName,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *UserClient) UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error {
	const op = "clients.grpc.billing.User.UpdateEmail"

	_, err := c.userApi.UpdateEmail(ctx, &blggrpc.UpdateEmailRequest{
		Id:    &blggrpc.UUID{Value: userID.String()},
		Email: email,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *UserClient) UpdatePassword(ctx context.Context, password string) error {
	const op = "clients.grpc.billing.User.UpdatePassword"

	_, err := c.userApi.UpdatePassword(ctx, &blggrpc.UpdatePasswordRequest{
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

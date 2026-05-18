package auth

/*
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
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthClient struct {
	authApi blggrpc.AuthServiceClient
	logger  *zap.Logger
}

func NewAuthClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*AuthClient, error) {
	const op = "clients.grpc.billing.auth.NewAuthClient"

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

	return &AuthClient{
		authApi: blggrpc.NewAuthServiceClient(cc),
		logger:  logger,
	}, nil
}

func (c *AuthClient) RefreshSession(
	ctx context.Context,
) (*models.UserSession, error) {
	const op = "clients.grpc.billing.auth.RefreshSession"

	response, err := c.authApi.RefreshSession(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.UserSession{
		AccessToken:  response.UserSession.AccessToken,
		RefreshToken: response.UserSession.RefreshToken,
	}, nil
}

func (c *AuthClient) Register(ctx context.Context, fullName, email, password string) (*models.UserSession, error) {
	const op = "clients.grpc.billing.auth.Register"

	response, err := c.authApi.Register(ctx, &blggrpc.RegisterRequest{
		Name:     fullName,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.UserSession{
		AccessToken:  response.UserSession.AccessToken,
		RefreshToken: response.UserSession.RefreshToken,
	}, nil
}

func (c *AuthClient) Login(ctx context.Context, email string, password string) (*models.UserSession, error) {
	const op = "clients.grpc.billing.auth.Login"

	response, err := c.authApi.Login(ctx, &blggrpc.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.UserSession{
		AccessToken:  response.UserSession.AccessToken,
		RefreshToken: response.UserSession.RefreshToken,
	}, nil
}

func (c *AuthClient) Logout(ctx context.Context, userID, sessionID uuid.UUID) error {
	const op = "clients.grpc.billing.auth.Logout"

	_, err := c.authApi.Logout(ctx, &blggrpc.LogoutRequest{
		UserId:    &blggrpc.UUID{Value: userID.String()},
		SessionId: &blggrpc.UUID{Value: sessionID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *AuthClient) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "clients.grpc.billing.auth.LogoutAll"

	_, err := c.authApi.LogoutAll(ctx, &blggrpc.LogoutAllRequest{
		Id: &blggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *AuthClient) GetAllSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	const op = "clients.grpc.billing.auth.GetAllSessions"

	response, err := c.authApi.GetAllSessions(ctx, &blggrpc.GetAllSessionsRequest{
		Id: &blggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoSessions := response.GetSession()
	sessions := make([]models.Session, 0, len(protoSessions))
	for _, session := range protoSessions {
		sessions = append(sessions, models.Session{
			ID:        uuid.MustParse(session.GetId().GetValue()),
			UserID:    uuid.MustParse(session.GetUserId().GetValue()),
			UserIP:    session.UserIp,
			UserAgent: session.UserAgent,
			ExpiresAt: session.ExpiresAt.AsTime(),
		})
	}

	return sessions, nil
}
*/

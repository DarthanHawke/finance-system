package session

/* legacy code
import (
	"billing-service/internal/clients/grpc/interceptor"
	"billing-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	ssogrpc "github.com/DarthanHawke/protos-finance-system/gen/go/sso"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type SessionClient struct {
	sessionApi ssogrpc.SessionServiceClient
	logger     *zap.Logger
}

func NewSessionClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*SessionClient, error) {
	const op = "clients.grpc.sso.session.NewSessionClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "sso-service",
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

	return &SessionClient{
		sessionApi: ssogrpc.NewSessionServiceClient(cc),
		logger:     logger,
	}, nil
}

func (c *SessionClient) CreateSession(ctx context.Context, userID uuid.UUID) (*models.UserSession, error) {
	const op = "clients.grpc.sso.session.CreateSession"

	response, err := c.sessionApi.CreateSession(ctx, &ssogrpc.CreateSessionRequest{
		UserId: &ssogrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.UserSession{
		AccessToken:  response.GetAccessToken(),
		RefreshToken: response.GetRefreshToken(),
	}, nil
}

func (c *SessionClient) RefreshSession(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
) (*models.UserSession, error) {
	const op = "clients.grpc.sso.session.RefreshSession"

	response, err := c.sessionApi.RefreshSession(ctx, &ssogrpc.RefreshSessionRequest{
		UserId:       &ssogrpc.UUID{Value: userID.String()},
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.UserSession{
		AccessToken:  response.GetAccessToken(),
		RefreshToken: response.GetRefreshToken(),
	}, nil
}

func (c *SessionClient) Logout(ctx context.Context, userID, sessionId uuid.UUID) error {
	const op = "clients.grpc.sso.session.Logout"

	_, err := c.sessionApi.Logout(ctx, &ssogrpc.LogoutRequest{
		UserId:    &ssogrpc.UUID{Value: userID.String()},
		SessionId: &ssogrpc.UUID{Value: sessionId.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *SessionClient) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "clients.grpc.sso.session.LogoutAll"

	_, err := c.sessionApi.LogoutAll(ctx, &ssogrpc.LogoutAllRequest{
		UserId: &ssogrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *SessionClient) GetAll(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	const op = "clients.grpc.sso.session.GetAll"

	response, err := c.sessionApi.GetAll(ctx, &ssogrpc.GetAllRequest{
		UserId: &ssogrpc.UUID{Value: userID.String()},
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

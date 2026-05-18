package auth

/* legacy code
import (
	"billing-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Auth interface {
	RefreshSession(ctx context.Context, refreshToken string) (*models.UserSession, error)
	Register(ctx context.Context, fullName, email, password string) (*models.UserSession, error)
	Login(ctx context.Context, email, password string) (*models.UserSession, error)
	Logout(ctx context.Context, targetUserID, targetSessionID uuid.UUID) error
	LogoutAll(ctx context.Context, targetID uuid.UUID) error
	GetAllSessions(ctx context.Context, targetID uuid.UUID) ([]models.Session, error)
}

type AuthServerAPI struct {
	blnggrpc.UnimplementedAuthServiceServer
	auth Auth
}

func NewAuthServer(gRPC *grpc.Server, auth Auth) {
	blnggrpc.RegisterAuthServiceServer(gRPC, &AuthServerAPI{auth: auth})
}

func (s *AuthServerAPI) RefreshSession(
	ctx context.Context,
	_ *emptypb.Empty,
) (*blnggrpc.RefreshSessionResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata missing")
	}

	authHeader := md.Get("x-refresh-token")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "x-refresh-token header missing")
	}

	userSession, err := s.auth.RefreshSession(ctx, authHeader[0])
	if err != nil {
		return nil, status.Error(codes.Internal, "access deneid")
	}

	return &blnggrpc.RefreshSessionResponse{
		UserSession: &blnggrpc.UserSession{
			AccessToken:  userSession.AccessToken,
			RefreshToken: userSession.RefreshToken,
		},
	}, nil
}

func (s *AuthServerAPI) Register(
	ctx context.Context,
	req *blnggrpc.RegisterRequest,
) (*blnggrpc.RegisterResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	userSession, err := s.auth.Register(ctx, req.GetName(), req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &blnggrpc.RegisterResponse{
		UserSession: &blnggrpc.UserSession{
			AccessToken:  userSession.AccessToken,
			RefreshToken: userSession.RefreshToken,
		},
	}, nil
}

func (s *AuthServerAPI) Login(
	ctx context.Context,
	req *blnggrpc.LoginRequest,
) (*blnggrpc.LoginResponse, error) {
	userSession, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &blnggrpc.LoginResponse{
		UserSession: &blnggrpc.UserSession{
			AccessToken:  userSession.AccessToken,
			RefreshToken: userSession.RefreshToken,
		},
	}, nil
}

func (s *AuthServerAPI) Logout(
	ctx context.Context,
	req *blnggrpc.LogoutRequest,
) (*emptypb.Empty, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetUserId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	var sessionID uuid.UUID
	sessionIDStr := req.GetSessionId().GetValue()
	if sessionIDStr != "" {
		sessionID, err = uuid.Parse(sessionIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	err = s.auth.Logout(ctx, userID, sessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &emptypb.Empty{}, nil
}

func (s *AuthServerAPI) LogoutAll(
	ctx context.Context,
	req *blnggrpc.LogoutAllRequest,
) (*emptypb.Empty, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	err = s.auth.LogoutAll(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &emptypb.Empty{}, nil
}

func (s *AuthServerAPI) GetAllSessions(
	ctx context.Context,
	req *blnggrpc.GetAllSessionsRequest,
) (*blnggrpc.GetAllSessionsResponse, error) {
	var userID uuid.UUID
	var err error
	userIDStr := req.GetId().GetValue()
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse user")
		}
	}

	sessions, err := s.auth.GetAllSessions(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get all sessions")
	}

	protoSessions := make([]*blnggrpc.Session, 0, len(sessions))
	for _, session := range sessions {
		protoSessions = append(protoSessions, &blnggrpc.Session{
			Id:        &blnggrpc.UUID{Value: session.ID.String()},
			UserId:    &blnggrpc.UUID{Value: session.UserID.String()},
			UserIp:    session.UserIP,
			UserAgent: session.UserAgent,
			ExpiresAt: timestamppb.New(session.ExpiresAt),
			CreatedAt: timestamppb.New(session.CreatedAt),
		})
	}

	return &blnggrpc.GetAllSessionsResponse{Session: protoSessions}, nil
}
*/

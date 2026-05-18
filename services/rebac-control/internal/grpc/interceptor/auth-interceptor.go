package interceptor

/* legacy code
import (
	"billing-service/internal/models"
	"context"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	jwtManage AuthJWTManage
	cache     RedisCacheManage
}

func NewAuthInterceptor(
	jwtManage AuthJWTManage,
	cache RedisCacheManage,
) *AuthInterceptor {
	return &AuthInterceptor{
		jwtManage: jwtManage,
		cache:     cache,
	}
}

type AuthJWTManage interface {
	ValidateAccessToken(tokenString string) (*models.AccessTokenClaims, error)
}

type RedisCacheManage interface {
	Exists(ctx context.Context, key string) (bool, error)
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Пропускаем аутентификацию для определенных методов
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Извлекаем токен из метаданных
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}
		ctx = metadata.NewOutgoingContext(ctx, md) // передаем метаданные дальше

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
		}

		accessToken := strings.TrimPrefix(authHeader[0], "Bearer ")

		// Валидация токена
		tokenClaims, err := i.jwtManage.ValidateAccessToken(accessToken)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		cacheKey := fmt.Sprintf("invalid_session:%s", tokenClaims.SessionID.String())
		exists, err := i.cache.Exists(ctx, cacheKey)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "cache is not provided")
		}
		if exists {
			return nil, status.Error(codes.Unauthenticated, "authorization token is expired")
		}

		// Добавляем userID и sessionID в контекст
		ctx = context.WithValue(ctx, models.UserIDKey, tokenClaims.UserID)
		ctx = context.WithValue(ctx, models.SessionIDKey, tokenClaims.SessionID)
		ctx = context.WithValue(ctx, models.SessionTTL, tokenClaims.ExpiresAt.Time)

		return handler(ctx, req)
	}
}

func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/billing.AuthService/Login",
		"/billing.AuthService/Register",
	}
	return slices.Contains(publicMethods, method)
}
*/

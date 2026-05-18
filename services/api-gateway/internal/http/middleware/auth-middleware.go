package middleware

/*
import (
	authCookie "client-service/internal/http/cookie"
	"client-service/internal/models"
	"context"
	"net/http"
	"slices"
	"strings"

	"google.golang.org/grpc/metadata"
)

type contextKey string

// Константы для ключей контекста
const (
	ContextKeyAccessToken  contextKey = "access_token"
	ContextKeyRefreshToken contextKey = "refresh_token"
)

type AuthService interface {
	RefreshSession(ctx context.Context) (*models.UserSession, error)
}

// BillingAuthMiddleware создает middleware для аутентификации в Billing сервисе
func BillingAuthMiddleware(authService AuthService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Игнорируем публичные маршруты
			if isPublicRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Пытаемся получить токены
			accessToken, err := r.Cookie("access_token")
			if err != nil {
				authCookie.RespondUnauthorized(w)
				return
			}

			refreshToken, err := r.Cookie("refresh_token")
			if err != nil {
				authCookie.RespondUnauthorized(w)
				return
			}

			// Проверяем и обновляем токен при необходимости
			ctx := r.Context()
			if authCookie.IsTokenExpired(accessToken.Value) {
				newTokens, err := authService.RefreshSession(ctx)
				if err != nil {
					authCookie.RespondUnauthorized(w)
					return
				}

				authCookie.SetTokenCookies(w, newTokens.AccessToken, newTokens.RefreshToken)
				accessToken.Value = newTokens.AccessToken
			}
			// Добавляем токен в gRPC контекст
			md := metadata.New(map[string]string{
				"authorization":   "Bearer " + accessToken.Value,
				"x-refresh-token": refreshToken.Value,
			})
			ctx = metadata.NewOutgoingContext(ctx, md)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isPublicRoute(path string) bool {
	if strings.HasPrefix(path, "/swagger/") {
		return true
	}
	if strings.HasPrefix(path, "/static/") {
		return true
	}
	publicRoutes := []string{
		"/favicon.ico",
		"/",
		"/auth",
		"/auth/login",
		"/auth/register",
	}
	return slices.Contains(publicRoutes, path)
}
*/

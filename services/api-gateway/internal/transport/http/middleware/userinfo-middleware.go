package middleware

/* legacy code
import (
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

func UserInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		userAgent := r.UserAgent()

		ctx := metadata.AppendToOutgoingContext(r.Context(), "x-client-ip", ip)
		ctx = metadata.AppendToOutgoingContext(ctx, "x-client-agent", userAgent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	if strings.Contains(ip, ",") {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	return ip
}
*/

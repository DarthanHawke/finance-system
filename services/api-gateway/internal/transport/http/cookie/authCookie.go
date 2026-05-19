package cookie

/* legacy code
import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
)

func RespondUnauthorized(w http.ResponseWriter) {
	// Можно очистить куки если они невалидные
	http.SetCookie(w, &http.Cookie{
		Name:   "access_token",
		Value:  "",
		MaxAge: -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		MaxAge: -1,
	})
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// setTokenCookies устанавливает HTTP-only куки с токенами
func SetTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true, // Только для HTTPS
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true, // Только для HTTPS
		Path:     "/",
	})
}

func IsTokenExpired(tokenString string) bool {
	// Парсим токен без проверки подписи (нам нужно только поле exp)
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		// Если токен невалидный, считаем его просроченным
		return true
	}

	// Получаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return true
	}

	// Извлекаем время истечения (exp)
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return true
	}

	// Проверяем, не истек ли срок
	return exp.Before(time.Now())
}
*/

package models

/* legacy code
import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserSession struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

// Claims для Access токена
type AccessTokenClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

type contextKey string

const (
	UserIDKey    contextKey = "userID"
	SessionIDKey contextKey = "sessionID"
	SessionTTL   contextKey = "sessionTTL"
)

type SessionBlacklist struct {
	CacheKey string
	TTL      time.Duration
}
*/

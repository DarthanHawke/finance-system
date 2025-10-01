package jwt

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Конфиг
type TokenConfig struct {
	accessTokenTTL  time.Duration // Время жизни Access токена
	refreshTokenTTL time.Duration // Время жизни Refresh токена
	issuer          string        // Идентификатор издателя
}

// Ключи
type TokenKeys struct {
	publicKey *rsa.PublicKey // Публичный ключ для проверки
}

// Генератор токенов
type TokenValidator struct {
	config *TokenConfig
	keys   *TokenKeys
}

func NewTokenValidator(
	accessTokenTTL, refreshTokenTTL time.Duration,
	issuer string,
	publicKeyPath string,
) (*TokenValidator, error) {
	keys, err := NewTokenKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}
	return &TokenValidator{
		config: NewTokenConfig(accessTokenTTL, refreshTokenTTL, issuer),
		keys:   keys,
	}, nil
}

func NewTokenConfig(accessTokenTTL, refreshTokenTTL time.Duration, issuer string) *TokenConfig {
	return &TokenConfig{
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		issuer:          issuer,
	}
}

func NewTokenKey(publicKeyPath string) (*TokenKeys, error) {
	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}

	pubBlock, _ := pem.Decode(pubBytes)

	pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &TokenKeys{
		publicKey: pubKey.(*rsa.PublicKey),
	}, nil
}

// ValidateAccessToken проверяет Access токен
func (g *TokenValidator) ValidateAccessToken(tokenString string) (*models.AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&models.AccessTokenClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return g.keys.publicKey, nil
		})
	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, billingerr.ErrTokenExpired
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*models.AccessTokenClaims); ok && token.Valid {
		if claims.TokenType != "access" {
			return nil, billingerr.ErrInavlidToken
		}
		return claims, nil
	}

	return nil, billingerr.ErrInavlidToken
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	RefreshTokenHash string    `json:"refresh_token_hash" db:"refresh_token_hash"`
	UserIP           string    `json:"user_ip" db:"user_ip"`
	UserAgent        string    `json:"user_agent" db:"user_agent"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type UserSession struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

// RegisterRequest - DTO для регистрации
type RegisterRequest struct {
	FullName string `json:"fullName" validate:"required,max=255"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

// LoginRequest - DTO для авторизации
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,max=255"`
}

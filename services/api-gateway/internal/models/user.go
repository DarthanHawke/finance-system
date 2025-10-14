package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type GetAllUsersRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// UpdateProfileRequest - DTO для обновления профиля
type UpdateNameRequest struct {
	UserID   uuid.UUID `json:"userID"`
	FullName string    `json:"fullName" validate:"required,max=255"`
}

type UpdateEmailRequest struct {
	UserID uuid.UUID `json:"userID"`
	Email  string    `json:"email" validate:"required,email,max=255"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=255"`
}

type GetCurrencyRateRequest struct {
	FromCurrency string `json:"from_currency" validate:"required,alpha,len=3"`
	ToCurrency   string `json:"to_currency" validate:"required,alpha,len=3"`
}

type CurrencyRateResponse struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
}

package mocks

import (
	"context"
	"transaction-service/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) CreateTransaction(
	ctx context.Context,
	req *models.CreateTransactionRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockTransactionManager) GetTransaction(
	ctx context.Context,
	req *models.GetTransactionRequest,
) (models.GetTransactionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(models.GetTransactionResponse), args.Error(1)
}

func (m *MockTransactionManager) UpdateTransactionStatus(
	ctx context.Context,
	req *models.UpdateTransactionStatusRequest,
) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

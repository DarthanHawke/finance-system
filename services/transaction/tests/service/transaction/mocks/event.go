package mocks

import (
	"context"
	"transaction-service/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockEventManager struct {
	mock.Mock
}

func (m *MockEventManager) CreateEvent(
	ctx context.Context,
	req *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

package mocks

import (
	"account-service/internal/models"
	"context"

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

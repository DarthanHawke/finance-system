package mocks

import (
	"account-service/internal/models"
	"context"

	"github.com/stretchr/testify/mock"
)

type MockBalanceManager struct {
	mock.Mock
}

func (m *MockBalanceManager) FreezeBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) UnfreezeBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) ReserveDeposit(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) UnreserveDeposit(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) WithdrawBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) DepositBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

func (m *MockBalanceManager) BlockAccountWithEvent(
	ctx context.Context,
	req *models.UpdateAccountRequest,
	event *models.CreateEventRequest,
) error {
	args := m.Called(ctx, req, event)
	return args.Error(0)
}

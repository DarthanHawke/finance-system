package mocks

import (
	"transaction-service/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockIso8583Manager struct {
	mock.Mock
}

func (m *MockIso8583Manager) CreateTransactionFromISO(
	msg *models.ISO8583Message,
) (*models.CreateTransactionRequest, error) {
	args := m.Called(msg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CreateTransactionRequest), args.Error(1)
}

func (m *MockIso8583Manager) CreateFinancialRequest(
	transaction *models.Transaction,
) (*models.ISO8583Message, error) {
	args := m.Called(transaction)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ISO8583Message), args.Error(1)
}

func (m *MockIso8583Manager) CreateFinancialResponse(
	transaction *models.Transaction,
	success bool,
	reason string,
) (*models.ISO8583Message, error) {
	args := m.Called(transaction, success, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ISO8583Message), args.Error(1)
}

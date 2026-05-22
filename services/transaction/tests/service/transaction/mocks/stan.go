package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type MockStanManager struct {
	mock.Mock
}

func (m *MockStanManager) GetNextSTAN(transactionType, senderType, recipientType string, date time.Time) (string, error) {
	args := m.Called(transactionType, senderType, recipientType, date)
	return args.String(0), args.Error(1)
}

func (m *MockStanManager) GetCurrentSTAN(transactionType, senderType, recipientType string, date time.Time) (string, error) {
	args := m.Called(transactionType, senderType, recipientType, date)
	return args.String(0), args.Error(1)
}

func (m *MockStanManager) ResetCounters() {
	m.Called()
}

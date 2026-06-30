// Пакет postgres реализовывает работу с PostgreSQL
package postgres

import (
	"context"
	"transaction-service/internal/models"

	"github.com/jmoiron/sqlx"
)

// SagaCoordinator реализует методы для продвижения саги в одной транзакции
type SagaCoordinator struct {
	db                    *Database
	transactionRepository *TransactionRepository
	sagaRepository        *SagaRepository
	eventRepository       *EventRepository
}

// NewSagaRepository создает обертку над бд, реализуюзую методы для продвижения саги в одной транзакции
func NewSagaCoordinator(
	db *Database,
	transactionRepository *TransactionRepository,
	sagaRepository *SagaRepository,
	eventRepository *EventRepository,
) *SagaCoordinator {
	return &SagaCoordinator{
		db:                    db,
		transactionRepository: transactionRepository,
		sagaRepository:        sagaRepository,
		eventRepository:       eventRepository,
	}
}

// Advance продвигает сагу(создает событие, новый шаг, обновляет статус саги)
func (c *SagaCoordinator) Advance(ctx context.Context, req *models.AdvanceRequest) error {
	return c.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, event := range req.Events {
			if err := c.eventRepository.InsertEvent(ctx, tx, &event); err != nil {
				return err
			}
		}
		for _, step := range req.Steps {
			if err := c.sagaRepository.InsertSagaStep(ctx, tx, &step); err != nil {
				return err
			}
		}
		return c.transactionRepository.UpdateSagaState(ctx, tx, req.UpdateSagaStateRequest)
	})
}

// AdvanceWithFirstStep первый шаг саги(создает транзакцию, событие, новый шаг, обновляет статус саги)
func (c *SagaCoordinator) AdvanceWithFirstStep(ctx context.Context, req *models.AdvanceWithFirstStepRequest) error {
	return c.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		if err := c.transactionRepository.InsertTransaction(ctx, tx, &req.InsertTransactionRequest); err != nil {
			return err
		}
		if err := c.transactionRepository.InsertParties(ctx, tx, &req.InsertPartiesRequest); err != nil {
			return err
		}
		for _, event := range req.Events {
			if err := c.eventRepository.InsertEvent(ctx, tx, &event); err != nil {
				return err
			}
		}
		for _, step := range req.Steps {
			if err := c.sagaRepository.InsertSagaStep(ctx, tx, &step); err != nil {
				return err
			}
		}
		return c.transactionRepository.UpdateSagaState(ctx, tx, req.UpdateSagaStateRequest)
	})
}

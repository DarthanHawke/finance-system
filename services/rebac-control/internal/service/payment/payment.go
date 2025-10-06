package payment

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentService struct {
	accountManage   AccountManage
	externalPayment ExternalPayment
	internalPayment InternalPayment
	roleManage      RoleManage
	ibanGenerator   IbanGenerator
	logger          *zap.Logger
}

func NewPaymentService(
	accountManage AccountManage,
	externalPayment ExternalPayment,
	internalPayment InternalPayment,
	roleManage RoleManage,
	ibanGenerator IbanGenerator,
	logger *zap.Logger,
) *PaymentService {
	return &PaymentService{
		accountManage:   accountManage,
		externalPayment: externalPayment,
		internalPayment: internalPayment,
		roleManage:      roleManage,
		ibanGenerator:   ibanGenerator,
		logger:          logger.With(zap.String("component", "billing_service")),
	}
}

type ExternalPayment interface {
	CreatePayment(
		ctx context.Context,
		sender, receiver string,
		amount float64,
		currency string, description string,
	) (uuid.UUID, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error)
	UpdateStatusPayment(ctx context.Context, paymentID uuid.UUID, status string) error
	CancelPayment(ctx context.Context, paymentID uuid.UUID) error
}

type InternalPayment interface {
	GetPaymentsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	Deposit(ctx context.Context, operationID uuid.UUID, accountCode string, amount float64, paymentID uuid.UUID, description string) error
	Withdraw(ctx context.Context, operationID uuid.UUID, accountCode string, amount float64, paymentID uuid.UUID, description string) error
	Transfer(ctx context.Context, operationID uuid.UUID, fromAccount, toAccount *models.CurrencyAccount, amount float64, paymentID uuid.UUID, description string) error
	ConvertCurrency(ctx context.Context, operationID uuid.UUID, fromaccountCode, toaccountCode *models.CurrencyAccount, amount float64, description string) error
	GetOperationHistory(ctx context.Context, accountCode string, limit, offset int) ([]models.BalanceOperation, error)
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

type RoleManage interface {
	CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
}

type AccountManage interface {
	GetAccount(ctx context.Context, accountCode string) (*models.CurrencyAccount, error)
	GetAccountId(ctx context.Context, account_code string) (uuid.UUID, error)
	GetBalance(ctx context.Context, accountCode string) (float64, error)
}

type IbanGenerator interface {
	ValidateInternal(iban string) error
	ValidateExternal(iban string) error
}

// Transfer переводит средства
func (s *PaymentService) Transfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "service.payment.Transfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.PaymentCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	senderAccount, err := s.accountManage.GetAccount(ctx, sender)
	if err != nil {
		logger.Error("sender account cant get",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("sender", sender),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if senderAccount.UserID != userID {
		logger.Error("sender account is non-user",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("sender", sender),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}
	receiverAccount, err := s.accountManage.GetAccount(ctx, receiver)
	if err != nil {
		if !errors.Is(err, billingerr.ErrAccountNotFound) {
			logger.Error("receiver account cant get",
				zap.Error(err),
				zap.String("UserID", userID.String()),
				zap.String("receiver", receiver),
			)
			return uuid.Nil, fmt.Errorf("%s: %w", op, err)
		}
	}
	var paymentID uuid.UUID
	if receiverAccount == nil {
		err := s.ibanGenerator.ValidateExternal(receiver)
		if err != nil {
			logger.Error("invalid receiver payment external account",
				zap.Error(err),
				zap.String("receiver", receiver),
			)
			return uuid.Nil, fmt.Errorf("invalid receiver payment account: %v: %w", op, err)
		}
		paymentID, err = s.externalTransfer(ctx, sender, receiver, amount, senderAccount.Currency, description)
		if err != nil {
			logger.Error("cant do external transfer", zap.Error(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransfer)
		}
	} else {
		err := s.ibanGenerator.ValidateInternal(receiver)
		if err != nil {
			logger.Error("invalid receiver payment internal account",
				zap.Error(err),
				zap.String("receiver", receiver),
			)
			return uuid.Nil, fmt.Errorf("invalid receiver payment account: %v: %w", op, err)
		}

		paymentID, err = s.internalTransfer(ctx, senderAccount, receiverAccount, amount, description)
		if err != nil {
			logger.Error("cant do internal transfer", zap.Error(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransfer)
		}
	}

	return paymentID, nil
}

// externalTtransfer переводит средства между другим сервисом
func (s *PaymentService) externalTransfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	currency, description string,
) (uuid.UUID, error) {
	const op = "service.payment.externalTtransfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.PaymentCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	balance, err := s.accountManage.GetBalance(ctx, sender)
	if err != nil {
		logger.Error("failed to check balance",
			zap.Error(err),
			zap.String("currency", currency),
		)
		return uuid.Nil, fmt.Errorf("%s: failed to check balance: %w", op, err)
	}

	if balance < amount {
		logger.Warn("insufficient funds",
			zap.Float64("available", balance),
			zap.Float64("requested", amount),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInsufficientFunds)
	}

	paymentId, err := s.externalPayment.CreatePayment(ctx, sender, receiver, amount, currency, description)
	if err != nil {
		logger.Error("cant create payment", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentCancel)
	}

	// 4. Списание средств с резервированием (Saga pattern)
	operationID := uuid.New()
	if err := s.internalPayment.Withdraw(ctx, operationID, sender, amount, paymentId, description); err != nil {
		// Компенсирующее действие - отмена платежа
		if cancelErr := s.externalPayment.CancelPayment(ctx, paymentId); cancelErr != nil {
			logger.Error("failed to cancel payment after withdraw failure",
				zap.NamedError("withdraw_error", err),
				zap.NamedError("cancel_error", cancelErr),
			)
		}

		logger.Error("failed to withdraw funds",
			zap.Error(err),
			zap.String("payment_id", paymentId.String()),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentCancel)
	}

	err = s.roleManage.CreateEntityWithID(ctx, paymentId, models.PaymentEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, paymentId, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}
	return paymentId, nil
}

// internalTransfer переводит средства между счетами сервиса Payment
func (s *PaymentService) internalTransfer(
	ctx context.Context,
	sender, receiver *models.CurrencyAccount,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "service.payment.internalTransfer"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("sender", sender.AccountCode),
		zap.String("receiver", receiver.AccountCode),
		zap.Float64("amount", amount),
	)

	if amount <= 0 {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	// Проверка прав доступа отправителя
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.PaymentCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Создаем платеж
	paymentID, err := s.externalPayment.CreatePayment(
		ctx,
		sender.AccountCode,
		receiver.AccountCode,
		amount,
		sender.Currency,
		description,
	)
	if err != nil {
		logger.Error("failed to create payment for transfer",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем перевод
	if err := s.internalPayment.Transfer(ctx, operationID, sender, receiver, amount, paymentID, description); err != nil {
		logger.Error("failed to transfer",
			zap.Error(err),
		)
		// Отменяем платеж в случае ошибки
		_ = s.externalPayment.CancelPayment(ctx, paymentID)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	err = s.roleManage.CreateEntityWithID(ctx, paymentID, models.PaymentEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, paymentID, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = s.externalPayment.UpdateStatusPayment(ctx, paymentID, "completed")
	if err != nil {
		return paymentID, fmt.Errorf("cant update payment status to completed: %v", err)
	}

	return paymentID, nil
}

// Deposit пополняет счет пользователя
func (s *PaymentService) Deposit(ctx context.Context, accountCode string, amount float64, description string) (uuid.UUID, error) {
	const op = "service.payment.Deposit"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("accountCode", accountCode),
		zap.Float64("amount", amount),
	)

	if amount <= 0 {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.PaymentCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Проверяем, что счет принадлежит пользователю
	account, err := s.accountManage.GetAccount(ctx, accountCode)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if account.UserID != userID {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	// Создаем платеж (внешнее пополнение)
	paymentID, err := s.externalPayment.CreatePayment(
		ctx,
		"external",  // источник - внешний
		accountCode, // получатель
		amount,
		account.Currency,
		description,
	)
	if err != nil {
		logger.Error("failed to create payment for deposit",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем пополнение
	if err := s.internalPayment.Deposit(ctx, operationID, accountCode, amount, paymentID, description); err != nil {
		logger.Error("failed to deposit",
			zap.Error(err),
		)
		// Отменяем платеж в случае ошибки
		_ = s.externalPayment.CancelPayment(ctx, paymentID)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	err = s.roleManage.CreateEntityWithID(ctx, paymentID, models.PaymentEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, paymentID, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = s.externalPayment.UpdateStatusPayment(ctx, paymentID, "completed")
	if err != nil {
		return paymentID, fmt.Errorf("cant update payment status to completed: %v", err)
	}

	return paymentID, nil
}

// GetPayment - возращает информацию о платеже
func (s *PaymentService) GetPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) (*models.Payment, error) {
	const op = "service.payment.GetPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting getting payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, paymentID, models.PaymentRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.PaymentRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	payment, err := s.externalPayment.GetPayment(ctx, paymentID)
	if err != nil {
		logger.Error("cant get payment", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}
	return payment, nil
}

// GetAllPayment - возвращает все платежи пользователя
func (s *PaymentService) GetAllPayment(ctx context.Context, targetID uuid.UUID) ([]models.Payment, error) {
	const op = "service.payment.GetAllPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.PaymentReadAll)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentReadAll),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.PaymentReadAll),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	paymentsId, err := s.internalPayment.GetPaymentsByUser(ctx, targetID)
	if err != nil {
		logger.Error("cant get payment", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}
	if paymentsId == nil {
		logger.Info("payments not found")
		return nil, nil
	}

	var paymants []models.Payment
	for _, paymentID := range paymentsId {
		payment, err := s.externalPayment.GetPayment(ctx, paymentID)
		if err != nil {
			logger.Error("cant get payment", zap.Error(err))
			payment = nil
		}
		if payment != nil {
			paymants = append(paymants, *payment)
		}
	}

	return paymants, nil
}

func (s *PaymentService) UpdateStatusPayment(
	ctx context.Context,
	paymentID uuid.UUID,
	status string,
) error {
	const op = "service.payment.UpdateStatusPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, paymentID, models.PaymentUpdateStatus)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentUpdateStatus),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentUpdateStatus),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.externalPayment.UpdateStatusPayment(ctx, paymentID, status)
	if err != nil {
		logger.Error("cant update payment", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrUpdatePayment)
	}

	if status == models.PaymentStatusRefunded || status == models.PaymentStatusCancelled {
		payment, err := s.externalPayment.GetPayment(ctx, paymentID)
		if err != nil {
			logger.Error("cant get payment", zap.Error(err))

			return fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
		}

		operationID := uuid.New()
		if err := s.internalPayment.Deposit(ctx, operationID, payment.Sender, payment.Amount, paymentID, "Refund of funds"); err != nil {
			logger.Error("failed to deposit",
				zap.Error(err),
			)
			_ = s.externalPayment.CancelPayment(ctx, paymentID)
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (s *PaymentService) CancelPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) error {
	const op = "service.payment.UpdateStatusPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Canceling payment")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, paymentID, models.PaymentCancel)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCancel),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCancel),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.externalPayment.CancelPayment(ctx, paymentID)
	if err != nil {
		logger.Error("cant cancel payment", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrUpdatePayment)
	}

	payment, err := s.externalPayment.GetPayment(ctx, paymentID)
	if err != nil {
		logger.Error("cant get payment", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrGetPayment)
	}

	operationID := uuid.New()
	if err := s.internalPayment.Deposit(ctx, operationID, payment.Sender, payment.Amount, paymentID, "payment cancelled"); err != nil {
		logger.Error("failed to deposit",
			zap.Error(err),
		)
		_ = s.externalPayment.CancelPayment(ctx, paymentID)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует средства между счетами в разных валютах
func (s *PaymentService) ConvertCurrency(ctx context.Context, fromaccountCode, toaccountCode string, amount float64, description string) (uuid.UUID, error) {
	const op = "service.payment.ConvertCurrency"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromaccountCode", fromaccountCode),
		zap.String("toaccountCode", toaccountCode),
		zap.Float64("amount", amount),
	)

	if amount <= 0 {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	// Проверка прав доступа
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.PaymentCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.PaymentCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Проверяем, что оба счета принадлежат пользователю
	fromAccount, err := s.accountManage.GetAccount(ctx, fromaccountCode)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if fromAccount.UserID != userID {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	toAccount, err := s.accountManage.GetAccount(ctx, toaccountCode)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if toAccount.UserID != userID {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем конвертацию
	if err := s.internalPayment.ConvertCurrency(ctx, operationID, fromAccount, toAccount, amount, description); err != nil {
		logger.Error("failed to convert currency",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// GetOperationHistory возвращает историю операций по счету
func (s *PaymentService) GetOperationHistory(ctx context.Context, accountCode string, limit, offset int) ([]models.BalanceOperation, error) {
	const op = "service.payment.GetOperationHistory"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("accountCode", accountCode),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	accountId, err := s.accountManage.GetAccountId(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get account uuid",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Account code", accountCode),
		)
		return nil, fmt.Errorf("failed to get account uuid: %v: %w", op, err)
	}

	// Проверка прав доступа
	allowed, err := s.roleManage.CheckPermission(ctx, userID, accountId, models.AccountRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Получение истории операций
	operations, err := s.internalPayment.GetOperationHistory(ctx, accountCode, limit, offset)
	if err != nil {
		logger.Error("failed to get operation history",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return operations, nil
}

// UpdateCurrencyRate обновляет курс валют (админская функция)
func (s *PaymentService) UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error {
	const op = "service.payment.UpdateCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromCurrency", fromCurrency),
		zap.String("toCurrency", toCurrency),
		zap.Float64("rate", rate),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	currencyEntityId, err := s.roleManage.GetEntityID(ctx, models.CurrencyEntity)
	if err != nil {
		return fmt.Errorf("%s: failed to get payment system entity: %w", op, err)
	}

	// Проверка прав доступа (только для администраторов)
	allowed, err := s.roleManage.CheckPermission(ctx, userID, currencyEntityId, models.CurrencyUpdate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.CurrencyUpdate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.CurrencyUpdate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	if err := s.internalPayment.UpdateCurrencyRate(ctx, fromCurrency, toCurrency, rate); err != nil {
		logger.Error("failed to update currency rate",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
func (s *PaymentService) GetCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
) (float64, error) {
	const op = "service.payment.GetCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromCurrency", fromCurrency),
		zap.String("toCurrency", toCurrency),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return 0, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	// Проверка прав доступа
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.CurrencyRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.CurrencyRead),
		)
		return 0, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.CurrencyRead),
		)
		return 0, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Получение курса валют
	rate, err := s.internalPayment.GetCurrencyRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		logger.Error("failed to get currency rate",
			zap.Error(err),
		)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if rate <= 0 {
		logger.Warn("invalid currency rate",
			zap.Float64("rate", rate),
		)
		return 0, fmt.Errorf("%s: %w", op, billingerr.ErrCurrencyRateNotFound)
	}

	return rate, nil
}

package transaction

/* legacy code
import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransactionService struct {
	accountManage       AccountManage
	externalTransaction ExternalTransaction
	internalTransaction InternalTransaction
	roleManage          RoleManage
	ibanGenerator       IbanGenerator
	logger              *zap.Logger
}

func NewTransactionService(
	accountManage AccountManage,
	externalTransaction ExternalTransaction,
	internalTransaction InternalTransaction,
	roleManage RoleManage,
	ibanGenerator IbanGenerator,
	logger *zap.Logger,
) *TransactionService {
	return &TransactionService{
		accountManage:       accountManage,
		externalTransaction: externalTransaction,
		internalTransaction: internalTransaction,
		roleManage:          roleManage,
		ibanGenerator:       ibanGenerator,
		logger:              logger.With(zap.String("component", "billing_service")),
	}
}

type ExternalTransaction interface {
	CreateTransaction(
		ctx context.Context,
		sender, receiver string,
		amount float64,
		currency string, description string,
	) (uuid.UUID, error)
	GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error)
	UpdateStatusTransaction(ctx context.Context, transactionID uuid.UUID, status string) error
	CancelTransaction(ctx context.Context, transactionID uuid.UUID) error
}

type InternalTransaction interface {
	GetTransactionsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	Deposit(ctx context.Context, operationID uuid.UUID, accountCode string, amount float64, transactionID uuid.UUID, description string) error
	Withdraw(ctx context.Context, operationID uuid.UUID, accountCode string, amount float64, transactionID uuid.UUID, description string) error
	Transfer(ctx context.Context, operationID uuid.UUID, fromAccount, toAccount *models.CurrencyAccount, amount float64, transactionID uuid.UUID, description string) error
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
func (s *TransactionService) Transfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "service.transaction.Transfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransactionCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
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
	var transactionID uuid.UUID
	if receiverAccount == nil {
		err := s.ibanGenerator.ValidateExternal(receiver)
		if err != nil {
			logger.Error("invalid receiver transaction external account",
				zap.Error(err),
				zap.String("receiver", receiver),
			)
			return uuid.Nil, fmt.Errorf("invalid receiver transaction account: %v: %w", op, err)
		}
		transactionID, err = s.externalTransfer(ctx, sender, receiver, amount, senderAccount.Currency, description)
		if err != nil {
			logger.Error("cant do external transfer", zap.Error(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransfer)
		}
	} else {
		err := s.ibanGenerator.ValidateInternal(receiver)
		if err != nil {
			logger.Error("invalid receiver transaction internal account",
				zap.Error(err),
				zap.String("receiver", receiver),
			)
			return uuid.Nil, fmt.Errorf("invalid receiver transaction account: %v: %w", op, err)
		}

		transactionID, err = s.internalTransfer(ctx, senderAccount, receiverAccount, amount, description)
		if err != nil {
			logger.Error("cant do internal transfer", zap.Error(err))
			return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransfer)
		}
	}

	return transactionID, nil
}

// externalTtransfer переводит средства между другим сервисом
func (s *TransactionService) externalTransfer(
	ctx context.Context,
	sender, receiver string,
	amount float64,
	currency, description string,
) (uuid.UUID, error) {
	const op = "service.transaction.externalTtransfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransactionCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
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

	transactionId, err := s.externalTransaction.CreateTransaction(ctx, sender, receiver, amount, currency, description)
	if err != nil {
		logger.Error("cant create transaction", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransactionCancel)
	}

	// 4. Списание средств с резервированием (Saga pattern)
	operationID := uuid.New()
	if err := s.internalTransaction.Withdraw(ctx, operationID, sender, amount, transactionId, description); err != nil {
		// Компенсирующее действие - отмена платежа
		if cancelErr := s.externalTransaction.CancelTransaction(ctx, transactionId); cancelErr != nil {
			logger.Error("failed to cancel transaction after withdraw failure",
				zap.NamedError("withdraw_error", err),
				zap.NamedError("cancel_error", cancelErr),
			)
		}

		logger.Error("failed to withdraw funds",
			zap.Error(err),
			zap.String("transaction_id", transactionId.String()),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransactionCancel)
	}

	err = s.roleManage.CreateEntityWithID(ctx, transactionId, models.TransactionEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, transactionId, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}
	return transactionId, nil
}

// internalTransfer переводит средства между счетами сервиса Transaction
func (s *TransactionService) internalTransfer(
	ctx context.Context,
	sender, receiver *models.CurrencyAccount,
	amount float64,
	description string,
) (uuid.UUID, error) {
	const op = "service.transaction.internalTransfer"

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
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Создаем платеж
	transactionID, err := s.externalTransaction.CreateTransaction(
		ctx,
		sender.AccountCode,
		receiver.AccountCode,
		amount,
		sender.Currency,
		description,
	)
	if err != nil {
		logger.Error("failed to create transaction for transfer",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем перевод
	if err := s.internalTransaction.Transfer(ctx, operationID, sender, receiver, amount, transactionID, description); err != nil {
		logger.Error("failed to transfer",
			zap.Error(err),
		)
		// Отменяем платеж в случае ошибки
		_ = s.externalTransaction.CancelTransaction(ctx, transactionID)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	err = s.roleManage.CreateEntityWithID(ctx, transactionID, models.TransactionEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, transactionID, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = s.externalTransaction.UpdateStatusTransaction(ctx, transactionID, "completed")
	if err != nil {
		return transactionID, fmt.Errorf("cant update transaction status to completed: %v", err)
	}

	return transactionID, nil
}

// Deposit пополняет счет пользователя
func (s *TransactionService) Deposit(ctx context.Context, accountCode string, amount float64, description string) (uuid.UUID, error) {
	const op = "service.transaction.Deposit"

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

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionCreate),
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
	transactionID, err := s.externalTransaction.CreateTransaction(
		ctx,
		"external",  // источник - внешний
		accountCode, // получатель
		amount,
		account.Currency,
		description,
	)
	if err != nil {
		logger.Error("failed to create transaction for deposit",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем пополнение
	if err := s.internalTransaction.Deposit(ctx, operationID, accountCode, amount, transactionID, description); err != nil {
		logger.Error("failed to deposit",
			zap.Error(err),
		)
		// Отменяем платеж в случае ошибки
		_ = s.externalTransaction.CancelTransaction(ctx, transactionID)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	err = s.roleManage.CreateEntityWithID(ctx, transactionID, models.TransactionEntity)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, transactionID, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = s.externalTransaction.UpdateStatusTransaction(ctx, transactionID, "completed")
	if err != nil {
		return transactionID, fmt.Errorf("cant update transaction status to completed: %v", err)
	}

	return transactionID, nil
}

// GetTransaction - возращает информацию о платеже
func (s *TransactionService) GetTransaction(
	ctx context.Context,
	transactionID uuid.UUID,
) (*models.Transaction, error) {
	const op = "service.transaction.GetTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting getting transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrTransactionCancel)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, transactionID, models.TransactionRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	transaction, err := s.externalTransaction.GetTransaction(ctx, transactionID)
	if err != nil {
		logger.Error("cant get transaction", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}
	return transaction, nil
}

// GetAllTransaction - возвращает все платежи пользователя
func (s *TransactionService) GetAllTransaction(ctx context.Context, targetID uuid.UUID) ([]models.Transaction, error) {
	const op = "service.transaction.GetAllTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.TransactionReadAll)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionReadAll),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionReadAll),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	transactionsId, err := s.internalTransaction.GetTransactionsByUser(ctx, targetID)
	if err != nil {
		logger.Error("cant get transaction", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}
	if transactionsId == nil {
		logger.Info("transactions not found")
		return nil, nil
	}

	var paymants []models.Transaction
	for _, transactionID := range transactionsId {
		transaction, err := s.externalTransaction.GetTransaction(ctx, transactionID)
		if err != nil {
			logger.Error("cant get transaction", zap.Error(err))
			transaction = nil
		}
		if transaction != nil {
			paymants = append(paymants, *transaction)
		}
	}

	return paymants, nil
}

func (s *TransactionService) UpdateStatusTransaction(
	ctx context.Context,
	transactionID uuid.UUID,
	status string,
) error {
	const op = "service.transaction.UpdateStatusTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, transactionID, models.TransactionUpdateStatus)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionUpdateStatus),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionUpdateStatus),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.externalTransaction.UpdateStatusTransaction(ctx, transactionID, status)
	if err != nil {
		logger.Error("cant update transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrUpdateTransaction)
	}

	if status == models.TransactionStatusRefunded || status == models.TransactionStatusCancelled {
		transaction, err := s.externalTransaction.GetTransaction(ctx, transactionID)
		if err != nil {
			logger.Error("cant get transaction", zap.Error(err))

			return fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
		}

		operationID := uuid.New()
		if err := s.internalTransaction.Deposit(ctx, operationID, transaction.Sender, transaction.Amount, transactionID, "Refund of funds"); err != nil {
			logger.Error("failed to deposit",
				zap.Error(err),
			)
			_ = s.externalTransaction.CancelTransaction(ctx, transactionID)
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (s *TransactionService) CancelTransaction(
	ctx context.Context,
	transactionID uuid.UUID,
) error {
	const op = "service.transaction.UpdateStatusTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Canceling transaction")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, transactionID, models.TransactionCancel)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCancel),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCancel),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.externalTransaction.CancelTransaction(ctx, transactionID)
	if err != nil {
		logger.Error("cant cancel transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrUpdateTransaction)
	}

	transaction, err := s.externalTransaction.GetTransaction(ctx, transactionID)
	if err != nil {
		logger.Error("cant get transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrGetTransaction)
	}

	operationID := uuid.New()
	if err := s.internalTransaction.Deposit(ctx, operationID, transaction.Sender, transaction.Amount, transactionID, "transaction cancelled"); err != nil {
		logger.Error("failed to deposit",
			zap.Error(err),
		)
		_ = s.externalTransaction.CancelTransaction(ctx, transactionID)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует средства между счетами в разных валютах
func (s *TransactionService) ConvertCurrency(ctx context.Context, fromaccountCode, toaccountCode string, amount float64, description string) (uuid.UUID, error) {
	const op = "service.transaction.ConvertCurrency"

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
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionCreate),
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
	if err := s.internalTransaction.ConvertCurrency(ctx, operationID, fromAccount, toAccount, amount, description); err != nil {
		logger.Error("failed to convert currency",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// GetOperationHistory возвращает историю операций по счету
func (s *TransactionService) GetOperationHistory(ctx context.Context, accountCode string, limit, offset int) ([]models.BalanceOperation, error) {
	const op = "service.transaction.GetOperationHistory"

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
	operations, err := s.internalTransaction.GetOperationHistory(ctx, accountCode, limit, offset)
	if err != nil {
		logger.Error("failed to get operation history",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return operations, nil
}

// UpdateCurrencyRate обновляет курс валют (админская функция)
func (s *TransactionService) UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error {
	const op = "service.transaction.UpdateCurrencyRate"

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
		return fmt.Errorf("%s: failed to get transaction system entity: %w", op, err)
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

	if err := s.internalTransaction.UpdateCurrencyRate(ctx, fromCurrency, toCurrency, rate); err != nil {
		logger.Error("failed to update currency rate",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
func (s *TransactionService) GetCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
) (float64, error) {
	const op = "service.transaction.GetCurrencyRate"

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
	rate, err := s.internalTransaction.GetCurrencyRate(ctx, fromCurrency, toCurrency)
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
*/

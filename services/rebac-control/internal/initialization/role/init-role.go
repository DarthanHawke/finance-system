package initialization

import (
	"billing-service/internal/config"
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Initializer struct {
	roleManage     RoleManage
	authUserManage AuthUserManage
	logger         *zap.Logger
}

type RoleManage interface {
	CreateEntity(ctx context.Context, entityType string) (uuid.UUID, error)
	CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
	AddPermission(ctx context.Context, name, description string) (uuid.UUID, error)
	AssignPermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	GetPermissionByName(ctx context.Context, name string) (*models.Permission, error)
}

type AuthUserManage interface {
	Register(ctx context.Context, fullName string, email string, password string) (uuid.UUID, error)
}

func NewInitializer(
	roleManage RoleManage,
	authUserManage AuthUserManage,
	logger *zap.Logger,
) *Initializer {
	return &Initializer{
		roleManage:     roleManage,
		authUserManage: authUserManage,
		logger:         logger,
	}
}

// InitSystem выполняет инициализацию системы ReBAC
func (i *Initializer) InitSystem(ctx context.Context, systemUsers []config.User) error {
	if err := i.checkAlreadyInitialized(ctx); err == nil {
		i.logger.Info("ReBAC system already initialized")
		return nil
	}

	if err := i.createPermissions(ctx); err != nil {
		i.logger.Warn("failed to create permissions")
		return fmt.Errorf("failed to create permissions: %w", err)
	}

	if err := i.assignPermissions(ctx); err != nil {
		i.logger.Warn("failed to assign permissions")
		return fmt.Errorf("failed to assign permissions: %w", err)
	}

	if err := i.createEntitysAndRelations(ctx); err != nil {
		i.logger.Warn("failed to create entitys and relations")
		return fmt.Errorf("failed to create entitys and relations: %w", err)
	}

	if err := i.createSystemUsers(ctx, systemUsers); err != nil {
		i.logger.Warn("failed to create systrm users")
		return fmt.Errorf("failed to create systrrm users: %w", err)
	}

	i.logger.Info("ReBAC system successfully initialized")
	return nil
}

// checkAlreadyInitialized проверяет, была ли уже выполнена инициализация
func (i *Initializer) checkAlreadyInitialized(ctx context.Context) error {
	// Проверяем наличие хотя бы одного разрешения как маркер инициализации
	_, err := i.roleManage.GetPermissionByName(ctx, "transaction_create")
	if err != nil {
		return fmt.Errorf("system not initialized: %w", err)
	}
	return nil
}

// createPermissions создает все необходимые разрешения
func (i *Initializer) createPermissions(ctx context.Context) error {
	permissions := []struct {
		name        string
		description string
	}{
		{models.TransactionCreate, "Create a transaction"},
		{models.TransactionRead, "Read transaction information"},
		{models.TransactionReadAll, "Read all user's transactions"},
		{models.TransactionCancel, "Cancel a transaction"},
		{models.TransactionConvert, "Convert between currencies"},
		{models.TransactionUpdateStatus, "Update transaction status"},
		{models.UserRead, "Read user information"},
		{models.UserReadAll, "Read all users information"},
		{models.UserUpdate, "Update user information"},
		{models.AccountCreate, "Create new currency accounts"},
		{models.AccountRead, "View account balances"},
		{models.CurrencyRead, "Read currency convert rate"},
		{models.CurrencyUpdate, "Update currency convert rate"},
		{models.RelationManage, "Managment of ReBAC system"},
		{models.RelationRead, "Read of ReBAC system"},
		{models.PermissionManage, "Managment of ReBAC system"},
		{models.PermissionRead, "Read of ReBAC system"},
		{models.SessionTerminate, "Log out users"},
		{models.SessionManage, "Managment of users sessions"},
	}

	for _, p := range permissions {
		if _, err := i.roleManage.AddPermission(ctx, p.name, p.description); err != nil {
			if !errors.Is(err, billingerr.ErrAlreadyExists) {
				return fmt.Errorf("failed to create permission %s: %w", p.name, err)
			}
			i.logger.Debug("Permission already exists", zap.String("name", p.name))
		}
	}
	return nil
}

// assignPermissions назначает разрешения для ролей
func (i *Initializer) assignPermissions(ctx context.Context) error {
	assignments := []struct {
		relationType string
		permission   string
	}{
		// Владелец
		{models.ClientRelationType, models.TransactionCreate},
		{models.ClientRelationType, models.TransactionRead},
		{models.ClientRelationType, models.TransactionReadAll},
		{models.ClientRelationType, models.TransactionCancel},
		{models.ClientRelationType, models.TransactionConvert},
		{models.ClientRelationType, models.UserRead},
		{models.ClientRelationType, models.UserUpdate},
		{models.ClientRelationType, models.AccountCreate},
		{models.ClientRelationType, models.AccountRead},
		{models.ClientRelationType, models.CurrencyRead},
		{models.ClientRelationType, models.SessionTerminate},

		// Администратор
		{models.AdminRelationType, models.TransactionRead},
		{models.AdminRelationType, models.TransactionReadAll},
		{models.AdminRelationType, models.TransactionCancel},
		{models.AdminRelationType, models.TransactionUpdateStatus},
		{models.AdminRelationType, models.AccountCreate},
		{models.AdminRelationType, models.AccountRead},
		{models.AdminRelationType, models.UserRead},
		{models.AdminRelationType, models.UserReadAll},
		{models.AdminRelationType, models.UserUpdate},
		{models.AdminRelationType, models.CurrencyRead},
		{models.AdminRelationType, models.CurrencyUpdate},
		{models.AdminRelationType, models.PermissionManage},
		{models.AdminRelationType, models.PermissionRead},
		{models.AdminRelationType, models.RelationManage},
		{models.AdminRelationType, models.RelationRead},
		{models.AdminRelationType, models.SessionTerminate},
		{models.AdminRelationType, models.SessionManage},

		// Поддержка
		{models.SupportRelationType, models.TransactionRead},
		{models.SupportRelationType, models.TransactionReadAll},
		{models.SupportRelationType, models.UserRead},
		{models.SupportRelationType, models.UserReadAll},
		{models.SupportRelationType, models.AccountRead},
		{models.SupportRelationType, models.CurrencyRead},
		{models.SupportRelationType, models.SessionTerminate},
		{models.SupportRelationType, models.RelationRead},
		{models.SupportRelationType, models.PermissionRead},
	}

	for _, a := range assignments {
		perm, err := i.roleManage.GetPermissionByName(ctx, a.permission)
		if err != nil {
			return fmt.Errorf("failed to get permission %s: %w", a.permission, err)
		}

		if err := i.roleManage.AssignPermission(ctx, perm.ID, a.relationType); err != nil {
			if !errors.Is(err, billingerr.ErrAlreadyExists) {
				return fmt.Errorf("failed to assign permission %s to %s: %w",
					a.permission, a.relationType, err)
			}
			i.logger.Debug("Permission already assigned",
				zap.String("permission", a.permission),
				zap.String("relation", a.relationType))
		}
	}
	return nil
}

// createEntitys создает все необходимые разрешения
func (i *Initializer) createEntitysAndRelations(ctx context.Context) error {
	financeSystemEntityId, err := i.roleManage.CreateEntity(ctx, models.FinanceSystemEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	transactionEntityId, err := i.roleManage.CreateEntity(ctx, models.TransactionEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	accountEntityId, err := i.roleManage.CreateEntity(ctx, models.AccountEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	currencyEntityId, err := i.roleManage.CreateEntity(ctx, models.CurrencyEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	userEntityId, err := i.roleManage.CreateEntity(ctx, models.UserEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	relationEntytyId, err := i.roleManage.CreateEntity(ctx, models.RelationEntyty)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	permissionEntityId, err := i.roleManage.CreateEntity(ctx, models.PermissionEntity)
	if err != nil {
		return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, transactionEntityId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, accountEntityId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, currencyEntityId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, userEntityId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, relationEntytyId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	err = i.roleManage.CreateRelation(ctx, financeSystemEntityId, permissionEntityId, models.SystemRelationType)
	if err != nil {
		return fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}

	return nil
}

// createSystemUsers создает системные аккаунты
func (i *Initializer) createSystemUsers(ctx context.Context, systemUsers []config.User) error {
	for _, user := range systemUsers {
		userID, err := i.authUserManage.Register(ctx, user.FullName, user.Email, user.Password)
		if err != nil {
			return fmt.Errorf("failed to register %s user: %v", user.Role, err)
		}

		financeSystemEntityId, err := i.roleManage.GetEntityID(ctx, models.FinanceSystemEntity)
		if err != nil {
			return fmt.Errorf("failed to get entity in ReBAC: %v", err)
		}

		userEntityId, err := i.roleManage.GetEntityID(ctx, models.UserEntity)
		if err != nil {
			return fmt.Errorf("failed to get entity in ReBAC: %v", err)
		}

		err = i.roleManage.CreateEntityWithID(ctx, userID, models.UserEntity)
		if err != nil {
			return fmt.Errorf("failed to create user entity in ReBAC: %v", err)
		}

		var relationType string
		switch user.Role {
		case "admin":
			relationType = models.AdminRelationType
		case "support":
			relationType = models.SupportRelationType
		default:
			return fmt.Errorf("unknown user role: %s", user.Role)
		}

		err = i.roleManage.CreateRelation(ctx, userEntityId, userID, relationType)
		if err != nil {
			return fmt.Errorf("failed to create relation in ReBAC: %v", err)
		}
		err = i.roleManage.CreateRelation(ctx, userID, userID, relationType)
		if err != nil {
			return fmt.Errorf("failed to create relation in ReBAC: %v", err)
		}
		err = i.roleManage.CreateRelation(ctx, userID, financeSystemEntityId, relationType)
		if err != nil {
			return fmt.Errorf("failed to create relation in ReBAC: %v", err)
		}
	}
	return nil
}

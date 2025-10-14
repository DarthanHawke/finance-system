package role

import (
	"client-service/internal/clients/grpc/interceptor"
	"client-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	blnggrpc "github.com/DarthanHawke/protos-payment-system/gen/go/billing"
	"github.com/google/uuid"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RoleClient struct {
	roleApi blnggrpc.RoleServiceClient
	logger  *zap.Logger
}

func NewRoleClient(
	ctx context.Context,
	logger *zap.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
	tlsConfig *tls.Config,
) (*RoleClient, error) {
	const op = "clients.grpc.billing.role.NewRoleClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "billing-service",
		Certificates: tlsConfig.Certificates,
		RootCAs:      tlsConfig.ClientCAs,
	})

	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptor.InterceptorLogger(logger), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RoleClient{
		roleApi: blnggrpc.NewRoleServiceClient(cc),
		logger:  logger,
	}, nil
}

// CreateRelation создает отношение между сущностями
func (c *RoleClient) CreateRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.billing.role.CreateRelation"

	_, err := c.roleApi.CreateRelation(ctx, &blnggrpc.CreateRelationRequest{
		SourceId:     &blnggrpc.UUID{Value: sourceID.String()},
		TargetId:     &blnggrpc.UUID{Value: targetID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteRelation удаляет отношение между сущностями
func (c *RoleClient) DeleteRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.billing.role.DeleteRelation"

	_, err := c.roleApi.DeleteRelation(ctx, &blnggrpc.DeleteRelationRequest{
		SourceId:     &blnggrpc.UUID{Value: sourceID.String()},
		TargetId:     &blnggrpc.UUID{Value: targetID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// AssignPermission назначает разрешение для типа отношения
func (c *RoleClient) AssignPermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.billing.role.AssignPermission"

	_, err := c.roleApi.AssignPermission(ctx, &blnggrpc.AssignPermissionRequest{
		PermissionId: &blnggrpc.UUID{Value: permissionID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// RevokePermission отзывает разрешение у типа отношения
func (c *RoleClient) RevokePermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.billing.role.RevokePermission"

	_, err := c.roleApi.RevokePermission(ctx, &blnggrpc.RevokePermissionRequest{
		PermissionId: &blnggrpc.UUID{Value: permissionID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CheckPermission проверяет наличие разрешения у субъекта для объекта
func (c *RoleClient) CheckPermission(
	ctx context.Context,
	subjectID, objectID uuid.UUID,
	permissionName string,
) (bool, error) {
	const op = "clients.grpc.billing.role.CheckPermission"

	response, err := c.roleApi.CheckPermission(ctx, &blnggrpc.CheckPermissionRequest{
		SubjectId: &blnggrpc.UUID{Value: subjectID.String()},
		ObjectId:  &blnggrpc.UUID{Value: objectID.String()},
		Name:      permissionName,
	})
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return response.GetHasPermission(), nil
}

// GetAllPermissions возвращает все доступные разрешения
func (c *RoleClient) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	const op = "clients.grpc.billing.role.GetAllPermissions"

	response, err := c.roleApi.GetAllPermissions(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoPermissions := response.GetPermissions()
	permissions := make([]models.Permission, 0, len(protoPermissions))
	for _, permission := range protoPermissions {
		permissionID, err := uuid.Parse(permission.GetId().GetValue())
		if err != nil {
			continue
		}
		permissions = append(permissions, models.Permission{
			ID:          permissionID,
			Name:        permission.GetName(),
			Description: permission.GetDescription(),
		})
	}

	return permissions, nil
}

// GetUserRelations возвращает все отношения пользователя
func (c *RoleClient) GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error) {
	const op = "clients.grpc.billing.role.GetUserRelations"

	response, err := c.roleApi.GetUserRelations(ctx, &blnggrpc.GetUserRelationsRequest{
		UserId: &blnggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoRelations := response.GetRelations()
	relations := make([]models.Relation, 0, len(protoRelations))
	for _, relation := range protoRelations {
		sourceID, err := uuid.Parse(relation.GetSourceId().GetValue())
		if err != nil {
			continue
		}
		targetID, err := uuid.Parse(relation.GetTargetId().GetValue())
		if err != nil {
			continue
		}
		relations = append(relations, models.Relation{
			SourceID:     sourceID,
			TargetID:     targetID,
			RelationType: relation.GetRelationType(),
		})
	}

	return relations, nil
}

// GetUserPermissions возвращает все разрешения пользователя
func (c *RoleClient) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	const op = "clients.grpc.billing.role.GetUserPermissions"

	response, err := c.roleApi.GetUserPermissions(ctx, &blnggrpc.GetUserPermissionsRequest{
		UserId: &blnggrpc.UUID{Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoPermissions := response.GetPermissions()
	permissions := make([]models.Permission, 0, len(protoPermissions))
	for _, permission := range protoPermissions {
		permissionID, err := uuid.Parse(permission.GetId().GetValue())
		if err != nil {
			continue
		}
		permissions = append(permissions, models.Permission{
			ID:          permissionID,
			Name:        permission.GetName(),
			Description: permission.GetDescription(),
		})
	}

	return permissions, nil
}

// GetAllEntities возвращает все сущности
func (c *RoleClient) GetAllEntities(ctx context.Context) ([]models.Entity, error) {
	const op = "clients.grpc.billing.role.GetAllEntities"

	response, err := c.roleApi.GetAllEntities(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoEntities := response.GetEntities()
	entities := make([]models.Entity, 0, len(protoEntities))
	for _, protoEntity := range protoEntities {
		id, err := uuid.Parse(protoEntity.GetId().GetValue())
		if err != nil {
			continue
		}
		entities = append(entities, models.Entity{
			ID:   id,
			Type: protoEntity.GetType(),
		})
	}

	return entities, nil
}

// GetAllEntities роль пользователя
func (c *RoleClient) GetUserRole(ctx context.Context) (string, error) {
	const op = "clients.grpc.billing.role.GetUserRole"

	response, err := c.roleApi.GetUserRole(ctx, &emptypb.Empty{})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return response.Role, nil
}

// GetPermissionsForRelationType возвращает разрешения для типа отношения
func (c *RoleClient) GetPermissionsForRelationType(
	ctx context.Context,
	relationType string,
) ([]models.Permission, error) {
	const op = "clients.grpc.billing.role.GetPermissionsForRelationType"

	response, err := c.roleApi.GetPermissionsForRelationType(ctx, &blnggrpc.GetPermissionsForRelationTypeRequest{
		RelationType: relationType,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoPermissions := response.GetPermissions()
	permissions := make([]models.Permission, 0, len(protoPermissions))
	for _, permission := range protoPermissions {
		permissionID, err := uuid.Parse(permission.GetId().GetValue())
		if err != nil {
			continue
		}
		permissions = append(permissions, models.Permission{
			ID:          permissionID,
			Name:        permission.GetName(),
			Description: permission.GetDescription(),
		})
	}

	return permissions, nil
}

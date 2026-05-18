package role

/* legacy code
import (
	"billing-service/internal/clients/grpc/interceptor"
	"billing-service/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	ssogrpc "github.com/DarthanHawke/protos-finance-system/gen/go/sso"
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
	roleApi ssogrpc.RoleServiceClient
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
	const op = "clients.sso.role.NewRoleClient"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   "sso-service",
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
		roleApi: ssogrpc.NewRoleServiceClient(cc),
		logger:  logger,
	}, nil
}

func (c *RoleClient) CreateEntity(ctx context.Context, entityType string) (uuid.UUID, error) {
	const op = "clients.grpc.sso.role.CreateEntity"

	response, err := c.roleApi.CreateEntity(ctx, &ssogrpc.CreateEntityRequest{
		EntityType: entityType,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	idEntity, err := uuid.Parse(response.GetId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: invalid UUID format in response: %w", op, err)
	}

	return idEntity, nil
}

func (c *RoleClient) CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error {
	const op = "clients.grpc.sso.role.CreateEntityWithID"

	_, err := c.roleApi.CreateEntityWithID(ctx, &ssogrpc.CreateEntityWithIDRequest{
		Id:         &ssogrpc.UUID{Value: id.String()},
		EntityType: entityType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (c *RoleClient) DeleteEntity(ctx context.Context, id uuid.UUID) error {
	const op = "clients.grpc.sso.role.DeleteEntity"

	_, err := c.roleApi.DeleteEntity(ctx, &ssogrpc.DeleteEntityRequest{
		Id: &ssogrpc.UUID{Value: id.String()},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *RoleClient) CreateRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.sso.role.CreateRelation"

	_, err := c.roleApi.CreateRelation(ctx, &ssogrpc.CreateRelationRequest{
		SourceId:     &ssogrpc.UUID{Value: sourceID.String()},
		TargetId:     &ssogrpc.UUID{Value: targetID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *RoleClient) DeleteRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.sso.role.DeleteRelation"

	_, err := c.roleApi.DeleteRelation(ctx, &ssogrpc.DeleteRelationRequest{
		SourceId:     &ssogrpc.UUID{Value: sourceID.String()},
		TargetId:     &ssogrpc.UUID{Value: targetID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *RoleClient) AddPermission(
	ctx context.Context,
	name, description string,
) (uuid.UUID, error) {
	const op = "clients.grpc.sso.role.AddPermission"

	response, err := c.roleApi.AddPermission(ctx, &ssogrpc.AddPermissionRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	permissionID, err := uuid.Parse(response.GetPermissionId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissionID, nil
}

func (c *RoleClient) AssignPermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.sso.role.AssignPermission"

	_, err := c.roleApi.AssignPermission(ctx, &ssogrpc.AssignPermissionRequest{
		PermissionId: &ssogrpc.UUID{Value: permissionID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *RoleClient) RevokePermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "clients.grpc.sso.role.RevokePermission"

	_, err := c.roleApi.RevokePermission(ctx, &ssogrpc.RevokePermissionRequest{
		PermissionId: &ssogrpc.UUID{Value: permissionID.String()},
		RelationType: relationType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *RoleClient) CheckPermission(
	ctx context.Context,
	subjectID, objectID uuid.UUID,
	permissionName string,
) (bool, error) {
	const op = "clients.grpc.sso.role.CheckPermission"

	response, err := c.roleApi.CheckPermission(ctx, &ssogrpc.CheckPermissionRequest{
		SubjectId: &ssogrpc.UUID{Value: subjectID.String()},
		ObjectId:  &ssogrpc.UUID{Value: objectID.String()},
		Name:      permissionName,
	})
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return response.GetHasPermission(), nil
}

func (c *RoleClient) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	const op = "clients.grpc.sso.role.GetPermissionByName"

	response, err := c.roleApi.GetPermissionByName(ctx, &ssogrpc.GetPermissionByNameRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoPermission := response.GetPermission()
	permissionID, err := uuid.Parse(protoPermission.GetId().GetValue())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.Permission{
		ID:          permissionID,
		Name:        protoPermission.GetName(),
		Description: protoPermission.GetDescription(),
	}, nil
}

func (c *RoleClient) GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error) {
	const op = "clients.grpc.sso.role.GetEntityID"

	response, err := c.roleApi.GetEntityId(ctx, &ssogrpc.GetEntityIdRequest{
		EntityType: entityType,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	id, err := uuid.Parse(response.GetId().GetValue())
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: invalid UUID format in response: %w", op, err)
	}

	return id, nil
}

func (c *RoleClient) GetAllEntities(ctx context.Context) ([]models.Entity, error) {
	const op = "clients.grpc.sso.role.GetAllEntities"

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

func (c *RoleClient) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	const op = "clients.grpc.sso.role.GetAllPermissions"

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

func (c *RoleClient) GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error) {
	const op = "clients.grpc.sso.role.GetUserRelations"

	response, err := c.roleApi.GetUserRelations(ctx, &ssogrpc.GetUserRelationsRequest{
		UserId: &ssogrpc.UUID{Value: userID.String()},
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

func (c *RoleClient) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	const op = "clients.grpc.sso.role.GetUserPermissions"

	response, err := c.roleApi.GetUserPermissions(ctx, &ssogrpc.GetUserPermissionsRequest{
		UserId: &ssogrpc.UUID{Value: userID.String()},
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

func (c *RoleClient) GetPermissionsForRelationType(
	ctx context.Context,
	relationType string,
) ([]models.Permission, error) {
	const op = "clients.grpc.sso.role.GetPermissionsForRelationType"

	response, err := c.roleApi.GetPermissionsForRelationType(ctx, &ssogrpc.GetPermissionsForRelationTypeRequest{
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

func (c *RoleClient) GetEntityRelations(
	ctx context.Context,
	entityID uuid.UUID,
) ([]models.Relation, error) {
	const op = "clients.grpc.sso.role.GetEntityRelations"

	response, err := c.roleApi.GetEntityRelations(ctx, &ssogrpc.GetEntityRelationsRequest{
		EntityId: &ssogrpc.UUID{Value: entityID.String()},
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
*/

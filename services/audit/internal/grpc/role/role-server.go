package role

/* legacy code
import (
	"billing-service/internal/models"
	"context"

	blnggrpc "github.com/DarthanHawke/protos-finance-system/gen/go/billing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Role interface {
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	DeleteRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	AssignPermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	RevokePermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
	GetAllEntities(ctx context.Context) ([]models.Entity, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
	GetPermissionsForRelationType(ctx context.Context, relationType string) ([]models.Permission, error)
	GetUserRole(ctx context.Context) (string, error)
}

type RoleServerAPI struct {
	blnggrpc.UnimplementedRoleServiceServer
	role Role
}

func NewRoleServer(gRPC *grpc.Server, role Role) {
	blnggrpc.RegisterRoleServiceServer(gRPC, &RoleServerAPI{role: role})
}

func (s *RoleServerAPI) CreateRelation(
	ctx context.Context,
	req *blnggrpc.CreateRelationRequest,
) (*emptypb.Empty, error) {
	sourceID, err := uuid.Parse(req.GetSourceId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source UUID format: %v", err)
	}

	targetID, err := uuid.Parse(req.GetTargetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target UUID format: %v", err)
	}

	if req.GetRelationType() == "" {
		return nil, status.Error(codes.InvalidArgument, "relation type is required")
	}

	err = s.role.CreateRelation(ctx, sourceID, targetID, req.GetRelationType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create relation")
	}
	return &emptypb.Empty{}, nil
}

func (s *RoleServerAPI) DeleteRelation(
	ctx context.Context,
	req *blnggrpc.DeleteRelationRequest,
) (*emptypb.Empty, error) {
	sourceID, err := uuid.Parse(req.GetSourceId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source UUID format: %v", err)
	}

	targetID, err := uuid.Parse(req.GetTargetId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target UUID format: %v", err)
	}

	if req.GetRelationType() == "" {
		return nil, status.Error(codes.InvalidArgument, "relation type is required")
	}

	err = s.role.DeleteRelation(ctx, sourceID, targetID, req.GetRelationType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete relation")
	}
	return &emptypb.Empty{}, nil
}

func (s *RoleServerAPI) AssignPermission(
	ctx context.Context,
	req *blnggrpc.AssignPermissionRequest,
) (*emptypb.Empty, error) {
	permissionID, err := uuid.Parse(req.GetPermissionId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid permission UUID format: %v", err)
	}

	if req.GetRelationType() == "" {
		return nil, status.Error(codes.InvalidArgument, "relation type is required")
	}

	err = s.role.AssignPermission(ctx, permissionID, req.GetRelationType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign permission")
	}
	return &emptypb.Empty{}, nil
}

func (s *RoleServerAPI) RevokePermission(
	ctx context.Context,
	req *blnggrpc.RevokePermissionRequest,
) (*emptypb.Empty, error) {
	permissionID, err := uuid.Parse(req.GetPermissionId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid permission UUID format: %v", err)
	}

	if req.GetRelationType() == "" {
		return nil, status.Error(codes.InvalidArgument, "relation type is required")
	}

	err = s.role.RevokePermission(ctx, permissionID, req.GetRelationType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign permission")
	}
	return &emptypb.Empty{}, nil
}

func (s *RoleServerAPI) CheckPermission(
	ctx context.Context,
	req *blnggrpc.CheckPermissionRequest,
) (*blnggrpc.CheckPermissionResponse, error) {
	subjectID, err := uuid.Parse(req.GetSubjectId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid subject UUID format: %v", err)
	}

	objectID, err := uuid.Parse(req.GetObjectId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid object UUID format: %v", err)
	}

	hasPermission, err := s.role.CheckPermission(ctx, subjectID, objectID, req.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check permission")
	}

	return &blnggrpc.CheckPermissionResponse{
		HasPermission: hasPermission,
	}, nil
}

func (s *RoleServerAPI) GetAllPermissions(
	ctx context.Context,
	_ *emptypb.Empty,
) (*blnggrpc.GetAllPermissionsResponse, error) {
	permissions, err := s.role.GetAllPermissions(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get permissions")
	}

	protoPermissions := make([]*blnggrpc.Permission, 0, len(permissions))
	for _, permission := range permissions {
		protoPermissions = append(protoPermissions, &blnggrpc.Permission{
			Id:          &blnggrpc.UUID{Value: permission.ID.String()},
			Name:        permission.Name,
			Description: permission.Description,
		})
	}

	return &blnggrpc.GetAllPermissionsResponse{Permissions: protoPermissions}, nil
}

func (s *RoleServerAPI) GetUserRelations(
	ctx context.Context,
	req *blnggrpc.GetUserRelationsRequest,
) (*blnggrpc.GetUserRelationsResponse, error) {
	userID, err := uuid.Parse(req.GetUserId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID format: %v", err)
	}

	relations, err := s.role.GetUserRelations(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user relations")
	}

	protoRelations := make([]*blnggrpc.Relation, 0, len(relations))
	for _, relation := range relations {
		protoRelations = append(protoRelations, &blnggrpc.Relation{
			SourceId:     &blnggrpc.UUID{Value: relation.SourceID.String()},
			TargetId:     &blnggrpc.UUID{Value: relation.TargetID.String()},
			RelationType: relation.RelationType,
		})
	}

	return &blnggrpc.GetUserRelationsResponse{Relations: protoRelations}, nil
}

func (s *RoleServerAPI) GetUserPermissions(
	ctx context.Context,
	req *blnggrpc.GetUserPermissionsRequest,
) (*blnggrpc.GetUserPermissionsResponse, error) {
	userID, err := uuid.Parse(req.GetUserId().GetValue())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID format: %v", err)
	}

	permissions, err := s.role.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user permissions")
	}

	protoPermissions := make([]*blnggrpc.Permission, 0, len(permissions))
	for _, permission := range permissions {
		protoPermissions = append(protoPermissions, &blnggrpc.Permission{
			Id:          &blnggrpc.UUID{Value: permission.ID.String()},
			Name:        permission.Name,
			Description: permission.Description,
		})
	}

	return &blnggrpc.GetUserPermissionsResponse{Permissions: protoPermissions}, nil
}

func (s *RoleServerAPI) GetAllEntities(
	ctx context.Context,
	_ *emptypb.Empty,
) (*blnggrpc.GetAllEntitiesResponse, error) {
	entities, err := s.role.GetAllEntities(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get entities")
	}

	protoEntities := make([]*blnggrpc.Entity, 0, len(entities))
	for _, entity := range entities {
		protoEntities = append(protoEntities, &blnggrpc.Entity{
			Id:   &blnggrpc.UUID{Value: entity.ID.String()},
			Type: entity.Type,
		})
	}

	return &blnggrpc.GetAllEntitiesResponse{Entities: protoEntities}, nil
}

func (s *RoleServerAPI) GetUserRole(
	ctx context.Context,
	_ *emptypb.Empty,
) (*blnggrpc.GetUserRoleResponse, error) {
	role, err := s.role.GetUserRole(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get role")
	}

	return &blnggrpc.GetUserRoleResponse{
		Role: role,
	}, nil
}

func (s *RoleServerAPI) GetPermissionsForRelationType(
	ctx context.Context,
	req *blnggrpc.GetPermissionsForRelationTypeRequest,
) (*blnggrpc.GetPermissionsForRelationTypeResponse, error) {
	if req.GetRelationType() == "" {
		return nil, status.Error(codes.InvalidArgument, "relation type is required")
	}

	permissions, err := s.role.GetPermissionsForRelationType(ctx, req.GetRelationType())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get permissions for relation type")
	}

	protoPermissions := make([]*blnggrpc.Permission, 0, len(permissions))
	for _, permission := range permissions {
		protoPermissions = append(protoPermissions, &blnggrpc.Permission{
			Id:          &blnggrpc.UUID{Value: permission.ID.String()},
			Name:        permission.Name,
			Description: permission.Description,
		})
	}

	return &blnggrpc.GetPermissionsForRelationTypeResponse{Permissions: protoPermissions}, nil
}
*/

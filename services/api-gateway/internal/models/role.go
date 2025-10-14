package models

import (
	"time"

	"github.com/google/uuid"
)

// Entity представляет сущность в системе ReBAC
type Entity struct {
	ID        uuid.UUID `db:"id"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
}

// Relation представляет отношение между сущностями
type Relation struct {
	ID           uuid.UUID `db:"id"`
	SourceID     uuid.UUID `db:"source_id"`
	TargetID     uuid.UUID `db:"target_id"`
	RelationType string    `db:"relation_type"`
	CreatedAt    time.Time `db:"created_at"`
}

// Permission представляет разрешение в системе
type Permission struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// PermissionAssignment представляет назначение разрешения для типа отношения
type PermissionAssignment struct {
	RelationType string    `db:"relation_type"`
	PermissionID uuid.UUID `db:"permission_id"`
	CreatedAt    time.Time `db:"created_at"`
}

type CreateRelationRequest struct {
	SourceID     uuid.UUID `json:"sourceID"`
	TargetID     uuid.UUID `json:"targetID"`
	RelationType string    `json:"relationType"`
}

type DeleteRelationRequest struct {
	SourceID     uuid.UUID `json:"sourceID"`
	TargetID     uuid.UUID `json:"targetID"`
	RelationType string    `json:"relationType"`
}

type AssignPermissionRequest struct {
	PermissionID uuid.UUID `json:"permissionID"`
	RelationType string    `json:"relationType"`
}

type RevokePermissionRequest struct {
	PermissionID uuid.UUID `json:"permissionID"`
	RelationType string    `json:"relationType"`
}

type PermissionCheckResponse struct {
	HasPermission bool `json:"hasPermission"`
}

// CheckPermissionRequest - запрос на проверку разрешения
type CheckPermissionRequest struct {
	SubjectID      uuid.UUID `json:"subject_id" validate:"required"`
	ObjectID       uuid.UUID `json:"object_id" validate:"required"`
	PermissionName string    `json:"permission_name" validate:"required"`
}

// CheckPermissionResponse - ответ на проверку разрешения
type CheckPermissionResponse struct {
	HasPermission bool `json:"has_permission"`
}

type UserRoleResponse struct {
	Role string `json:"role"`
}

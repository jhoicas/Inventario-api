package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// RoleRepository expone el catálogo de roles del sistema.
type RoleRepository interface {
	GetByID(id string) (*entity.Role, error)
	GetByKey(key string) (*entity.Role, error)
	List(limit, offset int) ([]*entity.Role, error)
}

// RBACRepository concentra el acceso a módulos, pantallas y permisos.
type RBACRepository interface {
	ListModulesWithScreens() ([]*entity.Module, error)
	GetMenuByRoleID(roleID string) ([]*entity.Module, error)
	CanAccess(roleID, apiEndpoint string) (bool, error)
	GetScreenByID(ctx context.Context, id string) (*entity.Screen, error)
	GetScreenByEndpoint(ctx context.Context, apiEndpoint string) (*entity.Screen, error)
	ReplaceRoleScreens(roleID string, screenIDs []string) error
}

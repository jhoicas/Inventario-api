package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// RBACUseCase encapsula la lógica de roles, módulos y pantallas.
type RBACUseCase struct {
	roleRepo repository.RoleRepository
	rbacRepo repository.RBACRepository
}

// NewRBACUseCase construye el caso de uso RBAC.
func NewRBACUseCase(roleRepo repository.RoleRepository, rbacRepo repository.RBACRepository) *RBACUseCase {
	return &RBACUseCase{roleRepo: roleRepo, rbacRepo: rbacRepo}
}

// ListRoles devuelve el catálogo de roles activos.
func (uc *RBACUseCase) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	if uc.roleRepo == nil {
		return nil, fmt.Errorf("role repository no configurado")
	}
	roles, err := uc.roleRepo.List(1000, 0)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RoleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, dto.RoleResponse{
			ID:        role.ID,
			Key:       role.Key,
			Name:      role.Name,
			CreatedAt: role.CreatedAt,
			UpdatedAt: role.UpdatedAt,
		})
	}
	return out, nil
}

// ListCatalog devuelve todos los módulos y pantallas disponibles.
func (uc *RBACUseCase) ListCatalog(ctx context.Context) (*dto.RBACCatalogResponse, error) {
	if uc.rbacRepo == nil {
		return nil, fmt.Errorf("rbac repository no configurado")
	}
	modules, err := uc.rbacRepo.ListModulesWithScreens()
	if err != nil {
		return nil, err
	}
	return &dto.RBACCatalogResponse{Modules: mapModules(modules)}, nil
}

// GetMenuByRoleRef devuelve el menú jerárquico permitido para el rol indicado.
func (uc *RBACUseCase) GetMenuByRoleRef(ctx context.Context, roleRef string) (*dto.RBACMenuResponse, error) {
	if uc.rbacRepo == nil {
		return nil, fmt.Errorf("rbac repository no configurado")
	}
	role, err := uc.resolveRole(ctx, roleRef)
	if err != nil {
		return nil, err
	}
	modules, err := uc.rbacRepo.GetMenuByRoleID(role.ID)
	if err != nil {
		return nil, err
	}
	return &dto.RBACMenuResponse{
		RoleID:   role.ID,
		RoleKey:  role.Key,
		RoleName: role.Name,
		Modules:  mapModules(modules),
	}, nil
}

// GetCurrentMenu devuelve el menú del rol activo almacenado en el JWT.
func (uc *RBACUseCase) GetCurrentMenu(ctx context.Context, roleRef string) (*dto.RBACMenuResponse, error) {
	return uc.GetMenuByRoleRef(ctx, roleRef)
}

// UpdateRoleScreens reemplaza las pantallas permitidas para un rol.
func (uc *RBACUseCase) UpdateRoleScreens(ctx context.Context, roleRef string, in dto.UpdateRoleScreensRequest) (*dto.RBACMenuResponse, error) {
	if uc.rbacRepo == nil {
		return nil, fmt.Errorf("rbac repository no configurado")
	}
	role, err := uc.resolveRole(ctx, roleRef)
	if err != nil {
		return nil, err
	}
	if err := uc.rbacRepo.ReplaceRoleScreens(role.ID, in.ScreenIDs); err != nil {
		return nil, err
	}
	return uc.GetMenuByRoleRef(ctx, role.ID)
}

// CanAccess valida si el rol indicado puede acceder a una ruta de API.
func (uc *RBACUseCase) CanAccess(ctx context.Context, roleRef, apiEndpoint string) (bool, error) {
	if uc.rbacRepo == nil {
		return false, fmt.Errorf("rbac repository no configurado")
	}
	role, err := uc.resolveRole(ctx, roleRef)
	if err != nil {
		return false, err
	}
	return uc.rbacRepo.CanAccess(role.ID, apiEndpoint)
}

// GetScreenByID devuelve una pantalla por su ID.
func (uc *RBACUseCase) GetScreenByID(ctx context.Context, id string) (*entity.Screen, error) {
	if uc.rbacRepo == nil {
		return nil, fmt.Errorf("rbac repository no configurado")
	}
	return uc.rbacRepo.GetScreenByID(ctx, id)
}

// GetScreenByEndpoint devuelve una pantalla por su endpoint API normalizado.
func (uc *RBACUseCase) GetScreenByEndpoint(ctx context.Context, apiEndpoint string) (*entity.Screen, error) {
	if uc.rbacRepo == nil {
		return nil, fmt.Errorf("rbac repository no configurado")
	}
	return uc.rbacRepo.GetScreenByEndpoint(ctx, apiEndpoint)
}

// ResolveRoleID normaliza un roleRef y devuelve su ID real.
func (uc *RBACUseCase) ResolveRoleID(ctx context.Context, roleRef string) (string, error) {
	role, err := uc.resolveRole(ctx, roleRef)
	if err != nil {
		return "", err
	}
	return role.ID, nil
}

func (uc *RBACUseCase) resolveRole(ctx context.Context, roleRef string) (*entity.Role, error) {
	if uc.roleRepo == nil {
		return nil, fmt.Errorf("role repository no configurado")
	}
	roleRef = strings.TrimSpace(roleRef)
	if roleRef == "" {
		return nil, domain.ErrInvalidInput
	}

	if _, err := uuid.Parse(roleRef); err == nil {
		role, err := uc.roleRepo.GetByID(roleRef)
		if err != nil {
			return nil, err
		}
		if role != nil {
			return role, nil
		}
	}

	role, err := uc.roleRepo.GetByKey(strings.ToLower(roleRef))
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.ErrNotFound
	}
	return role, nil
}

func mapModules(modules []*entity.Module) []dto.ModuleResponse {
	out := make([]dto.ModuleResponse, 0, len(modules))
	for _, module := range modules {
		if module == nil {
			continue
		}
		m := dto.ModuleResponse{
			ID:    module.ID,
			Key:   module.Key,
			Name:  module.Name,
			Icon:  module.Icon,
			Order: module.Order,
		}
		for _, screen := range module.Screens {
			m.Screens = append(m.Screens, dto.ScreenResponse{
				ID:                   screen.ID,
				Key:                  screen.Key,
				Name:                 screen.Name,
				ModuleKey:            screen.ModuleKey,
				ModuleName:           screen.ModuleName,
				ModuleKeySnapshot:    screen.ModuleKeySnapshot,
				FrontendRoute:        screen.FrontendRoute,
				ApiEndpoint:          screen.ApiEndpoint,
				Order:                screen.Order,
				ModuleClassification: deriveModuleClassification(module.Key),
			})
		}
		out = append(out, m)
	}
	return out
}

func deriveModuleClassification(moduleKey string) string {
	moduleKey = strings.TrimSpace(moduleKey)
	if moduleKey == "" {
		return ""
	}
	parts := strings.SplitN(moduleKey, ".", 2)
	return parts[0]
}

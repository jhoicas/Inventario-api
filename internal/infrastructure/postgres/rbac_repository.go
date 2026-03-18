package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.RoleRepository = (*RBACRepo)(nil)
var _ repository.RBACRepository = (*RBACRepo)(nil)

// RBACRepo concentra el catálogo de roles, módulos, pantallas y permisos.
type RBACRepo struct {
	pool *pgxpool.Pool
}

// NewRBACRepository construye el repositorio RBAC sobre PostgreSQL.
func NewRBACRepository(pool *pgxpool.Pool) *RBACRepo {
	return &RBACRepo{pool: pool}
}

// GetByID obtiene un rol por ID.
func (r *RBACRepo) GetByID(id string) (*entity.Role, error) {
	var role entity.Role
	err := r.pool.QueryRow(context.Background(), `
		SELECT id, key, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM roles
		WHERE id = $1`,
		id,
	).Scan(
		&role.ID,
		&role.Key,
		&role.Name,
		&role.Description,
		&role.IsActive,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role by id: %w", err)
	}
	return &role, nil
}

// GetByKey obtiene un rol por su key.
func (r *RBACRepo) GetByKey(key string) (*entity.Role, error) {
	var role entity.Role
	err := r.pool.QueryRow(context.Background(), `
		SELECT id, key, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM roles
		WHERE key = $1`,
		strings.ToLower(strings.TrimSpace(key)),
	).Scan(
		&role.ID,
		&role.Key,
		&role.Name,
		&role.Description,
		&role.IsActive,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role by key: %w", err)
	}
	return &role, nil
}

// List lista roles activos con paginación.
func (r *RBACRepo) List(limit, offset int) ([]*entity.Role, error) {
	rows, err := r.pool.Query(context.Background(), `
		SELECT id, key, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM roles
		WHERE is_active = true
		ORDER BY key
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var out []*entity.Role
	for rows.Next() {
		var role entity.Role
		if err := rows.Scan(&role.ID, &role.Key, &role.Name, &role.Description, &role.IsActive, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

// ListModulesWithScreens devuelve todos los módulos con sus pantallas activas.
func (r *RBACRepo) ListModulesWithScreens() ([]*entity.Module, error) {
	return r.loadModules(context.Background(), "")
}

// GetMenuByRoleID devuelve los módulos/pantallas permitidos para un rol.
func (r *RBACRepo) GetMenuByRoleID(roleID string) ([]*entity.Module, error) {
	return r.loadModules(context.Background(), roleID)
}

// CanAccess informa si el rol puede acceder al endpoint indicado.
func (r *RBACRepo) CanAccess(roleID, apiEndpoint string) (bool, error) {
	apiEndpoint = normalizeEndpoint(apiEndpoint)
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM role_screens rs
			JOIN screens s ON s.id = rs.screen_id
			JOIN modules m ON m.id = s.module_id
			WHERE rs.role_id = $1
			  AND s.is_active = true
			  AND m.is_active = true
			  AND (
			       $2 = s.api_endpoint
			       OR $2 LIKE s.api_endpoint || '/%'
			  )
		)`
	var allowed bool
	if err := r.pool.QueryRow(context.Background(), query, roleID, apiEndpoint).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check access: %w", err)
	}
	return allowed, nil
}

// ReplaceRoleScreens reemplaza todas las pantallas de un rol.
func (r *RBACRepo) ReplaceRoleScreens(roleID string, screenIDs []string) error {
	tx, err := r.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin rbac transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if _, err := tx.Exec(context.Background(), `DELETE FROM role_screens WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("clear role screens: %w", err)
	}

	if len(screenIDs) > 0 {
		for _, screenID := range screenIDs {
			if _, err := tx.Exec(context.Background(), `
				INSERT INTO role_screens (role_id, screen_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING`,
				roleID, screenID,
			); err != nil {
				return fmt.Errorf("replace role screens: %w", err)
			}
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit rbac transaction: %w", err)
	}
	return nil
}

func (r *RBACRepo) loadModules(ctx context.Context, roleID string) ([]*entity.Module, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if roleID == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT m.id, m.key, m.name, m.icon, m."order", m.is_active,
			       s.id, s.module_id, s.key, s.name, s.frontend_route, s.api_endpoint, s."order", s.is_active,
			       m.created_at, m.updated_at, s.created_at, s.updated_at
			FROM modules m
			JOIN screens s ON s.module_id = m.id AND s.is_active = true
			WHERE m.is_active = true
			ORDER BY m."order", s."order"`)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT m.id, m.key, m.name, m.icon, m."order", m.is_active,
			       s.id, s.module_id, s.key, s.name, s.frontend_route, s.api_endpoint, s."order", s.is_active,
			       m.created_at, m.updated_at, s.created_at, s.updated_at
			FROM modules m
			JOIN screens s ON s.module_id = m.id AND s.is_active = true
			JOIN role_screens rs ON rs.screen_id = s.id
			WHERE m.is_active = true
			  AND rs.role_id = $1
			ORDER BY m."order", s."order"`,
			roleID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("load modules with screens: %w", err)
	}
	defer rows.Close()

	modulesByID := map[string]*entity.Module{}
	ordered := make([]*entity.Module, 0)

	for rows.Next() {
		var (
			module     entity.Module
			screen     entity.Screen
		)

		if err := rows.Scan(
			&module.ID,
			&module.Key,
			&module.Name,
			&module.Icon,
			&module.Order,
			&module.IsActive,
			&screen.ID,
			&screen.ModuleID,
			&screen.Key,
			&screen.Name,
			&screen.FrontendRoute,
			&screen.ApiEndpoint,
			&screen.Order,
			&screen.IsActive,
			&module.CreatedAt,
			&module.UpdatedAt,
			&screen.CreatedAt,
			&screen.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rbac module row: %w", err)
		}

		if existing, ok := modulesByID[module.ID]; ok {
			if screen.ID != "" {
				existing.Screens = append(existing.Screens, screen)
			}
			continue
		}

		module.Screens = []entity.Screen{}
		module.Screens = append(module.Screens, screen)
		modulesByID[module.ID] = &module
		ordered = append(ordered, &module)
	}

	return ordered, rows.Err()
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return endpoint
	}
	switch endpoint {
	case "/api/dian/settings", "/api/dian/configuration":
		endpoint = "/api/settings/dian"
	}
	if strings.HasSuffix(endpoint, "/") && endpoint != "/" {
		endpoint = strings.TrimRight(endpoint, "/")
	}
	return endpoint
}


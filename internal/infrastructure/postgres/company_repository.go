package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// Asegura que CompanyRepo implementa repository.CompanyRepository.
var _ repository.CompanyRepository = (*CompanyRepo)(nil)

// CompanyRepo implementación del puerto CompanyRepository sobre PostgreSQL.
type CompanyRepo struct {
	pool *pgxpool.Pool
}

// NewCompanyRepository construye el adaptador de persistencia para empresas.
func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepo {
	return &CompanyRepo{pool: pool}
}

// Create persiste una nueva empresa.
func (r *CompanyRepo) Create(company *entity.Company) error {
	environment := company.Environment
	if environment == "" {
		environment = "habilitacion"
	}
	query := `
		INSERT INTO companies (id, name, nit, address, phone, email, status, environment, cert_hab, cert_prod, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.pool.Exec(context.Background(), query,
		company.ID, company.Name, company.NIT, company.Address,
		company.Phone, company.Email, company.Status,
		environment, nullIfEmpty(company.CertHab), nullIfEmpty(company.CertProd),
		company.CreatedAt, company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	return nil
}

// GetByID obtiene una empresa por ID.
func (r *CompanyRepo) GetByID(id string) (*entity.Company, error) {
	query := `
		SELECT id, name, nit, address, phone, email, status,
		       COALESCE(environment, 'habilitacion'), COALESCE(cert_hab, ''), COALESCE(cert_prod, ''),
		       created_at, updated_at
		FROM companies WHERE id = $1`
	var c entity.Company
	err := r.pool.QueryRow(context.Background(), query, id).Scan(
		&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status,
		&c.Environment, &c.CertHab, &c.CertProd,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get company: %w", err)
	}
	return &c, nil
}

// GetByNIT obtiene una empresa por NIT.
func (r *CompanyRepo) GetByNIT(nit string) (*entity.Company, error) {
	query := `
		SELECT id, name, nit, address, phone, email, status,
		       COALESCE(environment, 'habilitacion'), COALESCE(cert_hab, ''), COALESCE(cert_prod, ''),
		       created_at, updated_at
		FROM companies WHERE nit = $1`
	var c entity.Company
	err := r.pool.QueryRow(context.Background(), query, nit).Scan(
		&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status,
		&c.Environment, &c.CertHab, &c.CertProd,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get company by NIT: %w", err)
	}
	return &c, nil
}

// Update actualiza una empresa existente.
func (r *CompanyRepo) Update(company *entity.Company) error {
	environment := company.Environment
	if environment == "" {
		environment = "habilitacion"
	}
	query := `
		UPDATE companies
		SET name = $2, nit = $3, address = $4, phone = $5, email = $6, status = $7,
		    environment = $8, cert_hab = $9, cert_prod = $10, updated_at = $11
		WHERE id = $1`
	cmd, err := r.pool.Exec(context.Background(), query,
		company.ID, company.Name, company.NIT, company.Address,
		company.Phone, company.Email, company.Status,
		environment, nullIfEmpty(company.CertHab), nullIfEmpty(company.CertProd), company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil // o devolver domain.ErrNotFound si se prefiere
	}
	return nil
}

// List devuelve empresas con paginación.
func (r *CompanyRepo) List(limit, offset int) ([]*entity.Company, error) {
	query := `
		SELECT id, name, nit, address, phone, email, status,
		       COALESCE(environment, 'habilitacion'), COALESCE(cert_hab, ''), COALESCE(cert_prod, ''),
		       created_at, updated_at
		FROM companies ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var list []*entity.Company
	for rows.Next() {
		var c entity.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status, &c.Environment, &c.CertHab, &c.CertProd, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

// ListForAdmin devuelve empresas excluyendo la cuenta técnica del superadmin.
func (r *CompanyRepo) ListForAdmin(limit, offset int) ([]*entity.Company, error) {
	query := `
		SELECT id, name, nit, address, phone, email, status,
		       COALESCE(environment, 'habilitacion'), COALESCE(cert_hab, ''), COALESCE(cert_prod, ''),
		       created_at, updated_at
		FROM companies
		WHERE lower(email) <> lower('it@ludoia.com')
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list companies for admin: %w", err)
	}
	defer rows.Close()

	var list []*entity.Company
	for rows.Next() {
		var c entity.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status, &c.Environment, &c.CertHab, &c.CertProd, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan company for admin: %w", err)
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

// Delete elimina una empresa por ID.
func (r *CompanyRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM companies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}
	return nil
}

// HasActiveModule informa si la empresa tiene el módulo activo y sin vencer.
// Consulta directamente company_modules para una respuesta O(1) vía índice.
func (r *CompanyRepo) HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM company_modules
			 WHERE company_id  = $1
			   AND module_name = $2
			   AND is_active   = true
			   AND (expires_at IS NULL OR expires_at > now())
		)`
	var active bool
	if err := r.pool.QueryRow(ctx, query, companyID, moduleName).Scan(&active); err != nil {
		return false, fmt.Errorf("check module %s: %w", moduleName, err)
	}
	return active, nil
}

// ListModules devuelve los módulos contratados por la empresa.
func (r *CompanyRepo) ListModules(ctx context.Context, companyID string) ([]*entity.CompanyModule, error) {
	const query = `
		SELECT id, company_id, module_name, is_active, activated_at, expires_at, created_at, updated_at
		FROM company_modules
		WHERE company_id = $1
		ORDER BY module_name`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("list company modules: %w", err)
	}
	defer rows.Close()

	var list []*entity.CompanyModule
	for rows.Next() {
		var m entity.CompanyModule
		if err := rows.Scan(&m.ID, &m.CompanyID, &m.ModuleName, &m.IsActive, &m.ActivatedAt, &m.ExpiresAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan company module: %w", err)
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

// GetModule devuelve un módulo específico para una empresa.
func (r *CompanyRepo) GetModule(ctx context.Context, companyID, moduleName string) (*entity.CompanyModule, error) {
	const query = `
		SELECT id, company_id, module_name, is_active, activated_at, expires_at, created_at, updated_at
		FROM company_modules
		WHERE company_id = $1 AND module_name = $2
		LIMIT 1`
	var m entity.CompanyModule
	err := r.pool.QueryRow(ctx, query, companyID, moduleName).Scan(
		&m.ID, &m.CompanyID, &m.ModuleName, &m.IsActive, &m.ActivatedAt, &m.ExpiresAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get company module: %w", err)
	}
	return &m, nil
}

// UpsertModule crea o actualiza el estado de un módulo en una empresa.
func (r *CompanyRepo) UpsertModule(ctx context.Context, module *entity.CompanyModule) error {
	const query = `
		INSERT INTO company_modules (id, company_id, module_name, is_active, activated_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (company_id, module_name)
		DO UPDATE SET
			is_active = EXCLUDED.is_active,
			activated_at = EXCLUDED.activated_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at`
	_, err := r.pool.Exec(ctx, query,
		module.ID,
		module.CompanyID,
		module.ModuleName,
		module.IsActive,
		module.ActivatedAt,
		module.ExpiresAt,
		module.CreatedAt,
		module.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert company module: %w", err)
	}
	return nil
}

// DeleteModule elimina la relación company-module.
func (r *CompanyRepo) DeleteModule(ctx context.Context, companyID, moduleName string) error {
	const query = `DELETE FROM company_modules WHERE company_id = $1 AND module_name = $2`
	if _, err := r.pool.Exec(ctx, query, companyID, moduleName); err != nil {
		return fmt.Errorf("delete company module: %w", err)
	}
	return nil
}

// HasActiveScreen informa si la empresa tiene la pantalla activa.
func (r *CompanyRepo) HasActiveScreen(ctx context.Context, companyID, screenID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM company_screens
			 WHERE company_id = $1
			   AND screen_id = $2
			   AND is_active = true
		)
	`
	var active bool
	if err := r.pool.QueryRow(ctx, query, companyID, screenID).Scan(&active); err != nil {
		return false, fmt.Errorf("check company screen %s: %w", screenID, err)
	}
	return active, nil
}

// ListScreens devuelve las pantallas habilitadas para la empresa.
func (r *CompanyRepo) ListScreens(ctx context.Context, companyID string) ([]*entity.CompanyScreen, error) {
	const query = `
		SELECT cs.company_id, cs.screen_id, s.key, s.name, m.key, m.name, s.frontend_route, s.api_endpoint,
		       cs.is_active
		FROM company_screens cs
		JOIN screens s ON s.id = cs.screen_id
		JOIN modules m ON m.id = s.module_id
		WHERE cs.company_id = $1
		ORDER BY m."order", s."order"`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("list company screens: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.CompanyScreen, 0)
	for rows.Next() {
		var cs entity.CompanyScreen
		if err := rows.Scan(
			&cs.CompanyID,
			&cs.ScreenID,
			&cs.ScreenKey,
			&cs.ScreenName,
			&cs.ModuleKey,
			&cs.ModuleName,
			&cs.FrontendRoute,
			&cs.ApiEndpoint,
			&cs.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan company screen: %w", err)
		}
		list = append(list, &cs)
	}
	if list == nil {
		list = make([]*entity.CompanyScreen, 0)
	}
	return list, rows.Err()
}

// GetScreen devuelve una pantalla específica de la empresa.
func (r *CompanyRepo) GetScreen(ctx context.Context, companyID, screenID string) (*entity.CompanyScreen, error) {
	const query = `
		SELECT cs.company_id, cs.screen_id, s.key, s.name, m.key, m.name, s.frontend_route, s.api_endpoint,
		       cs.is_active
		FROM company_screens cs
		JOIN screens s ON s.id = cs.screen_id
		JOIN modules m ON m.id = s.module_id
		WHERE cs.company_id = $1 AND cs.screen_id = $2
		LIMIT 1`
	var cs entity.CompanyScreen
	err := r.pool.QueryRow(ctx, query, companyID, screenID).Scan(
		&cs.CompanyID,
		&cs.ScreenID,
		&cs.ScreenKey,
		&cs.ScreenName,
		&cs.ModuleKey,
		&cs.ModuleName,
		&cs.FrontendRoute,
		&cs.ApiEndpoint,
		&cs.IsActive,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get company screen: %w", err)
	}
	return &cs, nil
}

// UpsertScreen crea o actualiza el estado de una pantalla para una empresa.
func (r *CompanyRepo) UpsertScreen(ctx context.Context, screen *entity.CompanyScreen) error {
	const query = `
		INSERT INTO company_screens (company_id, screen_id, is_active)
		VALUES ($1, $2, $3)
		ON CONFLICT (company_id, screen_id)
		DO UPDATE SET
			is_active = EXCLUDED.is_active`
	_, err := r.pool.Exec(ctx, query,
		screen.CompanyID,
		screen.ScreenID,
		screen.IsActive,
	)
	if err != nil {
		return fmt.Errorf("upsert company screen: %w", err)
	}
	return nil
}

// DeleteScreen desactiva la pantalla para la empresa.
func (r *CompanyRepo) DeleteScreen(ctx context.Context, companyID, screenID string) error {
	const query = `
		UPDATE company_screens
		SET is_active = false
		WHERE company_id = $1 AND screen_id = $2`
	if _, err := r.pool.Exec(ctx, query, companyID, screenID); err != nil {
		return fmt.Errorf("delete company screen: %w", err)
	}
	return nil
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

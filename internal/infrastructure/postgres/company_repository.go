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
	query := `
		INSERT INTO companies (id, name, nit, address, phone, email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(context.Background(), query,
		company.ID, company.Name, company.NIT, company.Address,
		company.Phone, company.Email, company.Status,
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
		SELECT id, name, nit, address, phone, email, status, created_at, updated_at
		FROM companies WHERE id = $1`
	var c entity.Company
	err := r.pool.QueryRow(context.Background(), query, id).Scan(
		&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status,
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
		SELECT id, name, nit, address, phone, email, status, created_at, updated_at
		FROM companies WHERE nit = $1`
	var c entity.Company
	err := r.pool.QueryRow(context.Background(), query, nit).Scan(
		&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status,
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
	query := `
		UPDATE companies SET name = $2, nit = $3, address = $4, phone = $5, email = $6, status = $7, updated_at = $8
		WHERE id = $1`
	cmd, err := r.pool.Exec(context.Background(), query,
		company.ID, company.Name, company.NIT, company.Address,
		company.Phone, company.Email, company.Status, company.UpdatedAt,
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
		SELECT id, name, nit, address, phone, email, status, created_at, updated_at
		FROM companies ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var list []*entity.Company
	for rows.Next() {
		var c entity.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.NIT, &c.Address, &c.Phone, &c.Email, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
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

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

var _ repository.CrmHubProductRepository = (*CrmHubProductRepository)(nil)

// CrmHubProductRepository catálogo crm_products_hub (CRUD administrativo).
type CrmHubProductRepository struct {
	q Querier
}

// NewCrmHubProductRepository constructor (pool o tx como Querier).
func NewCrmHubProductRepository(q Querier) *CrmHubProductRepository {
	return &CrmHubProductRepository{q: q}
}

func nullableDecimalArg(d *decimal.Decimal) interface{} {
	if d == nil {
		return nil
	}
	return *d
}

func scanCrmProductHub(row pgx.Row) (*entity.CrmProductHub, error) {
	var p entity.CrmProductHub
	var catID sql.NullString
	var nd decimal.NullDecimal
	err := row.Scan(
		&p.ID, &p.CompanyID, &catID, &p.ProductCode, &p.ProductName,
		&nd, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if catID.Valid {
		p.CategoryID = catID.String
	}
	if nd.Valid {
		d := nd.Decimal
		p.UnitCost = &d
	}
	return &p, nil
}

// Create inserta un producto hub.
func (r *CrmHubProductRepository) Create(p *entity.CrmProductHub) error {
	cid := strings.TrimSpace(p.CategoryID)
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_products_hub (
			id, company_id, category_id, product_code, product_name, unit_cost, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.ID, p.CompanyID, nullableUUIDArg(&cid), p.ProductCode, p.ProductName,
		nullableDecimalArg(p.UnitCost), p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert crm_products_hub: %w", err)
	}
	return nil
}

// GetByID obtiene por id (sin filtrar empresa; el caso de uso valida).
func (r *CrmHubProductRepository) GetByID(id string) (*entity.CrmProductHub, error) {
	row := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, category_id, product_code, product_name, unit_cost, is_active, created_at, updated_at
		FROM crm_products_hub WHERE id = $1`, id)
	return scanCrmProductHub(row)
}

// GetByCompanyAndProductCode busca por empresa y código de producto.
func (r *CrmHubProductRepository) GetByCompanyAndProductCode(companyID, productCode string) (*entity.CrmProductHub, error) {
	row := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, category_id, product_code, product_name, unit_cost, is_active, created_at, updated_at
		FROM crm_products_hub
		WHERE company_id = $1 AND product_code = $2`,
		companyID, productCode,
	)
	return scanCrmProductHub(row)
}

// Update actualiza un producto hub.
func (r *CrmHubProductRepository) Update(p *entity.CrmProductHub) error {
	cid := strings.TrimSpace(p.CategoryID)
	cmd, err := r.q.Exec(context.Background(), `
		UPDATE crm_products_hub
		SET category_id = $2, product_code = $3, product_name = $4, unit_cost = $5, is_active = $6, updated_at = $7
		WHERE id = $1 AND company_id = $8`,
		p.ID, nullableUUIDArg(&cid), p.ProductCode, p.ProductName,
		nullableDecimalArg(p.UnitCost), p.IsActive, p.UpdatedAt, p.CompanyID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("update crm_products_hub: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	return nil
}

func (r *CrmHubProductRepository) buildListWhere(companyID string, active *bool) (string, []any) {
	where := `FROM crm_products_hub WHERE company_id = $1`
	args := []any{companyID}
	if active != nil {
		args = append(args, *active)
		where += fmt.Sprintf(` AND is_active = $%d`, len(args))
	}
	return where, args
}

// ListByCompany lista con paginación y filtro opcional por is_active.
func (r *CrmHubProductRepository) ListByCompany(companyID string, limit, offset int, active *bool) ([]*entity.CrmProductHub, int64, error) {
	where, args := r.buildListWhere(companyID, active)
	var total int64
	countQ := `SELECT COUNT(*) ` + where
	if err := r.q.QueryRow(context.Background(), countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count crm_products_hub: %w", err)
	}
	argPos := len(args) + 1
	q := `
		SELECT id, company_id, category_id, product_code, product_name, unit_cost, is_active, created_at, updated_at
		` + where + fmt.Sprintf(` ORDER BY product_code ASC LIMIT $%d OFFSET $%d`, argPos, argPos+1)
	args = append(args, limit, offset)
	rows, err := r.q.Query(context.Background(), q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list crm_products_hub: %w", err)
	}
	defer rows.Close()
	var out []*entity.CrmProductHub
	for rows.Next() {
		var p entity.CrmProductHub
		var catID sql.NullString
		var nd decimal.NullDecimal
		if err := rows.Scan(&p.ID, &p.CompanyID, &catID, &p.ProductCode, &p.ProductName, &nd, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan crm_products_hub: %w", err)
		}
		if catID.Valid {
			p.CategoryID = catID.String
		}
		if nd.Valid {
			d := nd.Decimal
			p.UnitCost = &d
		}
		out = append(out, &p)
	}
	return out, total, rows.Err()
}

// Deactivate marca is_active = false.
func (r *CrmHubProductRepository) Deactivate(companyID, id string) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE crm_products_hub SET is_active = false, updated_at = now()
		WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		return fmt.Errorf("deactivate crm_products_hub: %w", err)
	}
	return nil
}

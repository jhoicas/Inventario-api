package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.CategoryRepository = (*CategoryRepo)(nil)

// CategoryRepo persiste en crm_category_product_hub.
type CategoryRepo struct {
	q Querier
}

// NewCategoryRepository adaptador CategoryRepository.
func NewCategoryRepository(q Querier) *CategoryRepo {
	return &CategoryRepo{q: q}
}

func (r *CategoryRepo) Create(c *entity.CrmCategoryProductHub) error {
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_category_product_hub (id, company_id, name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.CompanyID, strings.TrimSpace(c.Name), c.IsActive, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert crm_category_product_hub: %w", err)
	}
	return nil
}

func (r *CategoryRepo) GetByID(id string) (*entity.CrmCategoryProductHub, error) {
	row := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, name, is_active, created_at, updated_at
		FROM crm_category_product_hub WHERE id = $1`, id)
	return scanCategoryHub(row)
}

func scanCategoryHub(row pgx.Row) (*entity.CrmCategoryProductHub, error) {
	var c entity.CrmCategoryProductHub
	err := row.Scan(&c.ID, &c.CompanyID, &c.Name, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan crm_category_product_hub: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepo) GetByCompanyAndName(companyID, name string) (*entity.CrmCategoryProductHub, error) {
	name = strings.TrimSpace(name)
	if companyID == "" || name == "" {
		return nil, nil
	}
	row := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, name, is_active, created_at, updated_at
		FROM crm_category_product_hub
		WHERE company_id = $1 AND upper(trim(name)) = upper(trim($2))
		LIMIT 1`, companyID, name)
	return scanCategoryHub(row)
}

func (r *CategoryRepo) Update(c *entity.CrmCategoryProductHub) error {
	cmd, err := r.q.Exec(context.Background(), `
		UPDATE crm_category_product_hub
		SET name = $2, is_active = $3, updated_at = $4
		WHERE id = $1 AND company_id = $5`,
		c.ID, strings.TrimSpace(c.Name), c.IsActive, c.UpdatedAt, c.CompanyID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("update crm_category_product_hub: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	return nil
}

func (r *CategoryRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.CrmCategoryProductHub, int64, error) {
	var total int64
	err := r.q.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM crm_category_product_hub WHERE company_id = $1`, companyID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count crm_category_product_hub: %w", err)
	}
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, name, is_active, created_at, updated_at
		FROM crm_category_product_hub
		WHERE company_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list crm_category_product_hub: %w", err)
	}
	defer rows.Close()
	var out []*entity.CrmCategoryProductHub
	for rows.Next() {
		var c entity.CrmCategoryProductHub
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.Name, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan crm_category_product_hub: %w", err)
		}
		out = append(out, &c)
	}
	return out, total, rows.Err()
}

func (r *CategoryRepo) Deactivate(companyID, id string) error {
	cmd, err := r.q.Exec(context.Background(), `
		UPDATE crm_category_product_hub
		SET is_active = false, updated_at = now()
		WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		return fmt.Errorf("deactivate crm_category_product_hub: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	return nil
}

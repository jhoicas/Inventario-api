package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

var _ repository.CustomerRepository = (*CustomerRepo)(nil)

// CustomerRepo implementación de CustomerRepository (usable con pool o tx).
type CustomerRepo struct {
	q Querier
}

// NewCustomerRepository construye el adaptador. Pasar pool o tx (Querier).
func NewCustomerRepository(q Querier) *CustomerRepo {
	return &CustomerRepo{q: q}
}

// Create persiste un nuevo cliente.
func (r *CustomerRepo) Create(customer *entity.Customer) error {
	query := `
		INSERT INTO customers (id, company_id, name, tax_id, email, phone, birth_date, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.q.Exec(context.Background(), query,
		customer.ID, customer.CompanyID, customer.Name, customer.TaxID, customer.Email, customer.Phone, customer.BirthDate,
		customer.IsActive,
		customer.CreatedAt, customer.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert customer: %w", err)
	}
	return nil
}

// GetByID obtiene un cliente por ID.
func (r *CustomerRepo) GetByID(id string) (*entity.Customer, error) {
	query := `
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), birth_date, is_active, created_at, updated_at
		FROM customers WHERE id = $1`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.BirthDate, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}
	return &c, nil
}

// GetByCompanyAndTaxID obtiene un cliente por empresa y NIT/cédula.
func (r *CustomerRepo) GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error) {
	query := `
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), birth_date, is_active, created_at, updated_at
		FROM customers WHERE company_id = $1 AND tax_id = $2`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, companyID, taxID).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.BirthDate, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer by tax_id: %w", err)
	}
	return &c, nil
}

// GetByCompanyAndEmail obtiene un cliente por empresa y correo electrónico.
func (r *CustomerRepo) GetByCompanyAndEmail(companyID, email string) (*entity.Customer, error) {
	query := `
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), birth_date, is_active, created_at, updated_at
		FROM customers WHERE company_id = $1 AND LOWER(email) = LOWER($2)`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, companyID, email).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.BirthDate, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	return &c, nil
}

// ListByCompany lista clientes de la empresa con paginación.
func (r *CustomerRepo) ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, int64, error) {
	filters := repository.CustomerListFilters{Search: search}
	return r.ListByCompanyWithFilters(companyID, filters, limit, offset)
}

// ListByCompanyWithFilters lista clientes de la empresa con filtros avanzados y paginación.
func (r *CustomerRepo) ListByCompanyWithFilters(companyID string, filters repository.CustomerListFilters, limit, offset int) ([]*entity.Customer, int64, error) {
	joins, where, args := r.buildFilters(companyID, filters)
	items, err := r.list(joins, where, args, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.count(joins, where, args)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *CustomerRepo) buildFilters(companyID string, filters repository.CustomerListFilters) (string, string, []any) {
	joins := `
		LEFT JOIN crm_customer_profiles cp ON c.id = cp.customer_id
		LEFT JOIN crm_categories cat ON cp.category_id = cat.id`
	where := `
		WHERE c.company_id = $1 AND c.is_active = true`
	args := []any{companyID}

	if filters.WithoutCategory {
		where += ` AND cp.category_id IS NULL`
	}

	if !filters.WithoutCategory && strings.TrimSpace(filters.CategoryID) != "" {
		args = append(args, strings.TrimSpace(filters.CategoryID))
		where += fmt.Sprintf(" AND cp.category_id = $%d", len(args))
	} else if !filters.WithoutCategory && strings.TrimSpace(filters.CategoryNameLegacy) != "" {
		args = append(args, "%"+strings.TrimSpace(filters.CategoryNameLegacy)+"%")
		where += fmt.Sprintf(` AND cp.category_id IN (
			SELECT id FROM crm_categories
			WHERE company_id = $1 AND is_active = true AND name ILIKE $%d
		)`, len(args))
	}

	if strings.TrimSpace(filters.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filters.Search)+"%")
		where += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.email ILIKE $%d OR c.tax_id ILIKE $%d)", len(args), len(args), len(args))
	}
	return joins, where, args
}

func (r *CustomerRepo) list(joins, where string, args []any, limit, offset int) ([]*entity.Customer, error) {
	argPos := len(args) + 1
	query := `
		SELECT
			c.id,
			c.company_id,
			c.name,
			c.tax_id,
			COALESCE(c.email, ''),
			COALESCE(c.phone, ''),
			c.birth_date,
			c.is_active,
			c.created_at,
			c.updated_at,
			COALESCE(cp.ltv, 0) AS ltv,
			COALESCE(cat.name, '') AS category_name
		FROM customers c
		` + joins + "\n" + where + fmt.Sprintf(" ORDER BY c.name LIMIT $%d OFFSET $%d", argPos, argPos+1)
	qArgs := append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), query, qArgs...)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()
	items := make([]*entity.Customer, 0)
	for rows.Next() {
		var c entity.Customer
		var ltv pgtype.Numeric
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.BirthDate, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &ltv, &c.CategoryName); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		if ltv.Valid && ltv.Int != nil {
			c.LTV = decimal.NewFromBigInt(ltv.Int, ltv.Exp)
		} else {
			c.LTV = decimal.Zero
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CustomerRepo) count(joins, where string, args []any) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT c.id)
		FROM customers c
		` + joins + "\n" + where
	var total int64
	if err := r.q.QueryRow(context.Background(), query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count customers: %w", err)
	}
	return total, nil
}

// Update actualiza un cliente.
func (r *CustomerRepo) Update(customer *entity.Customer) error {
	query := `
		UPDATE customers SET name = $2, tax_id = $3, email = $4, phone = $5, birth_date = $6, updated_at = $7
		WHERE id = $1`
	_, err := r.q.Exec(context.Background(), query,
		customer.ID, customer.Name, customer.TaxID, customer.Email, customer.Phone, customer.BirthDate, customer.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("update customer: %w", err)
	}
	return nil
}

// Delete elimina un cliente por ID.
func (r *CustomerRepo) Delete(id string) error {
	_, err := r.q.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	return nil
}

func (r *CustomerRepo) SetActive(companyID, id string, isActive bool) error {
	_, err := r.q.Exec(context.Background(),
		`UPDATE customers SET is_active = $3, updated_at = now() WHERE id = $1 AND company_id = $2`,
		id, companyID, isActive,
	)
	if err != nil {
		return fmt.Errorf("set customer active: %w", err)
	}
	return nil
}

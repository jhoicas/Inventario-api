package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
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
		INSERT INTO customers (id, company_id, name, tax_id, email, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.q.Exec(context.Background(), query,
		customer.ID, customer.CompanyID, customer.Name, customer.TaxID, customer.Email, customer.Phone,
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
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), is_active, created_at, updated_at
		FROM customers WHERE id = $1`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
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
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), is_active, created_at, updated_at
		FROM customers WHERE company_id = $1 AND tax_id = $2`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, companyID, taxID).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
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
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), is_active, created_at, updated_at
		FROM customers WHERE company_id = $1 AND LOWER(email) = LOWER($2)`
	var c entity.Customer
	err := r.q.QueryRow(context.Background(), query, companyID, email).Scan(
		&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
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
func (r *CustomerRepo) ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, error) {
	base := `
		SELECT id, company_id, name, tax_id, COALESCE(email, ''), COALESCE(phone, ''), is_active, created_at, updated_at
		FROM customers
		WHERE company_id = $1 AND is_active = true`
	args := []any{companyID}
	argIdx := 2
	if search != "" {
		base += fmt.Sprintf(" AND (name ILIKE $%d OR tax_id ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	base += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), base, args...)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()
	var list []*entity.Customer
	for rows.Next() {
		var c entity.Customer
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.Name, &c.TaxID, &c.Email, &c.Phone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

// Update actualiza un cliente.
func (r *CustomerRepo) Update(customer *entity.Customer) error {
	query := `
		UPDATE customers SET name = $2, tax_id = $3, email = $4, phone = $5, updated_at = $6
		WHERE id = $1`
	_, err := r.q.Exec(context.Background(), query,
		customer.ID, customer.Name, customer.TaxID, customer.Email, customer.Phone, customer.UpdatedAt,
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

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

var _ repository.SupplierRepository = (*SupplierRepo)(nil)

// SupplierRepo implementación del puerto SupplierRepository sobre PostgreSQL.
type SupplierRepo struct {
	q Querier
}

// NewSupplierRepository construye el adaptador de persistencia para proveedores.
func NewSupplierRepository(q Querier) *SupplierRepo {
	return &SupplierRepo{q: q}
}

// Create persiste un nuevo proveedor.
func (r *SupplierRepo) Create(supplier *entity.Supplier) error {
	const query = `
		INSERT INTO suppliers (id, company_id, name, nit, email, phone, payment_term_days, lead_time_days, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.q.Exec(context.Background(), query,
		supplier.ID,
		supplier.CompanyID,
		supplier.Name,
		supplier.NIT,
		supplier.Email,
		supplier.Phone,
		supplier.PaymentTermDays,
		supplier.LeadTimeDays,
		supplier.CreatedAt,
		supplier.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert supplier: %w", err)
	}

	return nil
}

// GetByID obtiene un proveedor por ID.
func (r *SupplierRepo) GetByID(id string) (*entity.Supplier, error) {
	const query = `
		SELECT id, company_id, name, nit, email, phone, payment_term_days, lead_time_days, created_at, updated_at
		FROM suppliers
		WHERE id = $1`

	var s entity.Supplier
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&s.ID,
		&s.CompanyID,
		&s.Name,
		&s.NIT,
		&s.Email,
		&s.Phone,
		&s.PaymentTermDays,
		&s.LeadTimeDays,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get supplier: %w", err)
	}

	return &s, nil
}

// GetByCompanyAndNIT obtiene un proveedor por empresa y NIT.
func (r *SupplierRepo) GetByCompanyAndNIT(companyID, nit string) (*entity.Supplier, error) {
	const query = `
		SELECT id, company_id, name, nit, email, phone, payment_term_days, lead_time_days, created_at, updated_at
		FROM suppliers
		WHERE company_id = $1 AND nit = $2`

	var s entity.Supplier
	err := r.q.QueryRow(context.Background(), query, companyID, nit).Scan(
		&s.ID,
		&s.CompanyID,
		&s.Name,
		&s.NIT,
		&s.Email,
		&s.Phone,
		&s.PaymentTermDays,
		&s.LeadTimeDays,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get supplier by nit: %w", err)
	}

	return &s, nil
}

// Update actualiza un proveedor existente.
func (r *SupplierRepo) Update(supplier *entity.Supplier) error {
	const query = `
		UPDATE suppliers
		SET name = $2,
		    nit = $3,
		    email = $4,
		    phone = $5,
		    payment_term_days = $6,
		    lead_time_days = $7,
		    updated_at = $8
		WHERE id = $1`

	_, err := r.q.Exec(context.Background(), query,
		supplier.ID,
		supplier.Name,
		supplier.NIT,
		supplier.Email,
		supplier.Phone,
		supplier.PaymentTermDays,
		supplier.LeadTimeDays,
		supplier.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("update supplier: %w", err)
	}

	return nil
}

// ListByCompany lista proveedores por empresa con búsqueda y paginación.
func (r *SupplierRepo) ListByCompany(companyID, search string, limit, offset int) ([]*entity.Supplier, error) {
	const query = `
		SELECT id, company_id, name, nit, email, phone, payment_term_days, lead_time_days, created_at, updated_at
		FROM suppliers
		WHERE company_id = $1
		  AND (
			$2 = ''
			OR name ILIKE '%' || $2 || '%'
			OR nit  ILIKE '%' || $2 || '%'
		  )
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.q.Query(context.Background(), query, companyID, search, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.Supplier, 0)
	for rows.Next() {
		var s entity.Supplier
		if err := rows.Scan(
			&s.ID,
			&s.CompanyID,
			&s.Name,
			&s.NIT,
			&s.Email,
			&s.Phone,
			&s.PaymentTermDays,
			&s.LeadTimeDays,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier: %w", err)
		}
		list = append(list, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suppliers: %w", err)
	}

	return list, nil
}

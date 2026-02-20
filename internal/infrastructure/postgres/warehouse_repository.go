package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

var _ repository.WarehouseRepository = (*WarehouseRepo)(nil)

// WarehouseRepo implementación del puerto WarehouseRepository sobre PostgreSQL.
type WarehouseRepo struct {
	pool *pgxpool.Pool
}

// NewWarehouseRepository construye el adaptador de persistencia para bodegas.
func NewWarehouseRepository(pool *pgxpool.Pool) *WarehouseRepo {
	return &WarehouseRepo{pool: pool}
}

// Create persiste una nueva bodega.
func (r *WarehouseRepo) Create(warehouse *entity.Warehouse) error {
	query := `
		INSERT INTO warehouses (id, company_id, name, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(context.Background(), query,
		warehouse.ID, warehouse.CompanyID, warehouse.Name, warehouse.Address,
		warehouse.CreatedAt, warehouse.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert warehouse: %w", err)
	}
	return nil
}

// GetByID obtiene una bodega por ID.
func (r *WarehouseRepo) GetByID(id string) (*entity.Warehouse, error) {
	query := `
		SELECT id, company_id, name, address, created_at, updated_at
		FROM warehouses WHERE id = $1`
	var w entity.Warehouse
	err := r.pool.QueryRow(context.Background(), query, id).Scan(
		&w.ID, &w.CompanyID, &w.Name, &w.Address, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get warehouse: %w", err)
	}
	return &w, nil
}

// Update actualiza una bodega existente.
func (r *WarehouseRepo) Update(warehouse *entity.Warehouse) error {
	query := `
		UPDATE warehouses SET name = $2, address = $3, updated_at = $4
		WHERE id = $1`
	cmd, err := r.pool.Exec(context.Background(), query,
		warehouse.ID, warehouse.Name, warehouse.Address, warehouse.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update warehouse: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	return nil
}

// ListByCompany lista bodegas por empresa con paginación.
func (r *WarehouseRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.Warehouse, error) {
	query := `
		SELECT id, company_id, name, address, created_at, updated_at
		FROM warehouses WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(context.Background(), query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()
	var list []*entity.Warehouse
	for rows.Next() {
		var w entity.Warehouse
		if err := rows.Scan(&w.ID, &w.CompanyID, &w.Name, &w.Address, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan warehouse: %w", err)
		}
		list = append(list, &w)
	}
	return list, rows.Err()
}

// Delete elimina una bodega por ID.
func (r *WarehouseRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM warehouses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete warehouse: %w", err)
	}
	return nil
}

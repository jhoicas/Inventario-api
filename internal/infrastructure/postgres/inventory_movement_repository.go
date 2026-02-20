package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

var _ repository.InventoryMovementRepository = (*InventoryMovementRepo)(nil)

// InventoryMovementRepo implementación sobre PostgreSQL (usable con pool o tx).
type InventoryMovementRepo struct {
	q Querier
}

// NewInventoryMovementRepository construye el adaptador. Pasar pool o tx (Querier).
func NewInventoryMovementRepository(q Querier) *InventoryMovementRepo {
	return &InventoryMovementRepo{q: q}
}

// Create persiste un movimiento de inventario.
func (r *InventoryMovementRepo) Create(movement *entity.InventoryMovement) error {
	if movement.ID == "" {
		movement.ID = uuid.New().String()
	}
	query := `
		INSERT INTO inventory_movements (id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, date, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	createdBy := (*string)(nil)
	if movement.CreatedBy != "" {
		createdBy = &movement.CreatedBy
	}
	_, err := r.q.Exec(context.Background(), query,
		movement.ID, movement.TransactionID, movement.ProductID, movement.WarehouseID,
		movement.Type, movement.Quantity, movement.UnitCost, movement.TotalCost,
		movement.Date, movement.CreatedAt, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create inventory movement: %w", err)
	}
	return nil
}

// GetByID obtiene un movimiento por ID.
func (r *InventoryMovementRepo) GetByID(id string) (*entity.InventoryMovement, error) {
	query := `
		SELECT id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, date, created_at, created_by
		FROM inventory_movements WHERE id = $1`
	var m entity.InventoryMovement
	var createdBy *string
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&m.ID, &m.TransactionID, &m.ProductID, &m.WarehouseID, &m.Type,
		&m.Quantity, &m.UnitCost, &m.TotalCost, &m.Date, &m.CreatedAt, &createdBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get movement: %w", err)
	}
	if createdBy != nil {
		m.CreatedBy = *createdBy
	}
	return &m, nil
}

// ListByWarehouse lista movimientos de una bodega en un rango de fechas.
func (r *InventoryMovementRepo) ListByWarehouse(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error) {
	query := `
		SELECT id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, date, created_at, created_by
		FROM inventory_movements WHERE warehouse_id = $1`
	args := []any{warehouseID}
	pos := 2
	if from != nil {
		query += fmt.Sprintf(" AND date >= $%d", pos)
		args = append(args, *from)
		pos++
	}
	if to != nil {
		query += fmt.Sprintf(" AND date <= $%d", pos)
		args = append(args, *to)
		pos++
	}
	query += fmt.Sprintf(" ORDER BY date DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list by warehouse: %w", err)
	}
	defer rows.Close()
	var list []*entity.InventoryMovement
	for rows.Next() {
		var m entity.InventoryMovement
		var createdBy *string
		if err := rows.Scan(&m.ID, &m.TransactionID, &m.ProductID, &m.WarehouseID, &m.Type,
			&m.Quantity, &m.UnitCost, &m.TotalCost, &m.Date, &m.CreatedAt, &createdBy); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		if createdBy != nil {
			m.CreatedBy = *createdBy
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

// ListByProduct lista movimientos de un producto en un rango de fechas.
func (r *InventoryMovementRepo) ListByProduct(productID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error) {
	query := `
		SELECT id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, date, created_at, created_by
		FROM inventory_movements WHERE product_id = $1`
	args := []any{productID}
	pos := 2
	if from != nil {
		query += fmt.Sprintf(" AND date >= $%d", pos)
		args = append(args, *from)
		pos++
	}
	if to != nil {
		query += fmt.Sprintf(" AND date <= $%d", pos)
		args = append(args, *to)
		pos++
	}
	query += fmt.Sprintf(" ORDER BY date DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list by product: %w", err)
	}
	defer rows.Close()
	var list []*entity.InventoryMovement
	for rows.Next() {
		var m entity.InventoryMovement
		var createdBy *string
		if err := rows.Scan(&m.ID, &m.TransactionID, &m.ProductID, &m.WarehouseID, &m.Type,
			&m.Quantity, &m.UnitCost, &m.TotalCost, &m.Date, &m.CreatedAt, &createdBy); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		if createdBy != nil {
			m.CreatedBy = *createdBy
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

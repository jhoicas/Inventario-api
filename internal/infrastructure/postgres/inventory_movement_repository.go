package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
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
		INSERT INTO inventory_movements (id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, notes, date, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	createdBy := (*string)(nil)
	if movement.CreatedBy != "" {
		createdBy = &movement.CreatedBy
	}
	notes := (*string)(nil)
	if movement.Notes != "" {
		notes = &movement.Notes
	}
	_, err := r.q.Exec(context.Background(), query,
		movement.ID, movement.TransactionID, movement.ProductID, movement.WarehouseID,
		movement.Type, movement.Quantity, movement.UnitCost, movement.TotalCost,
		notes, movement.Date, movement.CreatedAt, createdBy,
	)
	if err != nil {
		return fmt.Errorf("create inventory movement: %w", err)
	}
	return nil
}

// GetByID obtiene un movimiento por ID.
func (r *InventoryMovementRepo) GetByID(id string) (*entity.InventoryMovement, error) {
	query := `
		SELECT id, transaction_id, product_id, warehouse_id, type, quantity, unit_cost, total_cost, notes, date, created_at, created_by
		FROM inventory_movements WHERE id = $1`
	var m entity.InventoryMovement
	var createdBy *string
	var notes *string
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&m.ID, &m.TransactionID, &m.ProductID, &m.WarehouseID, &m.Type,
		&m.Quantity, &m.UnitCost, &m.TotalCost, &notes, &m.Date, &m.CreatedAt, &createdBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get movement: %w", err)
	}
	if notes != nil {
		m.Notes = *notes
	}
	if createdBy != nil {
		m.CreatedBy = *createdBy
	}
	return &m, nil
}

// List devuelve movimientos filtrados por empresa y filtros opcionales, con total para paginación.
func (r *InventoryMovementRepo) List(companyID string, f repository.MovementFilters) ([]*entity.InventoryMovement, int64, error) {
	conds := []string{
		"EXISTS (SELECT 1 FROM products p WHERE p.id = im.product_id AND p.company_id = $1)",
	}
	args := []any{companyID}
	pos := 2

	if f.ProductID != "" {
		conds = append(conds, fmt.Sprintf("im.product_id = $%d", pos))
		args = append(args, f.ProductID)
		pos++
	}
	if f.WarehouseID != "" {
		conds = append(conds, fmt.Sprintf("im.warehouse_id = $%d", pos))
		args = append(args, f.WarehouseID)
		pos++
	}
	if f.Type != "" {
		conds = append(conds, fmt.Sprintf("im.type = $%d", pos))
		args = append(args, f.Type)
		pos++
	}
	if !f.StartDate.IsZero() {
		conds = append(conds, fmt.Sprintf("im.date >= $%d", pos))
		args = append(args, f.StartDate)
		pos++
	}
	if !f.EndDate.IsZero() {
		conds = append(conds, fmt.Sprintf("im.date <= $%d", pos))
		args = append(args, f.EndDate)
		pos++
	}

	where := strings.Join(conds, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(1)
		FROM inventory_movements im
		WHERE %s`, where)

	var total int64
	if err := r.q.QueryRow(context.Background(), countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count movements: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT im.id, im.transaction_id, im.product_id, im.warehouse_id,
		       im.type, im.quantity, im.unit_cost, im.total_cost, im.notes, im.date, im.created_at, im.created_by
		FROM inventory_movements im
		WHERE %s
		ORDER BY im.date ASC, im.created_at ASC
		LIMIT $%d OFFSET $%d`, where, pos, pos+1)

	dataArgs := append(args, limit, offset)
	rows, err := r.q.Query(context.Background(), dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.InventoryMovement, 0)
	for rows.Next() {
		var m entity.InventoryMovement
		var createdBy *string
		var notes *string
		if err := rows.Scan(
			&m.ID, &m.TransactionID, &m.ProductID, &m.WarehouseID,
			&m.Type, &m.Quantity, &m.UnitCost, &m.TotalCost,
			&notes, &m.Date, &m.CreatedAt, &createdBy,
		); err != nil {
			return nil, 0, fmt.Errorf("scan movement list: %w", err)
		}
		if notes != nil {
			m.Notes = *notes
		}
		if createdBy != nil {
			m.CreatedBy = *createdBy
		}
		list = append(list, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate movement list: %w", err)
	}

	return list, total, nil
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

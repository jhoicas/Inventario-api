package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.InventoryLevelRepository = (*InventoryLevelRepo)(nil)

// InventoryLevelRepo implementación de InventoryLevelRepository sobre PostgreSQL.
type InventoryLevelRepo struct {
	q Querier
}

// NewInventoryLevelRepository construye el adaptador. Acepta pool o tx (Querier).
func NewInventoryLevelRepository(q Querier) *InventoryLevelRepo {
	return &InventoryLevelRepo{q: q}
}

func (r *InventoryLevelRepo) Get(companyID, warehouseID, productID string) (*entity.InventoryLevel, error) {
	query := `
		SELECT p.company_id, s.warehouse_id, s.product_id, s.quantity, s.updated_at
		FROM stock s
		JOIN products p ON p.id = s.product_id
		WHERE p.company_id = $1 AND s.warehouse_id = $2 AND s.product_id = $3`
	var l entity.InventoryLevel
	err := r.q.QueryRow(context.Background(), query, companyID, warehouseID, productID).Scan(
		&l.CompanyID, &l.WarehouseID, &l.ProductID, &l.Quantity, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get inventory level: %w", err)
	}
	return &l, nil
}

func (r *InventoryLevelRepo) Upsert(level *entity.InventoryLevel) error {
	query := `
		INSERT INTO stock (product_id, warehouse_id, quantity, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (product_id, warehouse_id)
		DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()`
	_, err := r.q.Exec(context.Background(), query, level.ProductID, level.WarehouseID, level.Quantity)
	if err != nil {
		return fmt.Errorf("upsert inventory level: %w", err)
	}
	return nil
}

func (r *InventoryLevelRepo) ListByWarehouse(warehouseID string, limit, offset int) ([]*entity.InventoryLevel, error) {
	query := `
		SELECT p.company_id, s.warehouse_id, s.product_id, s.quantity, s.updated_at
		FROM stock s
		JOIN products p ON p.id = s.product_id
		WHERE s.warehouse_id = $1
		ORDER BY s.updated_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.q.Query(context.Background(), query, warehouseID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list inventory levels by warehouse: %w", err)
	}
	defer rows.Close()
	var list []*entity.InventoryLevel
	for rows.Next() {
		var l entity.InventoryLevel
		if err := rows.Scan(&l.CompanyID, &l.WarehouseID, &l.ProductID, &l.Quantity, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan inventory level: %w", err)
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

func (r *InventoryLevelRepo) ListByProduct(productID string) ([]*entity.InventoryLevel, error) {
	query := `
		SELECT p.company_id, s.warehouse_id, s.product_id, s.quantity, s.updated_at
		FROM stock s
		JOIN products p ON p.id = s.product_id
		WHERE s.product_id = $1
		ORDER BY s.updated_at DESC`
	rows, err := r.q.Query(context.Background(), query, productID)
	if err != nil {
		return nil, fmt.Errorf("list inventory levels by product: %w", err)
	}
	defer rows.Close()
	var list []*entity.InventoryLevel
	for rows.Next() {
		var l entity.InventoryLevel
		if err := rows.Scan(&l.CompanyID, &l.WarehouseID, &l.ProductID, &l.Quantity, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan inventory level: %w", err)
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

// GetProductsBelowReorderPoint devuelve los productos de la empresa cuyo stock actual
// (en la bodega indicada) es menor que su punto de reorden.
// Si warehouseID es vacío, considera el stock agregado de todas las bodegas.
// Ordena por déficit descendente (mayor quiebre primero).
func (r *InventoryLevelRepo) GetProductsBelowReorderPoint(ctx context.Context, companyID, warehouseID string) ([]repository.ReplenishmentItem, error) {
	var (
		query string
		args  []any
	)

	if warehouseID != "" {
		query = `
			SELECT
				p.id,
				p.sku,
				p.name,
				COALESCE(s.quantity, 0)  AS current_stock,
				p.reorder_point,
				p.cost,
				p.price
			FROM products p
			LEFT JOIN stock s ON s.product_id = p.id AND s.warehouse_id = $2
			WHERE p.company_id    = $1
			  AND p.reorder_point > 0
			  AND COALESCE(s.quantity, 0) < p.reorder_point
			ORDER BY (p.reorder_point - COALESCE(s.quantity, 0)) DESC`
		args = []any{companyID, warehouseID}
	} else {
		query = `
			SELECT
				p.id,
				p.sku,
				p.name,
				COALESCE(SUM(s.quantity), 0) AS current_stock,
				p.reorder_point,
				p.cost,
				p.price
			FROM products p
			LEFT JOIN stock s ON s.product_id = p.id
			WHERE p.company_id    = $1
			  AND p.reorder_point > 0
			GROUP BY p.id, p.sku, p.name, p.reorder_point, p.cost, p.price
			HAVING COALESCE(SUM(s.quantity), 0) < p.reorder_point
			ORDER BY (p.reorder_point - COALESCE(SUM(s.quantity), 0)) DESC`
		args = []any{companyID}
	}

	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get products below reorder point: %w", err)
	}
	defer rows.Close()

	var items []repository.ReplenishmentItem
	for rows.Next() {
		var item repository.ReplenishmentItem
		if err := rows.Scan(
			&item.ProductID, &item.SKU, &item.ProductName,
			&item.CurrentStock, &item.ReorderPoint,
			&item.UnitCost, &item.Price,
		); err != nil {
			return nil, fmt.Errorf("scan replenishment item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

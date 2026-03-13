package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

var _ repository.StockRepository = (*StockRepo)(nil)

// StockRepo implementación de StockRepository sobre PostgreSQL (usable con pool o tx).
type StockRepo struct {
	q Querier
}

// NewStockRepository construye el adaptador de stock. Pasar pool o tx (Querier).
func NewStockRepository(q Querier) *StockRepo {
	return &StockRepo{q: q}
}

// Get obtiene el stock actual de un producto en una bodega.
func (r *StockRepo) Get(productID, warehouseID string) (*entity.Stock, error) {
	query := `
		SELECT product_id, warehouse_id, quantity, updated_at
		FROM stock WHERE product_id = $1 AND warehouse_id = $2`
	var s entity.Stock
	err := r.q.QueryRow(context.Background(), query, productID, warehouseID).Scan(
		&s.ProductID, &s.WarehouseID, &s.Quantity, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &entity.Stock{ProductID: productID, WarehouseID: warehouseID, Quantity: decimal.Zero}, nil
		}
		return nil, fmt.Errorf("get stock: %w", err)
	}
	return &s, nil
}

// GetByProduct lista el stock de un producto en todas las bodegas.
func (r *StockRepo) GetByProduct(productID string) ([]*entity.Stock, error) {
	const query = `
		SELECT product_id, warehouse_id, quantity, updated_at
		FROM stock
		WHERE product_id = $1
		ORDER BY warehouse_id`

	rows, err := r.q.Query(context.Background(), query, productID)
	if err != nil {
		return nil, fmt.Errorf("list stock by product: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.Stock, 0)
	for rows.Next() {
		var s entity.Stock
		if err := rows.Scan(&s.ProductID, &s.WarehouseID, &s.Quantity, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan stock by product: %w", err)
		}
		list = append(list, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock by product: %w", err)
	}

	return list, nil
}

// GetSummary devuelve un resumen de stock para un producto (y opcionalmente una bodega).
func (r *StockRepo) GetSummary(productID, warehouseID string) (*repository.StockSummary, error) {
	query := `
		SELECT COALESCE(SUM(quantity), 0), COALESCE(MAX(updated_at), now())
		FROM stock
		WHERE product_id = $1`
	args := []any{productID}
	if warehouseID != "" {
		query += ` AND warehouse_id = $2`
		args = append(args, warehouseID)
	}

	var current decimal.Decimal
	var lastUpdated time.Time
	if err := r.q.QueryRow(context.Background(), query, args...).Scan(&current, &lastUpdated); err != nil {
		return nil, fmt.Errorf("get stock summary: %w", err)
	}

	summary := &repository.StockSummary{
		CurrentStock:   current,
		ReservedStock:  decimal.Zero,
		AvailableStock: current,
		AvgCost:        decimal.Zero,
		LastUpdated:    lastUpdated,
	}

	return summary, nil
}

// Upsert inserta o actualiza la cantidad en stock (por producto y bodega).
func (r *StockRepo) Upsert(stock *entity.Stock) error {
	query := `
		INSERT INTO stock (product_id, warehouse_id, quantity, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (product_id, warehouse_id)
		DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()`
	_, err := r.q.Exec(context.Background(), query, stock.ProductID, stock.WarehouseID, stock.Quantity)
	if err != nil {
		return fmt.Errorf("upsert stock: %w", err)
	}
	return nil
}

// GetForUpdate obtiene el stock y bloquea la fila para update (SELECT FOR UPDATE).
func (r *StockRepo) GetForUpdate(productID, warehouseID string) (*entity.Stock, error) {
	query := `
		SELECT product_id, warehouse_id, quantity, updated_at
		FROM stock WHERE product_id = $1 AND warehouse_id = $2
		FOR UPDATE`
	var s entity.Stock
	err := r.q.QueryRow(context.Background(), query, productID, warehouseID).Scan(
		&s.ProductID, &s.WarehouseID, &s.Quantity, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &entity.Stock{ProductID: productID, WarehouseID: warehouseID, Quantity: decimal.Zero}, nil
		}
		return nil, fmt.Errorf("get stock for update: %w", err)
	}
	return &s, nil
}

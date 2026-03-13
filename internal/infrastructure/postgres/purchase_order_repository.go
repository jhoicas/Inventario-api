package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
)

var _ inventory.PurchaseOrderRepository = (*PurchaseOrderRepo)(nil)

type PurchaseOrderRepo struct {
	q Querier
}

func NewPurchaseOrderRepository(q Querier) *PurchaseOrderRepo {
	return &PurchaseOrderRepo{q: q}
}

func (r *PurchaseOrderRepo) Create(ctx context.Context, po *entity.PurchaseOrder) error {
	tx, shouldCommit, committed, err := beginIfPossible(ctx, r.q)
	if err != nil {
		return fmt.Errorf("begin purchase order create tx: %w", err)
	}
	defer rollbackUnlessCommitted(ctx, tx, shouldCommit, &committed)

	const insertPO = `
		INSERT INTO purchase_orders (id, company_id, supplier_id, number, date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if _, err := tx.Exec(ctx, insertPO,
		po.ID,
		po.CompanyID,
		po.SupplierID,
		po.Number,
		po.Date,
		po.Status,
		po.CreatedAt,
		po.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert purchase order: %w", err)
	}

	const insertItem = `
		INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost)
		VALUES ($1, $2, $3, $4)`

	for _, item := range po.Items {
		if _, err := tx.Exec(ctx, insertItem,
			po.ID,
			item.ProductID,
			item.Quantity,
			item.UnitCost,
		); err != nil {
			return fmt.Errorf("insert purchase order item: %w", err)
		}
	}

	if shouldCommit {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit purchase order create: %w", err)
		}
		committed = true
	}
	return nil
}

func (r *PurchaseOrderRepo) GetByID(ctx context.Context, id string) (*entity.PurchaseOrder, error) {
	const queryPO = `
		SELECT id, company_id, supplier_id, number, date, status, created_at, updated_at
		FROM purchase_orders
		WHERE id = $1`

	var po entity.PurchaseOrder
	err := r.q.QueryRow(ctx, queryPO, id).Scan(
		&po.ID,
		&po.CompanyID,
		&po.SupplierID,
		&po.Number,
		&po.Date,
		&po.Status,
		&po.CreatedAt,
		&po.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get purchase order: %w", err)
	}

	const queryItems = `
		SELECT product_id, quantity, unit_cost
		FROM purchase_order_items
		WHERE purchase_order_id = $1
		ORDER BY created_at ASC`

	rows, err := r.q.Query(ctx, queryItems, id)
	if err != nil {
		return nil, fmt.Errorf("list purchase order items: %w", err)
	}
	defer rows.Close()

	items := make([]entity.PurchaseOrderItem, 0)
	for rows.Next() {
		var item entity.PurchaseOrderItem
		var qty decimal.Decimal
		var cost decimal.Decimal
		if err := rows.Scan(&item.ProductID, &qty, &cost); err != nil {
			return nil, fmt.Errorf("scan purchase order item: %w", err)
		}
		item.Quantity = qty
		item.UnitCost = cost
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchase order items: %w", err)
	}

	po.Items = items
	return &po, nil
}

func (r *PurchaseOrderRepo) UpdateStatus(ctx context.Context, id, status string, updatedAt time.Time) error {
	const query = `
		UPDATE purchase_orders
		SET status = $2,
		    updated_at = $3
		WHERE id = $1`

	res, err := r.q.Exec(ctx, query, id, status, updatedAt)
	if err != nil {
		return fmt.Errorf("update purchase order status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

type txControl interface {
	Querier
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func beginIfPossible(ctx context.Context, q Querier) (txControl, bool, bool, error) {
	if existingTx, ok := q.(txControl); ok {
		return existingTx, false, true, nil
	}
	starter, ok := q.(txStarter)
	if !ok {
		return nil, false, false, fmt.Errorf("querier does not support transactions")
	}
	tx, err := starter.Begin(ctx)
	if err != nil {
		return nil, false, false, err
	}
	return tx, true, false, nil
}

func rollbackUnlessCommitted(ctx context.Context, tx txControl, shouldCommit bool, committed *bool) {
	if tx == nil || !shouldCommit || committed == nil || *committed {
		return
	}
	_ = tx.Rollback(ctx)
}

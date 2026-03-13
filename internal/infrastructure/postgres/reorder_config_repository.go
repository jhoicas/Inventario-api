package postgres

import (
	"context"
	"fmt"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
)

var _ inventory.ReorderConfigRepository = (*ReorderConfigRepo)(nil)

// ReorderConfigRepo implementación PostgreSQL para product_reorder_config.
type ReorderConfigRepo struct {
	q Querier
}

func NewReorderConfigRepository(q Querier) *ReorderConfigRepo {
	return &ReorderConfigRepo{q: q}
}

// UpsertProductReorderConfig inserta/actualiza la configuración de reposición por producto y bodega.
func (r *ReorderConfigRepo) UpsertProductReorderConfig(ctx context.Context, in dto.ReorderConfigRequest) error {
	const query = `
		INSERT INTO product_reorder_config (
			product_id, warehouse_id, reorder_point, min_stock, max_stock, lead_time_days, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
		ON CONFLICT (product_id, warehouse_id)
		DO UPDATE SET
			reorder_point = EXCLUDED.reorder_point,
			min_stock = EXCLUDED.min_stock,
			max_stock = EXCLUDED.max_stock,
			lead_time_days = EXCLUDED.lead_time_days,
			updated_at = now()`

	if _, err := r.q.Exec(ctx, query,
		in.ProductID,
		in.WarehouseID,
		in.ReorderPoint,
		in.MinStock,
		in.MaxStock,
		in.LeadTimeDays,
	); err != nil {
		return fmt.Errorf("upsert product reorder config: %w", err)
	}

	return nil
}

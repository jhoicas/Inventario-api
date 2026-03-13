package inventory

import (
	"context"
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// TxRunner ejecuta una función dentro de una transacción de BD, pasando repositorios atados a esa tx.
// Garantiza atomicidad para el motor de inventario.
type TxRunner interface {
	Run(ctx context.Context, fn func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
	) error) error
}

// StockSnapshotRepository permite listar el stock actual para crear snapshots de conteo físico.
type StockSnapshotRepository interface {
	ListByWarehouse(ctx context.Context, companyID, warehouseID string) ([]*entity.Stock, error)
}

// StocktakeRepository define persistencia para sesiones de conteo físico.
type StocktakeRepository interface {
	Create(ctx context.Context, stocktake *entity.Stocktake, items []entity.StocktakeItem) error
	GetByID(ctx context.Context, stocktakeID string) (*entity.Stocktake, error)
	ListItems(ctx context.Context, stocktakeID string) ([]entity.StocktakeItem, error)
	UpdateCounts(ctx context.Context, stocktakeID string, items []entity.StocktakeItem) error
	MarkClosed(ctx context.Context, stocktakeID string, closedAt time.Time) error
}

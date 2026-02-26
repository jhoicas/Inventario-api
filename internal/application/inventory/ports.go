package inventory

import (
	"context"

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

package billing

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// BillingTxRunner ejecuta una función dentro de una transacción que incluye repos de inventario y facturación.
type BillingTxRunner interface {
	RunBilling(ctx context.Context, fn func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		customerRepo repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error) error
}

// InventoryUseCase interfaz para integrar facturación con inventario.
// RegisterOUTInTx ejecuta una salida (OUT) usando los repositorios del caller (misma transacción).
// Si retorna error (ej: ErrInsufficientStock), el caller debe hacer rollback.
type InventoryUseCase interface {
	RegisterOUTInTx(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		product *entity.Product,
		productID, warehouseID, userID string,
		quantity decimal.Decimal,
		now time.Time,
		transactionID string, // referencia a la factura (invoice ID)
	) error
}

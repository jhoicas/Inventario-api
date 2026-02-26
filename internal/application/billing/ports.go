package billing

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// InvoicePDFGenerator es el puerto de salida para la generación de representaciones
// gráficas (PDF) de facturas electrónicas DIAN. La implementación concreta se
// encuentra en internal/infrastructure/pdf/.
type InvoicePDFGenerator interface {
	// GenerateInvoicePDF genera el PDF de la factura y devuelve sus bytes.
	// details es la lista de líneas de detalle ya enriquecidas (con nombre de producto).
	GenerateInvoicePDF(
		ctx context.Context,
		invoice *entity.Invoice,
		company *entity.Company,
		customer *entity.Customer,
		details []InvoiceDetailForPDF,
	) ([]byte, error)
}

// InvoiceDetailForPDF agrega el nombre del producto a la línea de detalle,
// ya que entity.InvoiceDetail solo guarda el productID.
type InvoiceDetailForPDF struct {
	entity.InvoiceDetail
	ProductName string
}

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
// ctx propaga la transacción SQL activa. Si retorna ErrInsufficientStock el caller hace rollback.
type InventoryUseCase interface {
	RegisterOUTInTx(
		ctx context.Context,
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

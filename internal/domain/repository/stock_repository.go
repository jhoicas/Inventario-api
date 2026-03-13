package repository

import (
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// StockSummary representa un resumen agregado del inventario.
type StockSummary struct {
	CurrentStock   decimal.Decimal
	ReservedStock  decimal.Decimal
	AvailableStock decimal.Decimal
	AvgCost        decimal.Decimal
	LastUpdated    time.Time
}

// StockRepository define el puerto para consultar/actualizar stock por bodega+producto.
// Usado dentro de transacciones para garantizar consistencia.
type StockRepository interface {
	Get(productID, warehouseID string) (*entity.Stock, error)
	GetByProduct(productID string) ([]*entity.Stock, error)
	GetSummary(productID, warehouseID string) (*StockSummary, error)
	Upsert(stock *entity.Stock) error
	// GetForUpdate opcional: bloquea la fila para update (SELECT FOR UPDATE).
	GetForUpdate(productID, warehouseID string) (*entity.Stock, error)
}

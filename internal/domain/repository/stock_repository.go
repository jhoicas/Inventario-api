package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// StockRepository define el puerto para consultar/actualizar stock por bodega+producto.
// Usado dentro de transacciones para garantizar consistencia.
type StockRepository interface {
	Get(productID, warehouseID string) (*entity.Stock, error)
	Upsert(stock *entity.Stock) error
	// GetForUpdate opcional: bloquea la fila para update (SELECT FOR UPDATE).
	GetForUpdate(productID, warehouseID string) (*entity.Stock, error)
}

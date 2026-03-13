package inventory

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// GetStockUseCase obtiene el resumen de stock de un producto (una bodega o todas).
type GetStockUseCase struct {
	stockRepo repository.StockRepository
}

// NewGetStockUseCase construye el caso de uso.
func NewGetStockUseCase(stockRepo repository.StockRepository) *GetStockUseCase {
	return &GetStockUseCase{stockRepo: stockRepo}
}

// Execute devuelve el resumen de stock. Si warehouseID está vacío, agrega stocks de todas las bodegas.
// companyID se recibe para consistencia con otros use cases (validación de empresa puede hacerse en capa superior).
func (uc *GetStockUseCase) Execute(ctx context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error) {
	summary, err := uc.stockRepo.GetSummary(productID, warehouseID)
	if err != nil {
		return nil, err
	}
	return &dto.StockSummaryDTO{
		ProductID:      productID,
		WarehouseID:    warehouseID,
		CurrentStock:   summary.CurrentStock,
		ReservedStock:  summary.ReservedStock,
		AvailableStock: summary.AvailableStock,
		AvgCost:        summary.AvgCost,
		LastUpdated:    summary.LastUpdated,
	}, nil
}

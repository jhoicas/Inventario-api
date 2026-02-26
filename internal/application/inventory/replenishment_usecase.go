package inventory

import (
	"context"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// ReplenishmentUseCase genera la lista semanal de reposición para una bodega.
// Combina datos de stock con historial de márgenes para priorizar los SKUs críticos.
type ReplenishmentUseCase struct {
	levelRepo     repository.InventoryLevelRepository
	analyticsRepo repository.AnalyticsRepository
}

// NewReplenishmentUseCase construye el caso de uso de reposición.
func NewReplenishmentUseCase(
	levelRepo repository.InventoryLevelRepository,
	analyticsRepo repository.AnalyticsRepository,
) *ReplenishmentUseCase {
	return &ReplenishmentUseCase{
		levelRepo:     levelRepo,
		analyticsRepo: analyticsRepo,
	}
}

// GenerateReplenishmentList devuelve los productos bajo punto de reorden con la cantidad
// sugerida de pedido y un ranking de prioridad basado en margen histórico y volumen de ventas.
// warehouseID puede ser vacío para considerar stock global de la empresa.
func (uc *ReplenishmentUseCase) GenerateReplenishmentList(
	ctx context.Context,
	companyID, warehouseID string,
) ([]dto.ReplenishmentSuggestionDTO, error) {

	// 1. Productos por debajo del punto de reorden
	rawItems, err := uc.levelRepo.GetProductsBelowReorderPoint(ctx, companyID, warehouseID)
	if err != nil {
		return nil, err
	}
	if len(rawItems) == 0 {
		return []dto.ReplenishmentSuggestionDTO{}, nil
	}

	// 2. Historial de márgenes por SKU (últimos 90 días, todos los SKUs de la empresa)
	end := time.Now()
	start := end.AddDate(0, 0, -90)
	skuMetrics, _ := uc.analyticsRepo.GetSKUMargins(ctx, companyID, start, end, 500)

	// Construir lookup: productID → SKUMarginResult
	marginByID := make(map[string]repository.SKUMarginResult, len(skuMetrics))
	for _, m := range skuMetrics {
		marginByID[m.ProductID] = m
	}

	// 3. Construir los DTOs enriquecidos
	hundred := decimal.NewFromInt(100)

	suggestions := make([]dto.ReplenishmentSuggestionDTO, 0, len(rawItems))
	for _, item := range rawItems {
		idealStock := item.ReorderPoint.Mul(decimal.NewFromFloat(1.5))
		suggestedQty := idealStock.Sub(item.CurrentStock)
		if suggestedQty.LessThanOrEqual(decimal.Zero) {
			suggestedQty = decimal.Zero
		}

		estimatedCost := suggestedQty.Mul(item.UnitCost)

		var grossMarginPct, unitsSold decimal.Decimal
		if m, ok := marginByID[item.ProductID]; ok {
			unitsSold = m.UnitsSold
			if m.GrossRevenue.GreaterThan(decimal.Zero) {
				grossMarginPct = m.GrossProfit.Div(m.GrossRevenue).Mul(hundred).Round(2)
			}
		} else {
			// Sin historial de ventas: estimar margen por precio y costo
			if item.Price.GreaterThan(decimal.Zero) {
				grossMarginPct = item.Price.Sub(item.UnitCost).Div(item.Price).Mul(hundred).Round(2)
			}
		}

		suggestions = append(suggestions, dto.ReplenishmentSuggestionDTO{
			ProductID:           item.ProductID,
			SKU:                 item.SKU,
			ProductName:         item.ProductName,
			CurrentStock:        item.CurrentStock,
			ReorderPoint:        item.ReorderPoint,
			IdealStock:          idealStock,
			SuggestedOrderQty:   suggestedQty,
			UnitCost:            item.UnitCost,
			EstimatedOrderCost:  estimatedCost,
			GrossMarginPct:      grossMarginPct,
			UnitsSoldLast90Days: unitsSold,
		})
	}

	// 4. Ordenar: primero mayor margen histórico, luego mayor volumen de ventas,
	//    finalmente mayor déficit relativo (% de caída bajo el reorden).
	sort.SliceStable(suggestions, func(i, j int) bool {
		a, b := suggestions[i], suggestions[j]
		if !a.GrossMarginPct.Equal(b.GrossMarginPct) {
			return a.GrossMarginPct.GreaterThan(b.GrossMarginPct)
		}
		if !a.UnitsSoldLast90Days.Equal(b.UnitsSoldLast90Days) {
			return a.UnitsSoldLast90Days.GreaterThan(b.UnitsSoldLast90Days)
		}
		// Tiebreak: mayor déficit absoluto
		defA := a.ReorderPoint.Sub(a.CurrentStock)
		defB := b.ReorderPoint.Sub(b.CurrentStock)
		return defA.GreaterThan(defB)
	})

	// 5. Asignar prioridad (1 = más urgente)
	for i := range suggestions {
		suggestions[i].Priority = i + 1
	}

	return suggestions, nil
}

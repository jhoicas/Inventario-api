package entity

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Product representa un producto o SKU del inventario (multi-bodega).
// Cost es promedio ponderado calculado desde movimientos; Stock se maneja por bodega en InventoryLevel.
type Product struct {
	ID           string
	CompanyID    string
	SKU          string    // código único por empresa
	Name         string
	Description  string
	Price        decimal.Decimal // precio de venta
	Cost         decimal.Decimal // costo promedio ponderado (inicia en 0)
	TaxRate      decimal.Decimal // Porcentaje (ej: 19, 5, 0, 7.5). Se normaliza a fracción en cálculos.
	UNSPSC_Code  string
	UnitMeasure  string
	Attributes   json.RawMessage
	COGS         decimal.Decimal // costo de bienes vendidos (analítica)
	ReorderPoint decimal.Decimal // punto de reorden para alertas de ruptura
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IdealStock retorna el nivel de stock objetivo: 1.5× el punto de reorden.
// Se usa para calcular la cantidad sugerida de pedido en reposición.
func (p *Product) IdealStock() decimal.Decimal {
	return p.ReorderPoint.Mul(decimal.NewFromFloat(1.5))
}

// CalculateProductionCost calcula el costo de producción basado en la receta (BOM).
// Fórmula por ítem: (RawMaterial.Cost * QuantityRequired) * (1 + WastePercentage).
// WastePercentage debe venir en forma fraccional (ej: 0.05 = 5% merma).
func (p *Product) CalculateProductionCost(recipeItems []RecipeItem) decimal.Decimal {
	total := decimal.Zero
	for _, item := range recipeItems {
		if item.RawMaterial == nil {
			continue
		}
		qty := item.QuantityRequired
		if !qty.GreaterThan(decimal.Zero) {
			continue
		}
		cost := item.RawMaterial.Cost
		wasteFactor := decimal.NewFromInt(1).Add(item.WastePercentage)
		lineCost := cost.Mul(qty).Mul(wasteFactor)
		total = total.Add(lineCost)
	}
	return total
}

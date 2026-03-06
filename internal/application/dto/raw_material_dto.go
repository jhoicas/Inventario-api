package dto

import "github.com/shopspring/decimal"

// RawMaterialImpactDTO representa el impacto financiero de una materia prima
// en el portafolio de productos top Pareto (top 20% de ingresos).
type RawMaterialImpactDTO struct {
	RawMaterialID   string          `json:"raw_material_id"`
	SKU             string          `json:"sku"`
	Name            string          `json:"name"`
	TotalCostImpact decimal.Decimal `json:"total_cost_impact"` // costo total imputado en SKUs Pareto
	UsagePct        decimal.Decimal `json:"usage_pct"`         // participación % sobre el costo total de materias primas
}


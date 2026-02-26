package dto

import "github.com/shopspring/decimal"

// ── Query parameters ──────────────────────────────────────────────────────────

// MarginsReportRequest parámetros para GET /api/analytics/margins.
type MarginsReportRequest struct {
	StartDate string `query:"start_date"` // YYYY-MM-DD; por defecto primer día del mes actual
	EndDate   string `query:"end_date"`   // YYYY-MM-DD; por defecto hoy
	TopN      int    `query:"top_n"`      // máx SKUs a devolver (default 20, max 200)
}

// ── Por canal ─────────────────────────────────────────────────────────────────

// MarginByChannelDTO margen calculado por canal de venta.
// Fórmula: margen = ingresos_netos - cogs - comisión_canal
type MarginByChannelDTO struct {
	ChannelID      string          `json:"channel_id"`       // UUID o "direct"
	ChannelName    string          `json:"channel_name"`     // nombre del canal o "Directo"
	ChannelType    string          `json:"channel_type"`     // ecommerce|pos|b2b|marketplace|other
	CommissionRate decimal.Decimal `json:"commission_rate"`  // % de comisión del canal
	InvoiceCount   int             `json:"invoice_count"`    // facturas emitidas en el período
	UnitsSold      decimal.Decimal `json:"units_sold"`       // unidades totales
	GrossRevenue   decimal.Decimal `json:"gross_revenue"`    // ingresos brutos (suma de subtotales)
	TotalCOGS      decimal.Decimal `json:"total_cogs"`       // costo total (qty * costo_prom_producto)
	CommissionCost decimal.Decimal `json:"commission_cost"`  // ingresos * commission_rate / 100
	TotalMargin    decimal.Decimal `json:"total_margin"`     // GrossRevenue - COGS - CommissionCost
	MarginPct      decimal.Decimal `json:"margin_pct"`       // TotalMargin / GrossRevenue * 100
	RevenuePct     decimal.Decimal `json:"revenue_pct"`      // participación % en ingresos totales
}

// ChannelProfitabilityDTO resumen de rentabilidad con detalle por canal.
type ChannelProfitabilityDTO struct {
	TotalRevenue    decimal.Decimal      `json:"total_revenue"`
	TotalCOGS       decimal.Decimal      `json:"total_cogs"`
	TotalMargin     decimal.Decimal      `json:"total_margin"`
	OverallMarginPct decimal.Decimal     `json:"overall_margin_pct"` // margen ponderado global
	Channels        []MarginByChannelDTO `json:"channels"`
}

// ── Por SKU ───────────────────────────────────────────────────────────────────

// SKURankingDTO margen y rentabilidad por SKU/producto.
type SKURankingDTO struct {
	Rank             int             `json:"rank"`               // posición (1 = más rentable)
	ProductID        string          `json:"product_id"`
	SKU              string          `json:"sku"`
	ProductName      string          `json:"product_name"`
	UnitsSold        decimal.Decimal `json:"units_sold"`
	GrossRevenue     decimal.Decimal `json:"gross_revenue"`
	TotalCOGS        decimal.Decimal `json:"total_cogs"`
	GrossProfit      decimal.Decimal `json:"gross_profit"`   // GrossRevenue - TotalCOGS
	MarginPct        decimal.Decimal `json:"margin_pct"`     // GrossProfit / GrossRevenue * 100
	RevenuePct       decimal.Decimal `json:"revenue_pct"`    // participación % en ingresos totales
	CumulativeRevPct decimal.Decimal `json:"cumulative_revenue_pct"` // acumulado descendente
	IsTopPareto      bool            `json:"is_top_pareto"`  // true si forma parte del top 80% de ingresos (Pareto)
}

// ── Reporte combinado ─────────────────────────────────────────────────────────

// PeriodDTO rango de fechas del reporte.
type PeriodDTO struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// MarginsReportDTO respuesta completa de GET /api/analytics/margins.
type MarginsReportDTO struct {
	Period          PeriodDTO               `json:"period"`
	Profitability   ChannelProfitabilityDTO `json:"channel_profitability"`
	SKURanking      []SKURankingDTO         `json:"sku_ranking"`       // top N por margen
	ParetoSKUs      []SKURankingDTO         `json:"pareto_skus"`       // SKUs del top 20% que generan ~80% ingresos
}

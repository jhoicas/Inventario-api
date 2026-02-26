package dto

import "github.com/shopspring/decimal"

// DashboardSummaryDTO respuesta de GET /api/dashboard/summary.
// Contiene los KPIs principales del día y del mes en curso, más el Top-5 SKUs del mes.
type DashboardSummaryDTO struct {
	// Métricas del día actual (00:00 – 23:59)
	TodaySales  decimal.Decimal `json:"today_sales"`  // ingresos brutos de hoy
	TodayMargin decimal.Decimal `json:"today_margin"` // margen bruto de hoy (revenue - COGS)

	// Métricas del mes en curso (día 1 – hoy)
	MonthlySales  decimal.Decimal `json:"monthly_sales"`  // ingresos brutos del mes
	MonthlyMargin decimal.Decimal `json:"monthly_margin"` // margen bruto del mes

	// Top 5 SKUs por ingreso del mes (ordenados de mayor a menor revenue)
	TopSKUs []TopSKUDTO `json:"top_skus"`

	// Metadatos del período
	DateLabel string `json:"date_label"` // ej: "Febrero 2026"
}

// TopSKUDTO resumen de un SKU para el widget del dashboard.
// Derivado de SKURankingDTO pero más ligero (sin acumulados Pareto).
type TopSKUDTO struct {
	ProductID        string          `json:"product_id"`
	SKU              string          `json:"sku"`
	ProductName      string          `json:"product_name"`
	QuantitySold     decimal.Decimal `json:"quantity_sold"`
	TotalRevenue     decimal.Decimal `json:"total_revenue"`
	MarginPercentage decimal.Decimal `json:"margin_percentage"` // (revenue - cogs) / revenue * 100
}

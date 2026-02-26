package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
)

// ChannelSalesResult resultado crudo de la consulta de ventas por canal.
// Lo produce la DB; el use case lo convierte en DTO.
type ChannelSalesResult struct {
	ChannelID      string          // UUID o "direct" si la factura no tiene canal
	ChannelName    string          // Nombre del canal o "Directo"
	ChannelType    string          // ecommerce|pos|b2b|marketplace|other
	CommissionRate decimal.Decimal // Porcentaje de comisión del canal
	InvoiceCount   int
	UnitsSold      decimal.Decimal
	GrossRevenue   decimal.Decimal // Suma de subtotales de las líneas de factura
	TotalCOGS      decimal.Decimal // qty * costo_promedio_producto (products.cost)
	CommissionCost decimal.Decimal // GrossRevenue * commission_rate / 100
	TotalMargin    decimal.Decimal // GrossRevenue - TotalCOGS - CommissionCost
}

// SKUMarginResult resultado crudo de la consulta de márgenes por SKU.
type SKUMarginResult struct {
	ProductID    string
	SKU          string
	ProductName  string
	UnitsSold    decimal.Decimal
	GrossRevenue decimal.Decimal
	TotalCOGS    decimal.Decimal // qty * products.cost
	GrossProfit  decimal.Decimal // GrossRevenue - TotalCOGS
}

// AnalyticsRepository define las consultas de lectura para analítica de rentabilidad.
// Las implementaciones son read-only (no modifican datos).
type AnalyticsRepository interface {
	// GetSalesByChannel devuelve margen y métricas por canal de venta en el período dado.
	// Las facturas sin canal se agrupan bajo "Directo".
	GetSalesByChannel(
		ctx context.Context,
		companyID string,
		startDate, endDate time.Time,
	) ([]ChannelSalesResult, error)

	// GetSKUMargins devuelve los SKUs ordenados por rentabilidad bruta descendente.
	// limit controla cuántos SKUs devolver como máximo.
	GetSKUMargins(
		ctx context.Context,
		companyID string,
		startDate, endDate time.Time,
		limit int,
	) ([]SKUMarginResult, error)

	// ── Métodos del Dashboard ─────────────────────────────────────────────────

	// GetSalesMetrics devuelve los ingresos brutos (revenue) y el COGS total
	// de todas las facturas válidas de una empresa en el rango de fechas dado.
	// Usa COALESCE para devolver cero si no hay facturas en el período.
	GetSalesMetrics(
		ctx context.Context,
		companyID string,
		startDate, endDate time.Time,
	) (revenue, cost decimal.Decimal, err error)

	// GetTopSKUs devuelve los `limit` SKUs con mayor ingreso en el período,
	// incluyendo cantidad vendida y porcentaje de margen bruto.
	GetTopSKUs(
		ctx context.Context,
		companyID string,
		startDate, endDate time.Time,
		limit int,
	) ([]dto.TopSKUDTO, error)
}

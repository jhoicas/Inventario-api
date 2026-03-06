package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.AnalyticsRepository = (*AnalyticsRepo)(nil)

// AnalyticsRepo consultas de solo lectura para rentabilidad y analítica de canales.
type AnalyticsRepo struct {
	pool *pgxpool.Pool
}

// NewAnalyticsRepository construye el adaptador de analítica.
func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepo {
	return &AnalyticsRepo{pool: pool}
}

// GetSalesByChannel agrupa ingresos, COGS y margen por canal de venta.
// Fórmula del margen: GrossRevenue - TotalCOGS - CommissionCost - LogisticsCost - DiscountTotal.
// Las facturas sin canal se consolidan en el grupo "Directo".
// logistics_cost y discount_total se reparten por factura (una vez por invoice, no por línea).
func (r *AnalyticsRepo) GetSalesByChannel(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
) ([]repository.ChannelSalesResult, error) {
	const query = `
	SELECT
	    COALESCE(sc.id::TEXT, 'direct')                                                                   AS channel_id,
	    COALESCE(sc.name,    'Directo')                                                                   AS channel_name,
	    COALESCE(sc.channel_type, 'other')                                                                AS channel_type,
	    COALESCE(sc.commission_rate, 0)                                                                  AS commission_rate,
	    COUNT(DISTINCT i.id)                                                                              AS invoice_count,
	    SUM(d.quantity)                                                                                   AS units_sold,
	    SUM(d.subtotal)                                                                                   AS gross_revenue,
	    SUM(d.quantity * p.cost)                                                                          AS total_cogs,
	    SUM(d.subtotal * COALESCE(sc.commission_rate, 0) / 100)                                           AS commission_cost,
	    SUM(COALESCE(i.logistics_cost, 0) / NULLIF((SELECT COUNT(*) FROM invoice_details d2 WHERE d2.invoice_id = i.id), 0)) AS logistics_cost,
	    SUM(COALESCE(i.discount_total, 0) / NULLIF((SELECT COUNT(*) FROM invoice_details d2 WHERE d2.invoice_id = i.id), 0))  AS discount_total
	FROM invoices i
	JOIN invoice_details d ON d.invoice_id = i.id
	JOIN products       p  ON p.id         = d.product_id
	LEFT JOIN sales_channels sc ON sc.id   = i.channel_id
	WHERE i.company_id = $1
	  AND i.date BETWEEN $2 AND $3
	  AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION')
	GROUP BY sc.id, sc.name, sc.channel_type, sc.commission_rate
	ORDER BY SUM(d.subtotal) - SUM(d.quantity * p.cost) - SUM(d.subtotal * COALESCE(sc.commission_rate, 0) / 100)
	       - SUM(COALESCE(i.logistics_cost, 0) / NULLIF((SELECT COUNT(*) FROM invoice_details d2 WHERE d2.invoice_id = i.id), 0))
	       - SUM(COALESCE(i.discount_total, 0) / NULLIF((SELECT COUNT(*) FROM invoice_details d2 WHERE d2.invoice_id = i.id), 0)) DESC`

	rows, err := r.pool.Query(ctx, query, companyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetSalesByChannel: %w", err)
	}
	defer rows.Close()

	var results []repository.ChannelSalesResult
	for rows.Next() {
		var row repository.ChannelSalesResult
		if err := rows.Scan(
			&row.ChannelID,
			&row.ChannelName,
			&row.ChannelType,
			&row.CommissionRate,
			&row.InvoiceCount,
			&row.UnitsSold,
			&row.GrossRevenue,
			&row.TotalCOGS,
			&row.CommissionCost,
			&row.LogisticsCost,
			&row.DiscountTotal,
		); err != nil {
			return nil, fmt.Errorf("analytics.GetSalesByChannel scan: %w", err)
		}
		row.TotalMargin = row.GrossRevenue.Sub(row.TotalCOGS).Sub(row.CommissionCost).Sub(row.LogisticsCost).Sub(row.DiscountTotal)
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetSalesMetrics devuelve ingresos brutos y COGS total de las facturas válidas del período.
// Excluye facturas en estado DRAFT o ERROR_GENERATION.
// Usa COALESCE para devolver cero si no hay filas (período sin ventas).
func (r *AnalyticsRepo) GetSalesMetrics(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
) (revenue, cost decimal.Decimal, err error) {
	const query = `
	SELECT
	    COALESCE(SUM(d.subtotal),           0) AS revenue,
	    COALESCE(SUM(d.quantity * p.cost),  0) AS cost
	FROM invoices i
	JOIN invoice_details d ON d.invoice_id = i.id
	JOIN products        p ON p.id         = d.product_id
	WHERE i.company_id  = $1
	  AND i.date BETWEEN $2 AND $3
	  AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION', 'Error')`

	err = r.pool.QueryRow(ctx, query, companyID, startDate, endDate).
		Scan(&revenue, &cost)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("analytics.GetSalesMetrics: %w", err)
	}
	return revenue, cost, nil
}

// GetTopSKUs devuelve los `limit` productos con mayor ingreso en el período.
// El margen se calcula como (revenue - cogs) / revenue * 100, protegido contra división por cero.
func (r *AnalyticsRepo) GetTopSKUs(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
	limit int,
) ([]dto.TopSKUDTO, error) {
	const query = `
	SELECT
	    p.id                                        AS product_id,
	    p.sku,
	    p.name                                      AS product_name,
	    SUM(d.quantity)                             AS quantity_sold,
	    SUM(d.subtotal)                             AS total_revenue,
	    CASE
	        WHEN SUM(d.subtotal) > 0
	        THEN ROUND(
	            (SUM(d.subtotal) - SUM(d.quantity * p.cost))
	            / SUM(d.subtotal) * 100, 2)
	        ELSE 0
	    END                                         AS margin_percentage
	FROM invoice_details d
	JOIN invoices i ON i.id = d.invoice_id
	JOIN products p ON p.id = d.product_id
	WHERE i.company_id  = $1
	  AND i.date BETWEEN $2 AND $3
	  AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION', 'Error')
	GROUP BY p.id, p.sku, p.name
	ORDER BY total_revenue DESC
	LIMIT $4`

	rows, err := r.pool.Query(ctx, query, companyID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetTopSKUs: %w", err)
	}
	defer rows.Close()

	var results []dto.TopSKUDTO
	for rows.Next() {
		var item dto.TopSKUDTO
		if err := rows.Scan(
			&item.ProductID,
			&item.SKU,
			&item.ProductName,
			&item.QuantitySold,
			&item.TotalRevenue,
			&item.MarginPercentage,
		); err != nil {
			return nil, fmt.Errorf("analytics.GetTopSKUs scan: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("analytics.GetTopSKUs rows: %w", err)
	}
	if results == nil {
		results = []dto.TopSKUDTO{}
	}
	return results, nil
}

// GetSKUMargins devuelve rentabilidad bruta por SKU, ordenada de mayor a menor beneficio.
// No incluye comisión de canal (análisis de producto puro: precio - COGS).
func (r *AnalyticsRepo) GetSKUMargins(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
	limit int,
) ([]repository.SKUMarginResult, error) {
	const query = `
	SELECT
	    p.id                                          AS product_id,
	    p.sku,
	    p.name                                        AS product_name,
	    SUM(d.quantity)                               AS units_sold,
	    SUM(d.subtotal)                               AS gross_revenue,
	    SUM(d.quantity * p.cost)                      AS total_cogs,
	    SUM(d.subtotal - d.quantity * p.cost)         AS gross_profit
	FROM invoice_details d
	JOIN invoices i ON i.id  = d.invoice_id
	JOIN products p ON p.id  = d.product_id
	WHERE i.company_id = $1
	  AND i.date BETWEEN $2 AND $3
	  AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION')
	GROUP BY p.id, p.sku, p.name
	ORDER BY gross_profit DESC
	LIMIT $4`

	rows, err := r.pool.Query(ctx, query, companyID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetSKUMargins: %w", err)
	}
	defer rows.Close()

	var results []repository.SKUMarginResult
	for rows.Next() {
		var row repository.SKUMarginResult
		if err := rows.Scan(
			&row.ProductID,
			&row.SKU,
			&row.ProductName,
			&row.UnitsSold,
			&row.GrossRevenue,
			&row.TotalCOGS,
			&row.GrossProfit,
		); err != nil {
			return nil, fmt.Errorf("analytics.GetSKUMargins scan: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetRawMaterialImpactRanking devuelve ranking de materias primas por impacto financiero
// en los productos vendidos en el período (uso proyectado vía BOM y coste de materia prima).
func (r *AnalyticsRepo) GetRawMaterialImpactRanking(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
	limit int,
) ([]dto.RawMaterialImpactDTO, error) {
	const query = `
	WITH sales AS (
	    SELECT d.product_id, SUM(d.quantity) AS qty
	    FROM invoices i
	    JOIN invoice_details d ON d.invoice_id = i.id
	    WHERE i.company_id = $1
	      AND i.date BETWEEN $2 AND $3
	      AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION', 'Error')
	    GROUP BY d.product_id
	),
	material_usage AS (
	    SELECT
	        rm.id   AS raw_material_id,
	        rm.sku,
	        rm.name,
	        (s.qty * bom.quantity_required * (1 + bom.waste_percentage)) * rm.cost AS cost_impact
	    FROM sales s
	    JOIN bill_of_materials bom ON bom.product_id = s.product_id
	    JOIN raw_materials rm ON rm.id = bom.raw_material_id AND rm.company_id = $1
	),
	ranking AS (
	    SELECT
	        raw_material_id,
	        sku,
	        name,
	        SUM(cost_impact) AS total_cost_impact
	    FROM material_usage
	    GROUP BY raw_material_id, sku, name
	    ORDER BY total_cost_impact DESC
	    LIMIT $4
	)
	SELECT
	    r.raw_material_id,
	    r.sku,
	    r.name,
	    r.total_cost_impact,
	    (r.total_cost_impact / NULLIF(SUM(r.total_cost_impact) OVER (), 0)) * 100 AS usage_pct
	FROM ranking r`

	rows, err := r.pool.Query(ctx, query, companyID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetRawMaterialImpactRanking: %w", err)
	}
	defer rows.Close()

	var results []dto.RawMaterialImpactDTO
	for rows.Next() {
		var item dto.RawMaterialImpactDTO
		if err := rows.Scan(
			&item.RawMaterialID,
			&item.SKU,
			&item.Name,
			&item.TotalCostImpact,
			&item.UsagePct,
		); err != nil {
			return nil, fmt.Errorf("analytics.GetRawMaterialImpactRanking scan: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("analytics.GetRawMaterialImpactRanking rows: %w", err)
	}
	if results == nil {
		results = []dto.RawMaterialImpactDTO{}
	}
	return results, nil
}

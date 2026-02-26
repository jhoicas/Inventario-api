package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

const (
	defaultTopN    = 20
	maxTopN        = 200
	paretoThreshold = 80 // Principio de Pareto: el top 20% de SKUs genera el 80% de ingresos
)

var (
	hundred  = decimal.NewFromInt(100)
	pareto80 = decimal.NewFromInt(paretoThreshold)
)

// AnalyticsUseCase orquesta las consultas de rentabilidad y aplica las reglas de negocio:
//   - Cálculo de márgenes por canal.
//   - Ranking de SKUs por margen bruto.
//   - Identificación del top 20% Pareto (SKUs que generan ~80% del ingreso).
type AnalyticsUseCase struct {
	analyticsRepo repository.AnalyticsRepository
}

// NewAnalyticsUseCase construye el caso de uso.
func NewAnalyticsUseCase(analyticsRepo repository.AnalyticsRepository) *AnalyticsUseCase {
	return &AnalyticsUseCase{analyticsRepo: analyticsRepo}
}

// GetMarginsReport genera el reporte completo de márgenes para un período.
func (uc *AnalyticsUseCase) GetMarginsReport(
	ctx context.Context,
	companyID string,
	req dto.MarginsReportRequest,
) (*dto.MarginsReportDTO, error) {
	startDate, endDate, err := parsePeriod(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	topN := req.TopN
	if topN <= 0 {
		topN = defaultTopN
	}
	if topN > maxTopN {
		topN = maxTopN
	}

	// 1) Consultar canales y SKUs en paralelo (llamadas independientes)
	type channelResult struct {
		rows []repository.ChannelSalesResult
		err  error
	}
	type skuResult struct {
		rows []repository.SKUMarginResult
		err  error
	}

	chChan := make(chan channelResult, 1)
	skuChan := make(chan skuResult, 1)

	go func() {
		rows, err := uc.analyticsRepo.GetSalesByChannel(ctx, companyID, startDate, endDate)
		chChan <- channelResult{rows, err}
	}()
	go func() {
		rows, err := uc.analyticsRepo.GetSKUMargins(ctx, companyID, startDate, endDate, topN)
		skuChan <- skuResult{rows, err}
	}()

	chRes := <-chChan
	skuRes := <-skuChan

	if chRes.err != nil {
		return nil, fmt.Errorf("analytics: canales: %w", chRes.err)
	}
	if skuRes.err != nil {
		return nil, fmt.Errorf("analytics: SKUs: %w", skuRes.err)
	}

	// 2) Construir rentabilidad por canal
	profitability := buildChannelProfitability(chRes.rows)

	// 3) Construir ranking de SKUs con análisis Pareto
	skuRanking := buildSKURanking(skuRes.rows)

	// 4) Filtrar los SKUs que conforman el top 80% de ingresos (Pareto)
	var paretoSKUs []dto.SKURankingDTO
	for _, sku := range skuRanking {
		if sku.IsTopPareto {
			paretoSKUs = append(paretoSKUs, sku)
		}
	}

	return &dto.MarginsReportDTO{
		Period: dto.PeriodDTO{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
		},
		Profitability: profitability,
		SKURanking:    skuRanking,
		ParetoSKUs:    paretoSKUs,
	}, nil
}

// buildChannelProfitability convierte los resultados raw en DTO con totales globales y % de participación.
func buildChannelProfitability(rows []repository.ChannelSalesResult) dto.ChannelProfitabilityDTO {
	var totalRevenue, totalCOGS, totalMargin decimal.Decimal
	for _, r := range rows {
		totalRevenue = totalRevenue.Add(r.GrossRevenue)
		totalCOGS = totalCOGS.Add(r.TotalCOGS)
		totalMargin = totalMargin.Add(r.TotalMargin)
	}

	channels := make([]dto.MarginByChannelDTO, 0, len(rows))
	for _, r := range rows {
		marginPct := decimal.Zero
		if r.GrossRevenue.IsPositive() {
			marginPct = r.TotalMargin.Div(r.GrossRevenue).Mul(hundred).Round(2)
		}
		revenuePct := decimal.Zero
		if totalRevenue.IsPositive() {
			revenuePct = r.GrossRevenue.Div(totalRevenue).Mul(hundred).Round(2)
		}
		channels = append(channels, dto.MarginByChannelDTO{
			ChannelID:      r.ChannelID,
			ChannelName:    r.ChannelName,
			ChannelType:    r.ChannelType,
			CommissionRate: r.CommissionRate,
			InvoiceCount:   r.InvoiceCount,
			UnitsSold:      r.UnitsSold,
			GrossRevenue:   r.GrossRevenue.Round(2),
			TotalCOGS:      r.TotalCOGS.Round(2),
			CommissionCost: r.CommissionCost.Round(2),
			TotalMargin:    r.TotalMargin.Round(2),
			MarginPct:      marginPct,
			RevenuePct:     revenuePct,
		})
	}

	overallMarginPct := decimal.Zero
	if totalRevenue.IsPositive() {
		overallMarginPct = totalMargin.Div(totalRevenue).Mul(hundred).Round(2)
	}

	return dto.ChannelProfitabilityDTO{
		TotalRevenue:     totalRevenue.Round(2),
		TotalCOGS:        totalCOGS.Round(2),
		TotalMargin:      totalMargin.Round(2),
		OverallMarginPct: overallMarginPct,
		Channels:         channels,
	}
}

// buildSKURanking convierte los SKUs en DTOs enriquecidos con:
//   - Rank (posición ordinal por margen bruto descendente).
//   - MarginPct y RevenuePct por SKU.
//   - CumulativeRevenuePct acumulado (para curva Pareto).
//   - IsTopPareto: true si este SKU cae dentro del primer 80% de ingresos acumulados.
func buildSKURanking(rows []repository.SKUMarginResult) []dto.SKURankingDTO {
	if len(rows) == 0 {
		return []dto.SKURankingDTO{}
	}

	// Calcular totales globales
	var totalRevenue decimal.Decimal
	for _, r := range rows {
		totalRevenue = totalRevenue.Add(r.GrossRevenue)
	}

	ranking := make([]dto.SKURankingDTO, 0, len(rows))
	var cumulative decimal.Decimal

	for i, r := range rows {
		marginPct := decimal.Zero
		if r.GrossRevenue.IsPositive() {
			marginPct = r.GrossProfit.Div(r.GrossRevenue).Mul(hundred).Round(2)
		}
		revenuePct := decimal.Zero
		if totalRevenue.IsPositive() {
			revenuePct = r.GrossRevenue.Div(totalRevenue).Mul(hundred).Round(2)
		}

		cumulative = cumulative.Add(revenuePct)
		// IsTopPareto: verdadero mientras el acumulado no haya superado el umbral del 80%
		// Incluimos el SKU que cruza el umbral (el principio del 80/20 es aproximado)
		isPareto := cumulative.LessThanOrEqual(pareto80) || (i == 0)

		ranking = append(ranking, dto.SKURankingDTO{
			Rank:             i + 1,
			ProductID:        r.ProductID,
			SKU:              r.SKU,
			ProductName:      r.ProductName,
			UnitsSold:        r.UnitsSold,
			GrossRevenue:     r.GrossRevenue.Round(2),
			TotalCOGS:        r.TotalCOGS.Round(2),
			GrossProfit:      r.GrossProfit.Round(2),
			MarginPct:        marginPct,
			RevenuePct:       revenuePct,
			CumulativeRevPct: cumulative.Round(2),
			IsTopPareto:      isPareto,
		})
	}
	return ranking
}

// parsePeriod convierte los strings de fecha en time.Time; aplica valores por defecto si están vacíos.
func parsePeriod(startStr, endStr string) (start, end time.Time, err error) {
	now := time.Now()

	if endStr == "" {
		end = now
	} else {
		end, err = time.ParseInLocation("2006-01-02", endStr, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("end_date inválido: %w", err)
		}
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // inclusive hasta el final del día
	}

	if startStr == "" {
		// Primer día del mes actual
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		start, err = time.ParseInLocation("2006-01-02", startStr, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("start_date inválido: %w", err)
		}
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_date no puede ser posterior a end_date")
	}
	return start, end, nil
}

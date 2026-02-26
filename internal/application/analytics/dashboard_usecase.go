// Package analytics contiene los casos de uso para reportes de negocio y el
// Dashboard de Analítica Financiera.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

const dashboardTopSKUs = 5 // número de SKUs en el widget del dashboard

// DashboardUseCase genera el resumen financiero del día y del mes en curso.
//
// Fuente de datos: AnalyticsRepository (consultas read-only).
// No accede directamente a la tabla de facturas; delega todo en el repositorio.
type DashboardUseCase struct {
	analyticsRepo repository.AnalyticsRepository
}

// NewDashboardUseCase construye el caso de uso.
func NewDashboardUseCase(analyticsRepo repository.AnalyticsRepository) *DashboardUseCase {
	return &DashboardUseCase{analyticsRepo: analyticsRepo}
}

// GetSummary construye el DashboardSummaryDTO para la empresa indicada.
//
// Tres llamadas en paralelo:
//  1. GetSalesMetrics(hoy)    → TodaySales + TodayMargin
//  2. GetSalesMetrics(mes)    → MonthlySales + MonthlyMargin
//  3. GetTopSKUs(mes, top 5)  → TopSKUs
func (uc *DashboardUseCase) GetSummary(
	ctx context.Context,
	companyID string,
) (*dto.DashboardSummaryDTO, error) {
	now := time.Now()

	// ── Rangos de fecha ────────────────────────────────────────────────────────
	// Hoy: 00:00:00.000 – 23:59:59.999
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24*time.Hour - time.Nanosecond)

	// Mes en curso: día 1 a las 00:00 – hoy a las 23:59:59
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := todayEnd

	// ── Goroutines para paralelizar las 3 consultas DB ────────────────────────
	type metricsResult struct {
		revenue decimal.Decimal
		cost    decimal.Decimal
		err     error
	}
	type topSKUsResult struct {
		skus []dto.TopSKUDTO
		err  error
	}

	todayCh := make(chan metricsResult, 1)
	monthCh := make(chan metricsResult, 1)
	skusCh := make(chan topSKUsResult, 1)

	go func() {
		rev, cost, err := uc.analyticsRepo.GetSalesMetrics(ctx, companyID, todayStart, todayEnd)
		todayCh <- metricsResult{rev, cost, err}
	}()
	go func() {
		rev, cost, err := uc.analyticsRepo.GetSalesMetrics(ctx, companyID, monthStart, monthEnd)
		monthCh <- metricsResult{rev, cost, err}
	}()
	go func() {
		skus, err := uc.analyticsRepo.GetTopSKUs(ctx, companyID, monthStart, monthEnd, dashboardTopSKUs)
		skusCh <- topSKUsResult{skus, err}
	}()

	today := <-todayCh
	month := <-monthCh
	skus := <-skusCh

	if today.err != nil {
		return nil, fmt.Errorf("dashboard: métricas de hoy: %w", today.err)
	}
	if month.err != nil {
		return nil, fmt.Errorf("dashboard: métricas del mes: %w", month.err)
	}
	if skus.err != nil {
		return nil, fmt.Errorf("dashboard: top SKUs: %w", skus.err)
	}

	// ── Calcular márgenes ──────────────────────────────────────────────────────
	todayMargin := today.revenue.Sub(today.cost).Round(2)
	monthMargin := month.revenue.Sub(month.cost).Round(2)

	// ── Construir DTO ──────────────────────────────────────────────────────────
	return &dto.DashboardSummaryDTO{
		TodaySales:    today.revenue.Round(2),
		TodayMargin:   todayMargin,
		MonthlySales:  month.revenue.Round(2),
		MonthlyMargin: monthMargin,
		TopSKUs:       skus.skus,
		DateLabel:     monthLabel(now),
	}, nil
}

// monthLabel devuelve una etiqueta legible del mes, ej: "Febrero 2026".
func monthLabel(t time.Time) string {
	months := [...]string{
		"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
		"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
	}
	return fmt.Sprintf("%s %d", months[t.Month()-1], t.Year())
}

package crm

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// AnalyticsUseCase expone las estadísticas CRM de una empresa.
type AnalyticsUseCase struct {
	profileRepo repository.CRMProfileRepository
}

// NewAnalyticsUseCase construye el caso de uso de analytics CRM.
func NewAnalyticsUseCase(profileRepo repository.CRMProfileRepository) *AnalyticsUseCase {
	return &AnalyticsUseCase{profileRepo: profileRepo}
}

// GetAnalytics retorna el dashboard CRM filtrado por companyID.
func (uc *AnalyticsUseCase) GetAnalytics(ctx context.Context, companyID string) (*dto.CRMAnalyticsResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	if uc.profileRepo == nil {
		return nil, domain.ErrConflict
	}
	out, err := uc.profileRepo.GetAnalytics(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out.EvolucionMensual = rebuildEvolutionVariation(out.EvolucionMensual)
	return out, nil
}

// GetRemarketingProspects retorna prospectos ideales para campañas de remarketing.
func (uc *AnalyticsUseCase) GetRemarketingProspects(ctx context.Context, companyID string) ([]dto.RemarketingProspect, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	if uc.profileRepo == nil {
		return nil, domain.ErrConflict
	}

	prospects, err := uc.profileRepo.GetRemarketingProspects(ctx, companyID)
	if err != nil {
		return nil, err
	}

	for i := range prospects {
		prospects[i].MensajeSugerido = buildRemarketingMessage(prospects[i].Segmento, prospects[i].Categoria)
	}

	return prospects, nil
}

// GetAnalyticsSegmentation retorna segmentación CRM con ventas y ticket promedio calculados.
func (uc *AnalyticsUseCase) GetAnalyticsSegmentation(ctx context.Context, companyID string) ([]dto.CRMAnalyticsSegmentationItem, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	if uc.profileRepo == nil {
		return nil, domain.ErrConflict
	}

	items, err := uc.profileRepo.GetDashboardSegmentation(companyID)
	if err != nil {
		return nil, err
	}

	out := make([]dto.CRMAnalyticsSegmentationItem, 0, len(items))
	for _, item := range items {
		totalSales := item.TotalSales.InexactFloat64()
		ticket := 0.0
		if item.Count > 0 {
			ticket = totalSales / float64(item.Count)
		}
		out = append(out, dto.CRMAnalyticsSegmentationItem{
			Category:       item.Category,
			Count:          item.Count,
			VentasTotales:  totalSales,
			TicketPromedio: ticket,
		})
	}

	_ = ctx
	return out, nil
}

func buildRemarketingMessage(segmento, categoria string) string {
	seg := strings.ToUpper(strings.TrimSpace(segmento))
	cat := strings.TrimSpace(categoria)
	if cat == "" {
		cat = "nuestra selección"
	}

	switch seg {
	case "VIP":
		return fmt.Sprintf("Acceso exclusivo: nuevos productos de %s. ¡Gracias por tu fidelidad!", cat)
	case "PREMIUM":
		return fmt.Sprintf("Descubre novedades premium de %s y aprovecha beneficios especiales.", cat)
	case "RECURRENTE":
		return fmt.Sprintf("Tenemos nuevas opciones para ti en %s. ¡Vuelve pronto!", cat)
	case "OCASIONAL":
		return fmt.Sprintf("¡Te extrañamos! Vuelve y descubre novedades en %s.", cat)
	default:
		return fmt.Sprintf("Conoce nuestras novedades en %s.", cat)
	}
}

func rebuildEvolutionVariation(items []dto.CRMAnalyticsEvolutionItem) []dto.CRMAnalyticsEvolutionItem {
	if len(items) == 0 {
		return items
	}
	out := make([]dto.CRMAnalyticsEvolutionItem, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		return parseAnalyticsMonth(out[i].Mes).Before(parseAnalyticsMonth(out[j].Mes))
	})
	for i := range out {
		if i == 0 {
			out[i].Variacion = "-"
			continue
		}
		out[i].Variacion = formatAnalyticsVariation(out[i].Ventas, out[i-1].Ventas)
	}
	return out
}

func parseAnalyticsMonth(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse("01/2006", value); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01", value); err == nil {
		return t
	}
	return time.Time{}
}

func formatAnalyticsVariation(current, previous float64) string {
	if previous <= 0 {
		if current <= 0 {
			return "-"
		}
		return "+100.00%"
	}
	variation := ((current - previous) / previous) * 100
	if math.IsNaN(variation) || math.IsInf(variation, 0) {
		return "-"
	}
	if variation >= 0 {
		return fmt.Sprintf("+%.2f%%", variation)
	}
	return fmt.Sprintf("%.2f%%", variation)
}

package crm

import (
	"context"
	"fmt"
	"strings"

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
	return uc.profileRepo.GetAnalytics(ctx, companyID)
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

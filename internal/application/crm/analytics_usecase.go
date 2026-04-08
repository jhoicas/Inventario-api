package crm

import (
	"context"

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

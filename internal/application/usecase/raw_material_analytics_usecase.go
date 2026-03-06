package usecase

import (
	"context"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// RawMaterialAnalyticsUseCase expone consultas de analítica sobre materias primas.
type RawMaterialAnalyticsUseCase struct {
	analyticsRepo repository.AnalyticsRepository
}

// NewRawMaterialAnalyticsUseCase construye el caso de uso.
func NewRawMaterialAnalyticsUseCase(analyticsRepo repository.AnalyticsRepository) *RawMaterialAnalyticsUseCase {
	return &RawMaterialAnalyticsUseCase{analyticsRepo: analyticsRepo}
}

// GetRawMaterialImpactRanking delega en el repositorio de analítica para obtener
// el ranking de materias primas por impacto financiero en SKUs top Pareto.
func (uc *RawMaterialAnalyticsUseCase) GetRawMaterialImpactRanking(
	ctx context.Context,
	companyID string,
	startDate, endDate time.Time,
	limit int,
) ([]dto.RawMaterialImpactDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	return uc.analyticsRepo.GetRawMaterialImpactRanking(ctx, companyID, startDate, endDate, limit)
}


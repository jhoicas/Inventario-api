package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// DIANSettingsRepository define el puerto para persistir configuración DIAN.
type DIANSettingsRepository interface {
	Upsert(ctx context.Context, settings *entity.DIANSettings) error
	// GetByCompanyID devuelve la configuración más reciente de la empresa (cualquier ambiente).
	GetByCompanyID(ctx context.Context, companyID string) (*entity.DIANSettings, error)
	// GetByCompanyIDAndEnvironment devuelve la configuración del ambiente indicado ("test" o "prod").
	GetByCompanyIDAndEnvironment(ctx context.Context, companyID, environment string) (*entity.DIANSettings, error)
}

package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// DIANSettingsRepository define el puerto para persistir configuración DIAN.
type DIANSettingsRepository interface {
	Upsert(ctx context.Context, settings *entity.DIANSettings) error
	GetByCompanyID(ctx context.Context, companyID string) (*entity.DIANSettings, error)
}

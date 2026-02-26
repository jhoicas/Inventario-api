package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// BillingResolutionRepository define el puerto de persistencia para resoluciones DIAN.
type BillingResolutionRepository interface {
	Create(ctx context.Context, res *entity.BillingResolution) error
	GetByID(ctx context.Context, id string) (*entity.BillingResolution, error)

	// GetActiveByCompanyAndPrefix devuelve la resolución activa para la empresa y prefijo dados.
	// Es la consulta crítica antes de construir el XML DIAN: sin resolución activa no se puede
	// incluir DianExtensions y la factura sería rechazada.
	GetActiveByCompanyAndPrefix(ctx context.Context, companyID, prefix string) (*entity.BillingResolution, error)

	// ListByCompany lista todas las resoluciones de una empresa (activas e inactivas).
	ListByCompany(ctx context.Context, companyID string) ([]*entity.BillingResolution, error)

	Update(ctx context.Context, res *entity.BillingResolution) error
}

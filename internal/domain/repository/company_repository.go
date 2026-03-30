package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CompanyRepository define el puerto de persistencia para Company (DIP).
// La implementación vive en infrastructure.
type CompanyRepository interface {
	Create(company *entity.Company) error
	GetByID(id string) (*entity.Company, error)
	GetByNIT(nit string) (*entity.Company, error)
	Update(company *entity.Company) error
	List(limit, offset int) ([]*entity.Company, error)
	Delete(id string) error

	// HasActiveModule informa si la empresa tiene el módulo activo y no vencido.
	HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error)

	// ListModules devuelve los módulos contratados por la empresa.
	ListModules(ctx context.Context, companyID string) ([]*entity.CompanyModule, error)

	// GetModule devuelve un módulo específico de la empresa.
	GetModule(ctx context.Context, companyID, moduleName string) (*entity.CompanyModule, error)

	// UpsertModule crea o actualiza el módulo de la empresa.
	UpsertModule(ctx context.Context, module *entity.CompanyModule) error

	// DeleteModule elimina un módulo asignado a la empresa.
	DeleteModule(ctx context.Context, companyID, moduleName string) error
}

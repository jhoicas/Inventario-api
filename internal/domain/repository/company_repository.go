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
}

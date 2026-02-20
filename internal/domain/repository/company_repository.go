package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

// CompanyRepository define el puerto de persistencia para Company (DIP).
// La implementación vive en infrastructure.
type CompanyRepository interface {
	Create(company *entity.Company) error
	GetByID(id string) (*entity.Company, error)
	GetByNIT(nit string) (*entity.Company, error)
	Update(company *entity.Company) error
	List(limit, offset int) ([]*entity.Company, error)
	Delete(id string) error
}

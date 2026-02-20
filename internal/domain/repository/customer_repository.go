package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

// CustomerRepository define el puerto de persistencia para Customer (facturación).
type CustomerRepository interface {
	Create(customer *entity.Customer) error
	GetByID(id string) (*entity.Customer, error)
	GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error)
	ListByCompany(companyID string, limit, offset int) ([]*entity.Customer, error)
	Update(customer *entity.Customer) error
	Delete(id string) error
}

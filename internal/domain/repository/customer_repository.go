package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CustomerRepository define el puerto de persistencia para Customer (facturación).
type CustomerRepository interface {
	Create(customer *entity.Customer) error
	GetByID(id string) (*entity.Customer, error)
	GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error)
	// ListByCompany lista clientes por empresa. Si search no es vacío, filtra por nombre o NIT (tax_id).
	ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, error)
	Update(customer *entity.Customer) error
	Delete(id string) error
	SetActive(companyID, id string, isActive bool) error
}

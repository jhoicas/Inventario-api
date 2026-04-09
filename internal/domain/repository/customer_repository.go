package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CustomerListFilters define filtros avanzados para listado de clientes.
type CustomerListFilters struct {
	Search             string
	CategoryID         string
	CategoryNameLegacy string
	WithoutCategory    bool
}

// CustomerRepository define el puerto de persistencia para Customer (facturación).
type CustomerRepository interface {
	Create(customer *entity.Customer) error
	GetByID(id string) (*entity.Customer, error)
	GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error)
	GetByCompanyAndEmail(companyID, email string) (*entity.Customer, error)
	// ListByCompany lista clientes por empresa. Si search no es vacío, filtra por nombre o NIT (tax_id).
	ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, int64, error)
	Update(customer *entity.Customer) error
	Delete(id string) error
	SetActive(companyID, id string, isActive bool) error
}

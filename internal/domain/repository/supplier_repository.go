package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// SupplierRepository define el puerto de persistencia para proveedores (DIP).
type SupplierRepository interface {
	Create(supplier *entity.Supplier) error
	GetByID(id string) (*entity.Supplier, error)
	GetByCompanyAndNIT(companyID, nit string) (*entity.Supplier, error)
	Update(supplier *entity.Supplier) error
	ListByCompany(companyID, search string, limit, offset int) ([]*entity.Supplier, error)
	SetActive(companyID, id string, isActive bool) error
}

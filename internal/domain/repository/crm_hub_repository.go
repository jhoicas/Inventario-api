package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CrmHubProductRepository persiste el catálogo CRM en crm_products_hub (paralelo a products de inventario).
type CrmHubProductRepository interface {
	Create(p *entity.CrmProductHub) error
	GetByID(id string) (*entity.CrmProductHub, error)
	GetByCompanyAndProductCode(companyID, productCode string) (*entity.CrmProductHub, error)
	Update(p *entity.CrmProductHub) error
	// ListByCompany lista con paginación. Si active es nil, no filtra por is_active.
	ListByCompany(companyID string, limit, offset int, active *bool) ([]*entity.CrmProductHub, int64, error)
	Deactivate(companyID, id string) error
}

package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CategoryRepository persiste categorías de producto en crm_category_product_hub.
type CategoryRepository interface {
	Create(category *entity.CrmCategoryProductHub) error
	GetByID(id string) (*entity.CrmCategoryProductHub, error)
	// GetByCompanyAndName detecta duplicados por empresa+nombre (misma semántica que el índice único del hub).
	GetByCompanyAndName(companyID, name string) (*entity.CrmCategoryProductHub, error)
	Update(category *entity.CrmCategoryProductHub) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.CrmCategoryProductHub, int64, error)
	// Deactivate marca la categoría como inactiva (is_active = false).
	Deactivate(companyID, id string) error
}

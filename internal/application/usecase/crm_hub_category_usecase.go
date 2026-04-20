package usecase

// CrmHubCategoryUseCase expone el catálogo hub de categorías bajo rutas CRM (/api/crm/categories-hub).
// Reutiliza la lógica de CategoryUseCase (misma tabla crm_category_product_hub).
type CrmHubCategoryUseCase struct {
	*CategoryUseCase
}

// NewCrmHubCategoryUseCase envuelve el caso de uso de categorías inventario/hub.
func NewCrmHubCategoryUseCase(inner *CategoryUseCase) *CrmHubCategoryUseCase {
	return &CrmHubCategoryUseCase{CategoryUseCase: inner}
}

// Delete elimina la categoría en BD (DELETE); productos hub quedan con category_id NULL por FK.
func (c *CrmHubCategoryUseCase) Delete(companyID, id string) error {
	return c.CategoryUseCase.Deactivate(companyID, id)
}

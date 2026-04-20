package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CrmHubCategoryHandler HTTP para crm_category_product_hub bajo /api/crm/categories-hub.
type CrmHubCategoryHandler struct {
	uc *usecase.CrmHubCategoryUseCase
}

// NewCrmHubCategoryHandler constructor.
func NewCrmHubCategoryHandler(uc *usecase.CrmHubCategoryUseCase) *CrmHubCategoryHandler {
	return &CrmHubCategoryHandler{uc: uc}
}

func mapProductCategoryToHub(in *dto.ProductCategoryResponse) *dto.CrmHubCategoryResponse {
	if in == nil {
		return nil
	}
	return &dto.CrmHubCategoryResponse{
		ID: in.ID, CompanyID: in.CompanyID, Name: in.Name,
		CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt,
	}
}

func mapProductCategoryListToHub(in *dto.ProductCategoryListResponse) *dto.CrmHubCategoryListResponse {
	if in == nil {
		return nil
	}
	items := make([]dto.CrmHubCategoryResponse, len(in.Items))
	for i := range in.Items {
		items[i] = *mapProductCategoryToHub(&in.Items[i])
	}
	return &dto.CrmHubCategoryListResponse{
		Items: items, Total: in.Total, Limit: in.Limit, Offset: in.Offset,
	}
}

// Create POST /api/crm/categories-hub
func (h *CrmHubCategoryHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	var body dto.CrmHubCategoryCreateRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	in := dto.CreateProductCategoryRequest{Name: body.Name, Code: body.Code}
	out, err := h.uc.Create(companyID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "nombre es obligatorio"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "ya existe una categoría con ese nombre"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(mapProductCategoryToHub(out))
}

// List GET
func (h *CrmHubCategoryHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	out, err := h.uc.List(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return respondPaginated(c, "crm.categories_hub.list", mapProductCategoryListToHub(out))
}

// GetByID GET /:id
func (h *CrmHubCategoryHandler) GetByID(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.GetByID(companyID, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
	}
	return c.JSON(mapProductCategoryToHub(out))
}

// Update PUT /:id
func (h *CrmHubCategoryHandler) Update(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	var body dto.CrmHubCategoryUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	in := dto.UpdateProductCategoryRequest{Name: body.Name}
	out, err := h.uc.Update(companyID, id, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "ya existe una categoría con ese nombre"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
	}
	return c.JSON(mapProductCategoryToHub(out))
}

// Delete DELETE /:id
func (h *CrmHubCategoryHandler) Delete(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	if err := h.uc.Delete(companyID, id); err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// CrmHubProductHandler HTTP para crm_products_hub bajo /api/crm/products-hub.
type CrmHubProductHandler struct {
	uc *usecase.CrmHubProductUseCase
}

// NewCrmHubProductHandler constructor.
func NewCrmHubProductHandler(uc *usecase.CrmHubProductUseCase) *CrmHubProductHandler {
	return &CrmHubProductHandler{uc: uc}
}

func parseOptionalBool(s string) *bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil
	}
	if s == "true" || s == "1" {
		t := true
		return &t
	}
	if s == "false" || s == "0" {
		f := false
		return &f
	}
	return nil
}

// Create POST
func (h *CrmHubProductHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	var in dto.CrmHubProductCreateRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.Create(companyID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "código de producto ya existe"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// List GET
func (h *CrmHubProductHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	active := parseOptionalBool(c.Query("is_active"))
	out, err := h.uc.List(companyID, limit, offset, active)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return respondPaginated(c, "crm.products_hub.list", out)
}

// GetByID GET /:id
func (h *CrmHubProductHandler) GetByID(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.GetByID(companyID, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
	}
	return c.JSON(out)
}

// Update PUT /:id
func (h *CrmHubProductHandler) Update(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	var in dto.CrmHubProductUpdateRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.Update(companyID, id, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "código de producto ya existe"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
	}
	return c.JSON(out)
}

// Deactivate PATCH /:id/deactivate
func (h *CrmHubProductHandler) Deactivate(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	if err := h.uc.Deactivate(companyID, id); err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

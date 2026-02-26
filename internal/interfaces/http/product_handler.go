package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ProductHandler maneja las peticiones HTTP para Product (protegido).
type ProductHandler struct {
	uc *usecase.ProductUseCase
}

// NewProductHandler construye el handler.
func NewProductHandler(uc *usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{uc: uc}
}

// Create godoc
// @Summary      Crear producto
// @Tags         products
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateProductRequest  true  "Datos del producto"
// @Success      201   {object}  dto.ProductResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/products [post]
func (h *ProductHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	var in dto.CreateProductRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.SKU == "" || in.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "sku y name son requeridos"})
	}
	if in.UnitMeasure == "" {
		in.UnitMeasure = "94"
	}
	out, err := h.uc.Create(companyID, in)
	if err != nil {
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "SKU ya existe en esta empresa"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetByID godoc
// @Summary      Obtener producto por ID
// @Tags         products
// @Security     Bearer
// @Produce      json
// @Param        id   path  string  true  "ID del producto"
// @Success      200  {object}  dto.ProductResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/products/{id} [get]
func (h *ProductHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
	}
	return c.JSON(out)
}

// List godoc
// @Summary      Listar productos
// @Tags         products
// @Security     Bearer
// @Produce      json
// @Param        limit   query  int  false  "Límite"   default(20)
// @Param        offset  query  int  false  "Offset"   default(0)
// @Success      200     {object}  dto.ProductListResponse
// @Router       /api/products [get]
func (h *ProductHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	out, err := h.uc.List(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// Update godoc
// @Summary      Actualizar producto
// @Tags         products
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "ID del producto"
// @Param        body  body  dto.UpdateProductRequest  true  "Datos a actualizar"
// @Success      200   {object}  dto.ProductResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /api/products/{id} [put]
func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	var in dto.UpdateProductRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.Update(id, in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
	}
	return c.JSON(out)
}

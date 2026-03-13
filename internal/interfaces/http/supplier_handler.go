package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// SupplierUseCase interface para permitir mocking en tests.
type SupplierUseCase interface {
	Create(companyID string, in dto.CreateSupplierRequest) (*dto.SupplierResponse, error)
	GetByID(id string) (*dto.SupplierResponse, error)
	List(companyID string, filters dto.SupplierFilters) (*dto.SupplierListResponse, error)
	Update(id string, in dto.UpdateSupplierRequest) (*dto.SupplierResponse, error)
}

// SupplierHandler maneja las peticiones HTTP para Supplier (protegido).
type SupplierHandler struct {
	uc SupplierUseCase
}

// NewSupplierHandler construye el handler.
func NewSupplierHandler(uc SupplierUseCase) *SupplierHandler {
	return &SupplierHandler{uc: uc}
}

// Create godoc
// @Summary      Crear proveedor
// @Tags         suppliers
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateSupplierRequest  true  "Datos del proveedor"
// @Success      201   {object}  dto.SupplierResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/suppliers [post]
func (h *SupplierHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}

	var in dto.CreateSupplierRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	if in.Name == "" || in.NIT == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name y nit son requeridos"})
	}

	out, err := h.uc.Create(companyID, in)
	if err != nil {
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "NIT ya existe en esta empresa"})
		}
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetByID godoc
// @Summary      Obtener proveedor por ID
// @Tags         suppliers
// @Security     Bearer
// @Produce      json
// @Param        id   path  string  true  "ID del proveedor"
// @Success      200  {object}  dto.SupplierResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/suppliers/{id} [get]
func (h *SupplierHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}

	out, err := h.uc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "proveedor no encontrado"})
	}

	return c.JSON(out)
}

// List godoc
// @Summary      Listar proveedores
// @Tags         suppliers
// @Security     Bearer
// @Produce      json
// @Param        search  query  string  false  "Buscar por nombre o NIT"
// @Param        limit   query  int     false  "Límite" default(20)
// @Param        offset  query  int     false  "Offset" default(0)
// @Success      200     {object}  dto.SupplierListResponse
// @Router       /api/suppliers [get]
func (h *SupplierHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}

	filters := dto.SupplierFilters{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 20),
		Offset: c.QueryInt("offset", 0),
	}

	out, err := h.uc.List(companyID, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(out)
}

// Update godoc
// @Summary      Actualizar proveedor
// @Tags         suppliers
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "ID del proveedor"
// @Param        body  body  dto.UpdateSupplierRequest  true  "Datos a actualizar"
// @Success      200   {object}  dto.SupplierResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/suppliers/{id} [put]
func (h *SupplierHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}

	var in dto.UpdateSupplierRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	out, err := h.uc.Update(id, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "NIT ya existe en esta empresa"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "proveedor no encontrado"})
	}

	return c.JSON(out)
}

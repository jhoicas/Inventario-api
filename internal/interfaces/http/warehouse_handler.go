package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
)

// WarehouseHandler maneja las peticiones HTTP para Warehouse (protegido).
type WarehouseHandler struct {
	uc *usecase.WarehouseUseCase
}

// NewWarehouseHandler construye el handler.
func NewWarehouseHandler(uc *usecase.WarehouseUseCase) *WarehouseHandler {
	return &WarehouseHandler{uc: uc}
}

// Create godoc
// @Summary      Crear bodega
// @Tags         warehouses
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateWarehouseRequest  true  "Datos de la bodega"
// @Success      201   {object}  dto.WarehouseResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/warehouses [post]
func (h *WarehouseHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "company_id requerido"})
	}
	var in dto.CreateWarehouseRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name es requerido"})
	}
	out, err := h.uc.Create(companyID, in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetByID godoc
// @Summary      Obtener bodega por ID
// @Tags         warehouses
// @Security     Bearer
// @Produce      json
// @Param        id   path  string  true  "ID de la bodega"
// @Success      200  {object}  dto.WarehouseResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/warehouses/{id} [get]
func (h *WarehouseHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "bodega no encontrada"})
	}
	return c.JSON(out)
}

// List godoc
// @Summary      Listar bodegas
// @Tags         warehouses
// @Security     Bearer
// @Produce      json
// @Param        limit   query  int  false  "Límite"   default(20)
// @Param        offset  query  int  false  "Offset"   default(0)
// @Success      200     {object}  dto.WarehouseListResponse
// @Router       /api/warehouses [get]
func (h *WarehouseHandler) List(c *fiber.Ctx) error {
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

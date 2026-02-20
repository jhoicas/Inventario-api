package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/usecase"
	"github.com/tu-usuario/inventory-pro/internal/domain"
)

// CompanyHandler maneja las peticiones HTTP para el recurso Company.
type CompanyHandler struct {
	uc *usecase.CompanyUseCase
}

// NewCompanyHandler construye el handler inyectando el caso de uso.
func NewCompanyHandler(uc *usecase.CompanyUseCase) *CompanyHandler {
	return &CompanyHandler{uc: uc}
}

// Create godoc
// @Summary      Crear empresa
// @Tags         companies
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateCompanyRequest  true  "Datos de la empresa"
// @Success      201   {object}  dto.CompanyResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/companies [post]
func (h *CompanyHandler) Create(c *fiber.Ctx) error {
	var in dto.CreateCompanyRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.Name == "" || in.NIT == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name y nit son requeridos"})
	}
	out, err := h.uc.Create(in)
	if err != nil {
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "empresa con ese NIT ya existe"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetByID godoc
// @Summary      Obtener empresa por ID
// @Tags         companies
// @Produce      json
// @Param        id   path  string  true  "ID de la empresa"
// @Success      200  {object}  dto.CompanyResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/companies/{id} [get]
func (h *CompanyHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa no encontrada"})
	}
	return c.JSON(out)
}

// List godoc
// @Summary      Listar empresas
// @Tags         companies
// @Produce      json
// @Param        limit   query  int  false  "Límite"   default(20)
// @Param        offset  query  int  false  "Offset"   default(0)
// @Success      200     {object}  dto.CompanyListResponse
// @Router       /api/companies [get]
func (h *CompanyHandler) List(c *fiber.Ctx) error {
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
	out, err := h.uc.List(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

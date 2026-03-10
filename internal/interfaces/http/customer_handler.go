package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CustomerUseCase interfaz local para permitir mocking en tests.
type CustomerUseCase interface {
	Create(companyID string, in dto.CreateCustomerRequest) (*dto.CustomerResponse, error)
	List(companyID string, search string, limit, offset int) ([]*dto.CustomerResponse, error)
}

// CustomerHandler maneja las peticiones HTTP de clientes (facturación, protegido).
type CustomerHandler struct {
	uc CustomerUseCase
}

// NewCustomerHandler construye el handler.
func NewCustomerHandler(uc CustomerUseCase) *CustomerHandler {
	return &CustomerHandler{uc: uc}
}

// Create godoc
// @Summary      Crear cliente
// @Description  Crea un cliente asociado a la empresa autenticada para uso en facturación
// @Tags         customers
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateCustomerRequest  true  "Datos del cliente"
// @Success      201   {object}  dto.CustomerResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/customers [post]
func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateCustomerRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	customer, err := h.uc.Create(companyID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name y tax_id son requeridos"})
		}
		if err == domain.ErrDuplicate {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "ya existe un cliente con ese NIT/cédula"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(customer)
}

// List godoc
// @Summary      Listar clientes
// @Description  Lista los clientes de la empresa autenticada con paginación
// @Tags         customers
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        search  query     string  false  "Buscar por nombre o NIT (tax_id)"
// @Param        limit   query     int   false  "Límite de resultados"
// @Param        offset  query     int   false  "Desplazamiento"
// @Success      200     {array}   dto.CustomerResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/customers [get]
func (h *CustomerHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	search := c.Query("search")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	list, err := h.uc.List(companyID, search, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(list)
}

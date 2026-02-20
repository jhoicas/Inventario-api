package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/billing"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
)

// CustomerHandler maneja las peticiones HTTP de clientes (facturación, protegido).
type CustomerHandler struct {
	uc *billing.CustomerUseCase
}

// NewCustomerHandler construye el handler.
func NewCustomerHandler(uc *billing.CustomerUseCase) *CustomerHandler {
	return &CustomerHandler{uc: uc}
}

// Create POST /api/customers
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

// List GET /api/customers?limit=20&offset=0
func (h *CustomerHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	list, err := h.uc.List(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(list)
}

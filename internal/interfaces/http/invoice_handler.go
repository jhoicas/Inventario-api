package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/billing"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
)

// InvoiceHandler maneja las peticiones HTTP de facturación (protegido).
type InvoiceHandler struct {
	uc *billing.CreateInvoiceUseCase
}

// NewInvoiceHandler construye el handler.
func NewInvoiceHandler(uc *billing.CreateInvoiceUseCase) *InvoiceHandler {
	return &InvoiceHandler{uc: uc}
}

// Create crea una factura y descuenta inventario.
// POST /api/invoices
func (h *InvoiceHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateInvoiceRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	invoice, err := h.uc.CreateInvoice(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cliente, bodega o producto no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if err == domain.ErrInsufficientStock {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "INSUFFICIENT_STOCK", Message: "stock insuficiente"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(invoice)
}

// GetByID obtiene el detalle completo de una factura.
// GET /api/invoices/:id
func (h *InvoiceHandler) GetByID(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	invoice, err := h.uc.GetInvoice(c.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(invoice)
}

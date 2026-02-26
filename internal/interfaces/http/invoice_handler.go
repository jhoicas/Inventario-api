package http

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// InvoiceHandler maneja las peticiones HTTP de facturación (protegido).
type InvoiceHandler struct {
	uc    *billing.CreateInvoiceUseCase
	pdfUC *billing.PDFUseCase
}

// NewInvoiceHandler construye el handler.
func NewInvoiceHandler(uc *billing.CreateInvoiceUseCase, pdfUC *billing.PDFUseCase) *InvoiceHandler {
	return &InvoiceHandler{uc: uc, pdfUC: pdfUC}
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
		if errors.Is(err, domain.ErrInsufficientStock) {
			// El mensaje contiene el SKU: "stock insuficiente para SKU 'PROD-001'"
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Code:    "INSUFFICIENT_STOCK",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(invoice)
}

// GetDIANStatus devuelve el estado DIAN de una factura (endpoint de polling para el frontend).
// GET /api/invoices/:id/status
// El frontend consulta este endpoint periódicamente hasta que dian_status sea
// EXITOSO o RECHAZADO, evitando websockets complejos para un flujo que suele tardar <5 s.
func (h *InvoiceHandler) GetDIANStatus(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	status, err := h.uc.GetInvoiceDIANStatus(c.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(status)
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

// DownloadPDF genera y devuelve la representación gráfica (PDF) de la factura.
// GET /api/invoices/:id/pdf
//
// El PDF se devuelve inline para que el navegador lo muestre directamente,
// o lo descargue si el cliente lo solicita con Accept: application/pdf.
// La factura debe tener CUFE (estado distinto de DRAFT).
func (h *InvoiceHandler) DownloadPDF(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}

	pdfBytes, filename, err := h.pdfUC.DownloadInvoicePDF(c.Context(), companyID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura no encontrada"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "NOT_READY", Message: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	c.Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	return c.Send(pdfBytes)
}

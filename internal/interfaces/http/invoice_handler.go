package http

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CreateInvoiceUseCase interface para permitir mocking en tests.
type CreateInvoiceUseCase interface {
	CreateInvoice(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error)
	GetInvoiceDIANStatus(ctx context.Context, companyID, id string) (*dto.InvoiceDIANStatusDTO, error)
	GetInvoice(ctx context.Context, companyID, id string) (*dto.InvoiceResponse, error)
}

// CreateCreditNoteUseCase interface para permitir mocking en tests.
type CreateCreditNoteUseCase interface {
	CreateCreditNote(ctx context.Context, companyID, userID, invoiceID string, in dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error)
}

// CreateDebitNoteUseCase interface para permitir mocking en tests.
type CreateDebitNoteUseCase interface {
	CreateDebitNote(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error)
}

// InvoicePDFUseCase interface para permitir mocking en tests.
type InvoicePDFUseCase interface {
	DownloadInvoicePDF(ctx context.Context, companyID, invoiceID string) (pdfBytes []byte, filename string, err error)
}

// InvoiceHandler maneja las peticiones HTTP de facturación (protegido).
type InvoiceHandler struct {
	uc       CreateInvoiceUseCase
	returnUC CreateCreditNoteUseCase
	debitUC  CreateDebitNoteUseCase
	pdfUC    InvoicePDFUseCase
}

// NewInvoiceHandler construye el handler.
func NewInvoiceHandler(
	uc CreateInvoiceUseCase,
	returnUC CreateCreditNoteUseCase,
	pdfUC InvoicePDFUseCase,
) *InvoiceHandler {
	return &InvoiceHandler{
		uc:       uc,
		returnUC: returnUC,
		pdfUC:    pdfUC,
	}
}

// NewInvoiceHandlerWithDebit construye el handler con soporte explícito de nota débito.
func NewInvoiceHandlerWithDebit(
	uc CreateInvoiceUseCase,
	returnUC CreateCreditNoteUseCase,
	debitUC CreateDebitNoteUseCase,
	pdfUC InvoicePDFUseCase,
) *InvoiceHandler {
	h := NewInvoiceHandler(uc, returnUC, pdfUC)
	h.debitUC = debitUC
	return h
}

// Create godoc
// @Summary      Crear factura
// @Description  Crea una factura electrónica y descuenta inventario del almacén
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateInvoiceRequest  true  "Datos de la factura"
// @Success      201   {object}  dto.InvoiceResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/invoices [post]
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

// HandleReturn godoc
// @Summary      Registrar devolución de factura (Nota Crédito)
// @Description  Registra una Nota Crédito electrónica y revierte parcialmente o totalmente una factura existente
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "ID de la factura original"
// @Param        body  body      dto.ReturnInvoiceRequest true  "Productos devueltos y bodega de reingreso"
// @Success      201   {object}  dto.InvoiceResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/return [post]
func (h *InvoiceHandler) HandleReturn(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}

	var in dto.ReturnInvoiceRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	creditNote, err := h.returnUC.CreateCreditNote(c.Context(), companyID, userID, invoiceID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura, producto o bodega no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if errors.Is(err, domain.ErrInsufficientStock) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Code:    "INSUFFICIENT_STOCK",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(creditNote)
}

// HandleDebitNote godoc
// @Summary      Registrar nota débito
// @Description  Registra una Nota Débito electrónica asociada a una factura existente y la envía a DIAN
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string                      true  "ID de la factura original"
// @Param        body  body      dto.CreateDebitNoteRequest  true  "Motivo e ítems de la nota débito"
// @Success      201   {object}  dto.DebitNoteResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/debit-note [post]
func (h *InvoiceHandler) HandleDebitNote(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	if h.debitUC == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: "servicio de nota débito no disponible"})
	}

	var in dto.CreateDebitNoteRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	debitNote, err := h.debitUC.CreateDebitNote(c.Context(), companyID, userID, invoiceID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura o producto no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if errors.Is(err, domain.ErrInsufficientStock) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Code:    "INSUFFICIENT_STOCK",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(debitNote)
}

// GetDIANStatus godoc
// @Summary      Obtener estado DIAN de una factura
// @Description  Devuelve el estado de la validación DIAN de una factura electrónica (polling desde el frontend)
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "ID de la factura"
// @Success      200  {object}  dto.InvoiceDIANStatusDTO
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/status [get]
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

// GetByID godoc
// @Summary      Obtener factura por ID
// @Description  Devuelve el detalle completo de una factura electrónica
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "ID de la factura"
// @Success      200  {object}  dto.InvoiceResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices/{id} [get]
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

// DownloadPDF godoc
// @Summary      Descargar PDF de la factura
// @Description  Genera y devuelve la representación gráfica (PDF) de la factura electrónica. La factura debe tener CUFE (estado distinto de DRAFT)
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      application/pdf
// @Param        id   path      string  true  "ID de la factura"
// @Success      200  {string}  binary  "PDF de la factura"
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/pdf [get]
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

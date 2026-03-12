package http

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CreateInvoiceUseCase interface para permitir mocking en tests.
type CreateInvoiceUseCase interface {
	CreateInvoice(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error)
	GetInvoiceDIANStatus(ctx context.Context, companyID, id string) (*dto.InvoiceDIANStatusDTO, error)
	GetInvoice(ctx context.Context, companyID, id string) (*dto.InvoiceResponse, error)
	ListInvoices(ctx context.Context, companyID string, in dto.InvoiceFilter) (*dto.InvoiceListResponse, error)
}

// CreateCreditNoteUseCase interface para permitir mocking en tests.
type CreateCreditNoteUseCase interface {
	CreateCreditNote(ctx context.Context, companyID, userID, invoiceID string, in dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error)
}

// CreateDebitNoteUseCase interface para permitir mocking en tests.
type CreateDebitNoteUseCase interface {
	CreateDebitNote(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error)
}

// VoidInvoiceUseCase interface para anulación por nota crédito.
type VoidInvoiceUseCase interface {
	VoidInvoice(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error)
}

// InvoicePDFUseCase interface para permitir mocking en tests.
type InvoicePDFUseCase interface {
	DownloadInvoicePDF(ctx context.Context, companyID, invoiceID string) (pdfBytes []byte, filename string, err error)
}

// InvoiceDIANRetryUseCase interface para reintento manual DIAN.
type InvoiceDIANRetryUseCase interface {
	RetryDIAN(ctx context.Context, companyID, invoiceID string) (*dto.InvoiceDIANStatusDTO, error)
}

// InvoiceMailerUseCase interfaz para el envío manual de correo de factura.
type InvoiceMailerUseCase interface {
	SendInvoiceEmailSync(ctx context.Context, companyID, invoiceID string) error
	SendCustomEmailSync(ctx context.Context, companyID, to, subject, body string) error
}

// InvoiceHandler maneja las peticiones HTTP de facturación (protegido).
type InvoiceHandler struct {
	uc       CreateInvoiceUseCase
	returnUC CreateCreditNoteUseCase
	debitUC  CreateDebitNoteUseCase
	voidUC   VoidInvoiceUseCase
	pdfUC    InvoicePDFUseCase
	mailerUC InvoiceMailerUseCase
	retryUC  InvoiceDIANRetryUseCase
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
		retryUC: func() InvoiceDIANRetryUseCase {
			if r, ok := uc.(InvoiceDIANRetryUseCase); ok {
				return r
			}
			return nil
		}(),
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

// NewInvoiceHandlerWithBillingOps construye el handler con nota débito, anulación y mailer.
func NewInvoiceHandlerWithBillingOps(
	uc CreateInvoiceUseCase,
	returnUC CreateCreditNoteUseCase,
	debitUC CreateDebitNoteUseCase,
	voidUC VoidInvoiceUseCase,
	pdfUC InvoicePDFUseCase,
	mailerUC ...InvoiceMailerUseCase,
) *InvoiceHandler {
	h := NewInvoiceHandlerWithDebit(uc, returnUC, debitUC, pdfUC)
	h.voidUC = voidUC
	if len(mailerUC) > 0 {
		h.mailerUC = mailerUC[0]
	}
	return h
}

// GetInvoices godoc
// @Summary      Listar facturas
// @Description  Devuelve facturas paginadas filtradas por fecha, cliente, estado DIAN y prefijo
// @Tags         billing
// @Security     Bearer
// @Produce      json
// @Param        start_date   query     string  false  "Fecha inicio (YYYY-MM-DD)"
// @Param        end_date     query     string  false  "Fecha fin (YYYY-MM-DD)"
// @Param        customer_id  query     string  false  "ID del cliente"
// @Param        dian_status  query     string  false  "Estado DIAN (DRAFT|SIGNED|EXITOSO|RECHAZADO|ERROR_GENERATION)"
// @Param        prefix       query     string  false  "Prefijo de la factura"
// @Param        limit        query     int     false  "Límite de resultados"   default(20)
// @Param        offset       query     int     false  "Desplazamiento"         default(0)
// @Success      200  {object}  dto.InvoiceListResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices [get]
func (h *InvoiceHandler) GetInvoices(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}

	filter := dto.InvoiceFilter{
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		CustomerID: c.Query("customer_id"),
		DIANStatus: c.Query("dian_status"),
		Prefix:     c.Query("prefix"),
		Limit:      c.QueryInt("limit", 20),
		Offset:     c.QueryInt("offset", 0),
	}

	out, err := h.uc.ListInvoices(c.Context(), companyID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
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

// HandleVoidInvoice godoc
// @Summary      Anular factura por nota crédito total
// @Description  Crea una Nota Crédito total sobre una factura en estado Sent, la envía a DIAN y marca la factura como VOID
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string                       true  "ID de la factura original"
// @Param        body  body      dto.CreateVoidInvoiceRequest true  "Código de concepto y motivo"
// @Success      201   {object}  dto.VoidInvoiceResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/void [post]
func (h *InvoiceHandler) HandleVoidInvoice(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	if h.voidUC == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: "servicio de anulación no disponible"})
	}

	var in dto.CreateVoidInvoiceRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	out, err := h.voidUC.VoidInvoice(c.Context(), companyID, userID, invoiceID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if err == domain.ErrConflict {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "CONFLICT", Message: "la factura debe estar en estado Sent"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(out)
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

// SendEmail godoc
// @Summary      Enviar factura por correo
// @Description  Envía la factura electrónica (PDF + XML firmado) al correo del cliente
// @Tags         billing
// @Security     Bearer
// @Param        id   path      string  true  "ID de la factura"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/send-email [post]
func (h *InvoiceHandler) SendEmail(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	if h.mailerUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "MAILER_DISABLED", Message: "envío de correo no configurado"})
	}

	if err := h.mailerUC.SendInvoiceEmailSync(c.Context(), companyID, id); err != nil {
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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "correo enviado correctamente"})
}

// SendCustomEmail godoc
// @Summary      Enviar correo libre
// @Description  Envía un correo manual indicando destinatario, asunto y cuerpo
// @Tags         billing
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body      dto.SendCustomEmailRequest  true  "Datos del correo"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/emails/send [post]
func (h *InvoiceHandler) SendCustomEmail(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.mailerUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "MAILER_DISABLED", Message: "envío de correo no configurado"})
	}

	var in dto.SendCustomEmailRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	in.To = strings.TrimSpace(in.To)
	in.Subject = strings.TrimSpace(in.Subject)
	in.Body = strings.TrimSpace(in.Body)

	if in.To == "" || in.Subject == "" || in.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "to, subject y body son requeridos"})
	}
	if !strings.Contains(in.To, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "email destino inválido"})
	}

	if err := h.mailerUC.SendCustomEmailSync(c.Context(), companyID, in.To, in.Subject, in.Body); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "correo enviado correctamente"})
}

// RetryDIAN godoc
// @Summary      Reintentar envío DIAN
// @Description  Reintenta el envío a la DIAN de una factura en estado CONTINGENCIA
// @Tags         billing
// @Security     Bearer
// @Param        id   path      string  true  "ID de la factura"
// @Success      200  {object}  dto.InvoiceDIANStatusDTO
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/invoices/{id}/retry-dian [post]
func (h *InvoiceHandler) RetryDIAN(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.retryUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "RETRY_DISABLED", Message: "reintento DIAN no configurado"})
	}

	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}

	status, err := h.retryUC.RetryDIAN(c.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "factura no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		if err == domain.ErrConflict {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "INVALID_STATE", Message: "la factura debe estar en estado CONTINGENCIA"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(status)
}

package billing

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CustomerLookupHandler expone el servicio GetAcquirer de la DIAN vía HTTP.
type CustomerLookupHandler struct {
	dianEnv string // "dev" | "test" | "prod"
}

// NewCustomerLookupHandler crea el handler. dianEnv proviene de cfg.DIAN.AppEnv.
func NewCustomerLookupHandler(dianEnv string) *CustomerLookupHandler {
	return &CustomerLookupHandler{dianEnv: dianEnv}
}

// Lookup godoc
// @Summary      Consultar contribuyente en DIAN
// @Description  Retorna los datos tributarios de un contribuyente consultando el servicio GetAcquirer de la DIAN
// @Tags         customers
// @Security     Bearer
// @Produce      json
// @Param        id_type    query     string  true  "Tipo de documento (ej: 31=NIT, 13=Cédula)"
// @Param        id_number  query     string  true  "Número de documento"
// @Success      200        {object}  AcquirerInfo
// @Failure      400        {object}  dto.ErrorResponse
// @Failure      401        {object}  dto.ErrorResponse
// @Failure      404        {object}  dto.ErrorResponse
// @Failure      502        {object}  dto.ErrorResponse
// @Router       /api/customers/lookup [get]
func (h *CustomerLookupHandler) Lookup(c *fiber.Ctx) error {
	idType := c.Query("id_type")
	idNumber := c.Query("id_number")
	if idType == "" || idNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code:    "MISSING_PARAMS",
			Message: "id_type e id_number son requeridos",
		})
	}

	// Obtener configuración DIAN inyectada por el middleware
	dianURL, dianCert, err := GetDIANConfigFromContext(c)
	if err != nil {
		// Fallback a variables globales si no está disponible
		dianURL = "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc"
		dianCert = ""
	}

	info, err := GetAcquirer(c.Context(), dianURL, idType, idNumber, dianCert)
	if err != nil {
		if errors.Is(err, ErrAcquirerNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResponse{
			Code:    "DIAN_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(info)
}

// Similar to CreateCreditNote handler above

type VoidInvoiceUseCase interface {
	VoidInvoice(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error)
}

type BillingHandler struct {
	voidUC VoidInvoiceUseCase
}

func NewBillingHandler(voidUC VoidInvoiceUseCase) *BillingHandler {
	return &BillingHandler{voidUC: voidUC}
}

// VoidInvoice handles POST /api/invoices/{id}/void
func (h *BillingHandler) VoidInvoice(c *fiber.Ctx) error {
	if h.voidUC == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Code:    "INTERNAL",
			Message: "servicio de anulación no disponible",
		})
	}

	invoiceID := c.Params("id")
	if invoiceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code:    "VALIDATION",
			Message: "id requerido",
		})
	}

	companyID, _ := c.Locals("company_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "token inválido",
		})
	}

	var in dto.CreateVoidInvoiceRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code:    "INVALID_BODY",
			Message: "cuerpo inválido",
		})
	}
	if in.ConceptCode < 1 || in.ConceptCode > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code:    "VALIDATION",
			Message: "concept_code debe estar entre 1 y 5",
		})
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

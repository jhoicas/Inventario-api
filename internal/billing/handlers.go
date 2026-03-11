package billing

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
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

	info, err := GetAcquirer(c.Context(), h.dianEnv, idType, idNumber)
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

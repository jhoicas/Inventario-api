package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/usecase"
)

// AnalyticsHandler maneja los endpoints de analítica de rentabilidad.
type AnalyticsHandler struct {
	uc *usecase.AnalyticsUseCase
}

// NewAnalyticsHandler construye el handler.
func NewAnalyticsHandler(uc *usecase.AnalyticsUseCase) *AnalyticsHandler {
	return &AnalyticsHandler{uc: uc}
}

// GetMargins godoc
// @Summary      Reporte de márgenes por canal y ranking de SKUs (Pareto 80/20)
// @Description  Devuelve rentabilidad por canal de venta y el ranking de SKUs más rentables
//               con análisis de Pareto. Requiere módulo 'analytics' activo.
// @Tags         analytics
// @Security     Bearer
// @Produce      json
// @Param        start_date  query  string  false  "Inicio del período (YYYY-MM-DD). Default: primer día del mes."
// @Param        end_date    query  string  false  "Fin del período (YYYY-MM-DD). Default: hoy."
// @Param        top_n       query  int     false  "Máx. SKUs en el ranking (default 20, max 200)."
// @Success      200  {object}  dto.MarginsReportDTO
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/analytics/margins [get]
func (h *AnalyticsHandler) GetMargins(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Code: "UNAUTHORIZED", Message: "company_id no encontrado en el token",
		})
	}

	var req dto.MarginsReportRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code: "INVALID_PARAMS", Message: "parámetros de consulta inválidos",
		})
	}

	report, err := h.uc.GetMarginsReport(c.Context(), companyID, req)
	if err != nil {
		// Errores de validación de fechas son errores del cliente
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code: "BAD_REQUEST", Message: err.Error(),
		})
	}

	return c.JSON(report)
}

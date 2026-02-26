package http

import (
	"github.com/gofiber/fiber/v2"
	appanalytics "github.com/tu-usuario/inventory-pro/internal/application/analytics"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
)

// DashboardHandler maneja los endpoints del módulo de Dashboard.
type DashboardHandler struct {
	uc *appanalytics.DashboardUseCase
}

// NewDashboardHandler construye el handler.
func NewDashboardHandler(uc *appanalytics.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{uc: uc}
}

// GetSummary devuelve el resumen financiero del día y del mes en curso.
// GET /api/dashboard/summary
//
// Respuesta: DashboardSummaryDTO (today_sales, today_margin, monthly_sales,
// monthly_margin, top_skus[5], date_label).
// No requiere parámetros; las fechas se calculan automáticamente en el servidor.
func (h *DashboardHandler) GetSummary(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Code: "UNAUTHORIZED", Message: "company_id no encontrado en el token",
		})
	}

	summary, err := h.uc.GetSummary(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Code: "INTERNAL", Message: err.Error(),
		})
	}

	return c.JSON(summary)
}

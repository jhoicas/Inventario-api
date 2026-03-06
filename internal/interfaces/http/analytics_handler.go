package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
)

// AnalyticsHandler maneja los endpoints de analítica de rentabilidad.
type AnalyticsHandler struct {
	uc             *usecase.AnalyticsUseCase
	rawMaterialUC  *usecase.RawMaterialAnalyticsUseCase
}

// NewAnalyticsHandler construye el handler.
func NewAnalyticsHandler(uc *usecase.AnalyticsUseCase, rawMaterialUC *usecase.RawMaterialAnalyticsUseCase) *AnalyticsHandler {
	return &AnalyticsHandler{uc: uc, rawMaterialUC: rawMaterialUC}
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

// GetRawMaterialImpactRanking godoc
// @Summary      Ranking de impacto de materias primas
// @Description  Devuelve el ranking de materias primas por peso financiero en productos vendidos en el período (BOM + coste).
// @Tags         analytics
// @Security     Bearer
// @Produce      json
// @Param        start_date  query  string  false  "Inicio del período (YYYY-MM-DD). Default: primer día del mes."
// @Param        end_date    query  string  false  "Fin del período (YYYY-MM-DD). Default: hoy."
// @Param        limit       query  int     false  "Máx. materias primas en el ranking (default 50)."
// @Success      200  {array}  dto.RawMaterialImpactDTO
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/analytics/raw-materials-impact [get]
func (h *AnalyticsHandler) GetRawMaterialImpactRanking(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Code: "UNAUTHORIZED", Message: "company_id no encontrado en el token",
		})
	}

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	now := time.Now()

	var startDate, endDate time.Time
	var err error
	if endStr == "" {
		endDate = now
	} else {
		endDate, err = time.ParseInLocation("2006-01-02", endStr, now.Location())
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Code: "INVALID_PARAMS", Message: "end_date inválido; use formato YYYY-MM-DD",
			})
		}
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	if startStr == "" {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		startDate, err = time.ParseInLocation("2006-01-02", startStr, now.Location())
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Code: "INVALID_PARAMS", Message: "start_date inválido; use formato YYYY-MM-DD",
			})
		}
	}
	if startDate.After(endDate) {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code: "INVALID_PARAMS", Message: "start_date no puede ser posterior a end_date",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	resultados, err := h.rawMaterialUC.GetRawMaterialImpactRanking(c.Context(), companyID, startDate, endDate, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Code: "INTERNAL", Message: err.Error(),
		})
	}
	return c.JSON(resultados)
}

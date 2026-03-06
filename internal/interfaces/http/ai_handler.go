package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
)

// AIHandler maneja los endpoints de clasificación arancelaria asistida por IA.
type AIHandler struct {
	uc *usecase.AIUseCase
}

// NewAIHandler construye el handler.
func NewAIHandler(uc *usecase.AIUseCase) *AIHandler {
	return &AIHandler{uc: uc}
}

// SuggestClassification está deshabilitado: la parametrización de impuestos y códigos DIAN
// es estrictamente manual (contador/admin). Se mantiene el método para no romper referencias.
func (h *AIHandler) SuggestClassification(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(dto.ErrorResponse{
		Code: "FEATURE_DISABLED",
		Message: "la sugerencia de clasificación por IA está deshabilitada; la parametrización de impuestos y códigos DIAN es manual",
	})
	// Lógica anterior (IA) eliminada por requerimiento de negocio:
	// if GetUserID(c) == "" { ... }
	// var req dto.AIClassificationRequest; c.BodyParser(&req)
	// result, err := h.uc.SuggestClassification(c.Context(), req); return c.JSON(result)
}

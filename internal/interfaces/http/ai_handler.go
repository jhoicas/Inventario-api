package http

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/usecase"
)

// AIHandler maneja los endpoints de clasificación arancelaria asistida por IA.
type AIHandler struct {
	uc *usecase.AIUseCase
}

// NewAIHandler construye el handler.
func NewAIHandler(uc *usecase.AIUseCase) *AIHandler {
	return &AIHandler{uc: uc}
}

// SuggestClassification godoc
// @Summary      Sugerir clasificación arancelaria DIAN con IA
// @Description  Analiza el nombre y descripción de un producto y devuelve el código UNSPSC
//               más probable, la tarifa de IVA (0, 5 o 19) y el razonamiento del modelo.
//               Requiere autenticación. Timeout interno de 10 s.
// @Tags         ai
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.AIClassificationRequest  true  "product_name (obligatorio) y description (opcional)"
// @Success      200   {object}  dto.AIClassificationDTO
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      408   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/ai/suggest-classification [post]
func (h *AIHandler) SuggestClassification(c *fiber.Ctx) error {
	if GetUserID(c) == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Code: "UNAUTHORIZED", Message: "token inválido",
		})
	}

	var req dto.AIClassificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Code: "INVALID_BODY", Message: "cuerpo de la petición inválido",
		})
	}

	result, err := h.uc.SuggestClassification(c.Context(), req)
	if err != nil {
		// Timeout del contexto → 408 Request Timeout
		if errors.Is(err, c.Context().Err()) || isTimeout(err) {
			return c.Status(fiber.StatusRequestTimeout).JSON(dto.ErrorResponse{
				Code: "TIMEOUT", Message: "el servicio de IA tardó demasiado; intenta de nuevo",
			})
		}
		// Validación (product_name vacío)
		if strings.Contains(err.Error(), "obligatorio") {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Code: "VALIDATION", Message: err.Error(),
			})
		}
		// API key no configurada
		if strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
				Code: "AI_UNAVAILABLE", Message: "el servicio de clasificación IA no está configurado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Code: "INTERNAL", Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// isTimeout detecta errores de timeout/cancelación de contexto en el mensaje de error.
func isTimeout(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "cancelación")
}

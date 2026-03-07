package crm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// AICRMUseCase casos de uso de IA para CRM (copy, resúmenes, sentimiento).
type AICRMUseCase struct {
	llm ports.LLMService
}

// NewAICRMUseCase construye el caso de uso.
func NewAICRMUseCase(llm ports.LLMService) *AICRMUseCase {
	return &AICRMUseCase{llm: llm}
}

// GenerateCampaignCopy redacta texto de campaña (ej. correo) a partir de un prompt.
func (uc *AICRMUseCase) GenerateCampaignCopy(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", nil
	}
	fullPrompt := fmt.Sprintf("Redacta un correo o texto de campaña de fidelización para el siguiente contexto. Responde solo con el texto listo para enviar, sin explicaciones.\n\nContexto:\n%s", prompt)
	return uc.llm.GenerateText(ctx, fullPrompt)
}

// SummarizeTimeline resume una lista de interacciones para el asesor.
func (uc *AICRMUseCase) SummarizeTimeline(ctx context.Context, interactions []*entity.CRMInteraction) (string, error) {
	if len(interactions) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, m := range interactions {
		fmt.Fprintf(&b, "%d. [%s] %s - %s\n   %s\n", i+1, m.CreatedAt.Format("2006-01-02 15:04"), m.Type, m.Subject, m.Body)
	}
	fullPrompt := fmt.Sprintf("Resume en un párrafo breve (máximo 5 líneas) el siguiente timeline de interacciones con un cliente. Destaca fechas clave y temas. Responde solo con el resumen.\n\n%s", b.String())
	return uc.llm.GenerateText(ctx, fullPrompt)
}

// AnalyzePQRSentiment clasifica el sentimiento de la descripción de un ticket PQR como 'positive', 'neutral' o 'negative'.
func (uc *AICRMUseCase) AnalyzePQRSentiment(ctx context.Context, description string) (string, error) {
	if description == "" {
		return "", nil
	}
	fullPrompt := fmt.Sprintf("Clasifica el sentimiento del siguiente texto (petición, queja o reclamo de un cliente) en exactamente una de estas tres palabras: positive, neutral, negative. Responde ÚNICAMENTE con una de esas tres palabras, nada más.\n\nTexto:\n%s", description)
	raw, err := uc.llm.GenerateText(ctx, fullPrompt)
	if err != nil {
		return "", err
	}
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "positive", "neutral", "negative":
		return s, nil
	default:
		if strings.Contains(s, "positive") {
			return "positive", nil
		}
		if strings.Contains(s, "negative") {
			return "negative", nil
		}
		return "neutral", nil
	}
}

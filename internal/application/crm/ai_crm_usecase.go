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
	systemInstruction := "Eres un estratega senior de CRM y email marketing. Redacta mensajes profesionales, claros y persuasivos, con tono cercano. Entrega solo el texto final listo para enviar, sin explicaciones ni markdown."
	userText := fmt.Sprintf("Contexto de campaña:\n%s", prompt)
	return uc.llm.GenerateTextWithSystem(ctx, systemInstruction, userText)
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
	systemInstruction := "Eres un asistente CRM para ejecutivos de cuenta. Resume interacciones de forma accionable y concisa."
	userText := fmt.Sprintf("Resume en un párrafo breve (máximo 5 líneas) el siguiente timeline de interacciones con un cliente. Destaca fechas clave y temas. Responde solo con el resumen.\n\n%s", b.String())
	return uc.llm.GenerateTextWithSystem(ctx, systemInstruction, userText)
}

// AnalyzePQRSentiment clasifica el sentimiento de la descripción de un ticket PQR como 'positive', 'neutral' o 'negative'.
func (uc *AICRMUseCase) AnalyzePQRSentiment(ctx context.Context, description string) (string, error) {
	if description == "" {
		return "", nil
	}
	systemInstruction := "Eres un clasificador de sentimiento para tickets CRM."
	userText := fmt.Sprintf("Clasifica el sentimiento del siguiente texto (petición, queja o reclamo de un cliente) en exactamente una de estas tres palabras: positive, neutral, negative. Responde ÚNICAMENTE con una de esas tres palabras, nada más.\n\nTexto:\n%s", description)
	raw, err := uc.llm.GenerateTextWithSystem(ctx, systemInstruction, userText)
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

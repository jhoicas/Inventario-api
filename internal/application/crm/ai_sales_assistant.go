package crm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

const (
	retentionSystemPrompt = "Eres un experto en ventas B2B. Redacta un mensaje corto y persuasivo (máx 3 líneas) para que un vendedor llame a un cliente y le ofrezca un descuento o incentivo para recuperarlo."
)

// AISalesAssistant genera mensajes de retencion para clientes en riesgo de churn.
type AISalesAssistant struct {
	llm ports.LLMService
	log *logger.Logger
}

// NewAISalesAssistant construye el asistente IA de ventas.
func NewAISalesAssistant(llm ports.LLMService, log *logger.Logger) *AISalesAssistant {
	return &AISalesAssistant{llm: llm, log: log}
}

// GenerateRetentionPitch produce un pitch breve de retencion usando un prompt de sistema y uno de usuario.
func (s *AISalesAssistant) GenerateRetentionPitch(ctx context.Context, customerName, favoriteProduct string, daysInactive int) (string, error) {
	if s == nil || s.llm == nil {
		return "", fmt.Errorf("asistente IA de ventas no configurado")
	}
	if daysInactive <= 0 {
		return "", fmt.Errorf("daysInactive debe ser mayor que cero")
	}

	customerName = strings.TrimSpace(customerName)
	if customerName == "" {
		customerName = "Cliente"
	}

	favoriteProduct = strings.TrimSpace(favoriteProduct)
	if favoriteProduct == "" {
		favoriteProduct = "Producto no identificado"
	}

	userPrompt := fmt.Sprintf("Cliente: %s, Producto habitual: %s, Días sin comprar: %d", customerName, favoriteProduct, daysInactive)

	// El puerto LLM actual expone solo GenerateText, por lo que encapsulamos ambos prompts en un unico mensaje.
	fullPrompt := fmt.Sprintf("System Prompt:\n%s\n\nUser Prompt:\n%s\n\nResponde unicamente con el mensaje final (maximo 3 lineas).", retentionSystemPrompt, userPrompt)

	pitch, err := s.llm.GenerateText(ctx, fullPrompt)
	if err != nil {
		if s.log != nil {
			s.log.Error().Err(err).Str("customer_name", customerName).Msg("generar pitch de retencion")
		}
		return "", fmt.Errorf("generar pitch de retencion: %w", err)
	}

	pitch = strings.TrimSpace(pitch)
	if pitch == "" {
		return "", fmt.Errorf("el modelo retorno un pitch vacio")
	}

	return enforceMaxLines(pitch, 3), nil
}

func enforceMaxLines(text string, maxLines int) string {
	if maxLines <= 0 {
		return strings.TrimSpace(text)
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:maxLines], "\n")
}

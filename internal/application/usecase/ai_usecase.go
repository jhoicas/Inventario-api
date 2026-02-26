package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/ports"
)

// AIUseCase orquesta la clasificación arancelaria asistida por IA.
// Aplica un timeout de 10 segundos en cada llamada al LLM para evitar
// que las latencias externas bloqueen los goroutines del servidor.
type AIUseCase struct {
	llm ports.LLMService
}

// NewAIUseCase construye el caso de uso inyectando el puerto LLMService.
func NewAIUseCase(llm ports.LLMService) *AIUseCase {
	return &AIUseCase{llm: llm}
}

// SuggestClassification valida la entrada y delega al servicio de LLM.
// Envuelve el contexto con un timeout de 10 s para respetar los SLAs de la API.
func (uc *AIUseCase) SuggestClassification(
	ctx context.Context,
	req dto.AIClassificationRequest,
) (*dto.AIClassificationDTO, error) {
	if req.ProductName == "" {
		return nil, fmt.Errorf("product_name es obligatorio")
	}

	// Timeout de 10 s: las llamadas a LLMs pueden demorar varios segundos.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := uc.llm.SuggestProductClassification(ctx, req.ProductName, req.Description)
	if err != nil {
		return nil, fmt.Errorf("clasificación IA: %w", err)
	}

	return result, nil
}

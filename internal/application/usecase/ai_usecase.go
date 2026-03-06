package usecase

import (
	"context"
	"fmt"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
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

// SuggestClassification está deshabilitado: parametrización de impuestos y códigos DIAN es manual.
// Se mantiene la firma por si algún código sigue llamándola.
func (uc *AIUseCase) SuggestClassification(
	_ context.Context,
	_ dto.AIClassificationRequest,
) (*dto.AIClassificationDTO, error) {
	return nil, fmt.Errorf("sugerencia de clasificación IA deshabilitada: parametrización manual")
	// Lógica anterior (LLM) eliminada por requerimiento de negocio:
	// ctx, cancel := context.WithTimeout(ctx, 10*time.Second); defer cancel()
	// return uc.llm.SuggestProductClassification(ctx, req.ProductName, req.Description)
}

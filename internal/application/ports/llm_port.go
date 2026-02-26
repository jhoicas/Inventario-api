package ports

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// LLMService define el puerto de salida para los servicios de inteligencia artificial.
// Cualquier adaptador (Gemini, OpenAI, Ollama, mock) debe implementar esta interfaz.
// Siguiendo el principio de inversión de dependencias (DIP), el dominio/aplicación
// solo conoce este contrato, no la implementación concreta.
type LLMService interface {
	// SuggestProductClassification analiza el nombre y descripción de un producto
	// y sugiere el código UNSPSC, la tarifa de IVA aplicable y el razonamiento.
	// El contexto debe llevar un timeout para evitar bloqueos en llamadas externas.
	SuggestProductClassification(
		ctx context.Context,
		productName string,
		description string,
	) (*dto.AIClassificationDTO, error)
}

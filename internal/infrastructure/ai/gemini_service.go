package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/ports"
)

// Verificar en tiempo de compilación que GeminiService implementa LLMService.
var _ ports.LLMService = (*GeminiService)(nil)

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

	// systemPrompt es el prompt del sistema que define el rol del modelo y el formato de salida.
	// Usar response_mime_type=application/json obliga a Gemini a devolver JSON puro,
	// eliminando la necesidad de limpiar bloques de markdown.
	systemPrompt = `Eres un experto tributario de la DIAN en Colombia especializado en clasificación arancelaria.
Dado el nombre y descripción de un producto, devuelve ÚNICAMENTE un objeto JSON (sin texto adicional) con la siguiente estructura exacta:
{
  "suggested_unspsc": "<código UNSPSC de 8 dígitos como string>",
  "suggested_tax_rate": <número: 0, 5 o 19>,
  "confidence_score": <número decimal entre 0.0 y 1.0>,
  "reasoning": "<explicación concisa en español de la clasificación y la tarifa de IVA asignada>"
}

Reglas:
- suggested_unspsc: código UNSPSC colombiano de 8 dígitos. Si no hay certeza, usa el código de categoría más cercano.
- suggested_tax_rate: 0 (exento), 5 (tarifa diferencial) o 19 (tarifa general).
- confidence_score: 0.9–1.0 = certeza alta, 0.7–0.89 = probable, <0.7 = estimado.
- reasoning: máximo 200 caracteres, en español.`
)

// GeminiService adaptador que implementa LLMService llamando a la API REST de Google Gemini.
// Usa únicamente la librería estándar de Go (net/http) para no añadir dependencias externas.
type GeminiService struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGeminiService construye el adaptador. model suele ser "gemini-1.5-flash".
// Si apiKey está vacío, las llamadas devuelven ErrNoAPIKey en lugar de fallar en producción.
func NewGeminiService(apiKey, model string) *GeminiService {
	return &GeminiService{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 20 * time.Second, // timeout de red; el caller también pone WithTimeout
		},
	}
}

// ── Estructuras internas para la API de Gemini ────────────────────────────────

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  genConfig       `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type genConfig struct {
	ResponseMIMEType string  `json:"responseMimeType"` // "application/json" → JSON puro garantizado
	Temperature      float32 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// llmClassificationPayload es el JSON que esperamos recibir del modelo.
type llmClassificationPayload struct {
	SuggestedUNSPSC  string  `json:"suggested_unspsc"`
	SuggestedTaxRate float64 `json:"suggested_tax_rate"`
	ConfidenceScore  float64 `json:"confidence_score"`
	Reasoning        string  `json:"reasoning"`
}

// ── Implementación del puerto ─────────────────────────────────────────────────

// SuggestProductClassification llama a Gemini con el nombre y descripción del producto
// y devuelve la clasificación arancelaria sugerida.
func (s *GeminiService) SuggestProductClassification(
	ctx context.Context,
	productName string,
	description string,
) (*dto.AIClassificationDTO, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("AI: GEMINI_API_KEY no configurado")
	}

	userText := fmt.Sprintf("Nombre del producto: %s\nDescripción: %s", productName, description)

	payload := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: userText}},
			},
		},
		GenerationConfig: genConfig{
			ResponseMIMEType: "application/json",
			Temperature:      0.2, // baja temperatura para respuestas más deterministas
			MaxOutputTokens:  256,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("AI: serializar request: %w", err)
	}

	url := fmt.Sprintf(geminiBaseURL, s.model, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("AI: crear HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("AI: timeout o cancelación: %w", ctx.Err())
		}
		return nil, fmt.Errorf("AI: llamada HTTP fallida: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("AI: leer respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Intentar extraer el mensaje de error de Gemini
		var errResp geminiResponse
		if jsonErr := json.Unmarshal(rawBody, &errResp); jsonErr == nil && errResp.Error != nil {
			return nil, fmt.Errorf("AI: Gemini error %d: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return nil, fmt.Errorf("AI: Gemini HTTP %d", resp.StatusCode)
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(rawBody, &gemResp); err != nil {
		return nil, fmt.Errorf("AI: deserializar respuesta Gemini: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("AI: Gemini devolvió respuesta vacía")
	}

	rawJSON := strings.TrimSpace(gemResp.Candidates[0].Content.Parts[0].Text)

	var classification llmClassificationPayload
	if err := json.Unmarshal([]byte(rawJSON), &classification); err != nil {
		return nil, fmt.Errorf("AI: respuesta del modelo no es JSON válido: %w (respuesta: %s)", err, rawJSON)
	}

	// Normalizar tax rate a uno de los valores válidos de Colombia (0, 5, 19)
	taxRate := normalizeTaxRate(classification.SuggestedTaxRate)

	// Clamp confidence al rango [0, 1]
	confidence := classification.ConfidenceScore
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	return &dto.AIClassificationDTO{
		SuggestedUNSPSC:  classification.SuggestedUNSPSC,
		SuggestedTaxRate: decimal.NewFromFloat(taxRate),
		ConfidenceScore:  confidence,
		Reasoning:        classification.Reasoning,
	}, nil
}

// normalizeTaxRate fuerza el valor al conjunto {0, 5, 19}.
// Si el modelo devuelve algo distinto, elige el más cercano.
func normalizeTaxRate(raw float64) float64 {
	valid := []float64{0, 5, 19}
	best := valid[0]
	bestDiff := abs(raw - best)
	for _, v := range valid[1:] {
		if d := abs(raw - v); d < bestDiff {
			bestDiff = d
			best = v
		}
	}
	return best
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

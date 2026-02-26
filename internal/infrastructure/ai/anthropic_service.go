package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
)

// Verificar en tiempo de compilación que AnthropicService implementa LLMService.
var _ ports.LLMService = (*AnthropicService)(nil)

const (
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"

	anthropicSystemPrompt = `Eres un experto tributario de la DIAN en Colombia especializado en clasificación arancelaria.
Devuelve ÚNICAMENTE un objeto JSON válido (sin markdown, sin bloques de código` + " ```json" + `) con esta estructura exacta:
{
  "suggested_unspsc": "<código UNSPSC de 8 dígitos como string>",
  "suggested_tax_rate": <número: 0, 5 o 19>,
  "confidence_score": <número decimal entre 0.0 y 1.0>,
  "reasoning": "<explicación concisa en español de la clasificación y la tarifa de IVA, máximo 200 caracteres>"
}

Reglas:
- suggested_unspsc: código UNSPSC colombiano de 8 dígitos. Si no hay certeza alta, usa el código de categoría más cercano.
- suggested_tax_rate: 0 (exento por ley colombiana), 5 (tarifa diferencial) o 19 (tarifa general).
- confidence_score: 0.9–1.0 = alta certeza, 0.7–0.89 = probable, <0.7 = estimado.
- No incluyas texto fuera del JSON. Solo el objeto JSON.`
)

// AnthropicService adaptador que implementa LLMService usando la API REST de Anthropic (Claude).
// Usa net/http de la librería estándar de Go; no requiere el SDK oficial.
type AnthropicService struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAnthropicService construye el adaptador.
// model suele ser "claude-3-5-haiku-20241022".
// Si apiKey está vacío las llamadas devuelven error descriptivo en lugar de panic.
func NewAnthropicService(apiKey, model string) *AnthropicService {
	return &AnthropicService{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			// Timeout de red de 25 s; el use case impone además un context.WithTimeout de 10 s.
			Timeout: 25 * time.Second,
		},
	}
}

// ── Estructuras internas del protocolo Anthropic Messages API ─────────────────

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// jsonBlockRe extrae el primer objeto JSON del texto aunque Claude lo envuelva en markdown.
// Captura desde el primer '{' hasta el último '}' coincidente.
var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

// ── Implementación del puerto ─────────────────────────────────────────────────

// SuggestProductClassification envía el nombre y descripción del producto a Claude
// y devuelve la clasificación arancelaria sugerida.
func (s *AnthropicService) SuggestProductClassification(
	ctx context.Context,
	productName string,
	description string,
) (*dto.AIClassificationDTO, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("AI: ANTHROPIC_API_KEY no configurado")
	}

	userContent := fmt.Sprintf("Producto: %s\nDescripción: %s", productName, description)

	payload := anthropicRequest{
		Model:     s.model,
		MaxTokens: 1024,
		System:    anthropicSystemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userContent},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("AI: serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("AI: crear HTTP request: %w", err)
	}
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

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

	// Manejar errores HTTP de la API de Anthropic
	if resp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		if jsonErr := json.Unmarshal(rawBody, &errResp); jsonErr == nil && errResp.Error != nil {
			return nil, fmt.Errorf("AI: Anthropic error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("AI: Anthropic HTTP %d: %s", resp.StatusCode, string(rawBody))
	}

	var anthResp anthropicResponse
	if err := json.Unmarshal(rawBody, &anthResp); err != nil {
		return nil, fmt.Errorf("AI: deserializar respuesta Anthropic: %w", err)
	}

	if len(anthResp.Content) == 0 {
		return nil, fmt.Errorf("AI: Claude devolvió respuesta vacía")
	}

	rawText := anthResp.Content[0].Text

	// Parseo seguro: extraer solo el bloque JSON aunque Claude añada texto adicional.
	cleanJSON := extractJSON(rawText)
	if cleanJSON == "" {
		return nil, fmt.Errorf("AI: no se encontró JSON válido en la respuesta del modelo (respuesta: %s)", rawText)
	}

	var classification llmClassificationPayload
	if err := json.Unmarshal([]byte(cleanJSON), &classification); err != nil {
		return nil, fmt.Errorf("AI: parsear JSON de clasificación: %w (JSON extraído: %s)", err, cleanJSON)
	}

	taxRate := normalizeTaxRate(classification.SuggestedTaxRate)

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

// extractJSON extrae el primer objeto JSON bien formado de un texto libre.
// Estrategia en dos pasos:
//  1. Eliminar bloques de código markdown (```json … ``` o ``` … ```).
//  2. Usar regex para capturar el primer bloque { … }.
func extractJSON(text string) string {
	// Eliminar bloques markdown ```json ... ``` o ``` ... ```
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "```"); idx != -1 {
		// Quitar la línea de apertura (```json o ```)
		after := text[idx+3:]
		if nl := strings.Index(after, "\n"); nl != -1 {
			after = after[nl+1:]
		}
		// Quitar el cierre ```
		if close := strings.LastIndex(after, "```"); close != -1 {
			after = after[:close]
		}
		text = strings.TrimSpace(after)
	}

	// Si el texto resultante ya empieza con '{', usarlo directamente
	if strings.HasPrefix(text, "{") {
		return text
	}

	// Fallback: regex para extraer el primer {...}
	match := jsonBlockRe.FindString(text)
	return strings.TrimSpace(match)
}

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

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/shopspring/decimal"
)

var _ ports.LLMService = (*AnthropicService)(nil)

const (
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"
)

type AnthropicService struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAnthropicService(apiKey, model string) *AnthropicService {
	return &AnthropicService{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
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

type llmClassificationPayload struct {
	SuggestedUNSPSC  string  `json:"suggested_unspsc"`
	SuggestedTaxRate float64 `json:"suggested_tax_rate"`
	ConfidenceScore  float64 `json:"confidence_score"`
	Reasoning        string  `json:"reasoning"`
}

func (s *AnthropicService) SuggestProductClassification(ctx context.Context, productName string, description string) (*dto.AIClassificationDTO, error) {
	system := `Eres un experto tributario de la DIAN en Colombia especializado en clasificación arancelaria.
Devuelve ÚNICAMENTE un objeto JSON válido con:
{
  "suggested_unspsc": "<código UNSPSC de 8 dígitos como string>",
  "suggested_tax_rate": <número: 0, 5 o 19>,
  "confidence_score": <número decimal entre 0.0 y 1.0>,
  "reasoning": "<explicación concisa en español>"
}`
	user := fmt.Sprintf("Producto: %s\nDescripción: %s", strings.TrimSpace(productName), strings.TrimSpace(description))
	raw, err := s.GenerateTextWithSystem(ctx, system, user)
	if err != nil {
		return nil, err
	}
	raw = extractJSON(raw)
	var payload llmClassificationPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("AI: respuesta de Anthropic no es JSON válido: %w", err)
	}
	taxRate := normalizeTaxRate(payload.SuggestedTaxRate)
	confidence := payload.ConfidenceScore
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}
	return &dto.AIClassificationDTO{
		SuggestedUNSPSC:  payload.SuggestedUNSPSC,
		SuggestedTaxRate: decimal.NewFromFloat(taxRate),
		ConfidenceScore:  confidence,
		Reasoning:        payload.Reasoning,
	}, nil
}

func (s *AnthropicService) GenerateText(ctx context.Context, prompt string) (string, error) {
	return s.GenerateTextWithSystem(ctx, "You are a helpful assistant. Return only the requested output.", prompt)
}

func (s *AnthropicService) GenerateTextWithSystem(ctx context.Context, systemInstruction, userText string) (string, error) {
	if s.apiKey == "" {
		return "", fmt.Errorf("AI: ANTHROPIC_API_KEY no configurado")
	}
	payload := anthropicRequest{
		Model:     s.model,
		MaxTokens: 2048,
		System:    systemInstruction,
		Messages:  []anthropicMessage{{Role: "user", Content: userText}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("AI: serializar request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("AI: crear HTTP request: %w", err)
	}
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("AI: timeout o cancelación: %w", ctx.Err())
		}
		return "", fmt.Errorf("AI: llamada HTTP fallida: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", fmt.Errorf("AI: leer respuesta: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		if json.Unmarshal(rawBody, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("AI: Anthropic error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return "", fmt.Errorf("AI: Anthropic HTTP %d: %s", resp.StatusCode, string(rawBody))
	}

	var out anthropicResponse
	if err := json.Unmarshal(rawBody, &out); err != nil {
		return "", fmt.Errorf("AI: deserializar respuesta Anthropic: %w", err)
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("AI: Claude devolvió respuesta vacía")
	}
	return strings.TrimSpace(out.Content[0].Text), nil
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "```"); idx != -1 {
		after := text[idx+3:]
		if nl := strings.Index(after, "\n"); nl != -1 {
			after = after[nl+1:]
		}
		if close := strings.LastIndex(after, "```"); close != -1 {
			after = after[:close]
		}
		text = strings.TrimSpace(after)
	}
	if strings.HasPrefix(text, "{") {
		return text
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}
	return text
}

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

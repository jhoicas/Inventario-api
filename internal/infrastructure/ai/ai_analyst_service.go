package ai

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"strings"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// AIAnalystService motor Text-to-SQL + ejecución en PostgreSQL con validación de seguridad.
type AIAnalystService struct {
	llmService    ports.LLMService
	sqlGuard      *SQLGuard
	analyticsRepo repository.AIAnalyticsRepository
	log           *logger.Logger
}

// NewAIAnalystService constructor (usar Anthropic u otro LLM que implemente GenerateTextWithSystem).
func NewAIAnalystService(
	llm ports.LLMService,
	analyticsRepo repository.AIAnalyticsRepository,
	log *logger.Logger,
) *AIAnalystService {
	return &AIAnalystService{
		llmService:    llm,
		sqlGuard:      NewSQLGuard(),
		analyticsRepo: analyticsRepo,
		log:           log,
	}
}

// Ask traduce la pregunta a SQL SELECT, valida, ejecuta y devuelve answer + data + chartType.
func (s *AIAnalystService) Ask(ctx context.Context, companyID, question string) (*dto.CRMTextToSQLResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if question == "" {
		return nil, fmt.Errorf("pregunta vacia")
	}
	if companyID == "" {
		return nil, fmt.Errorf("company_id requerido")
	}

	textToSQL, err := s.generateSQLFromQuestion(ctx, companyID, question)
	if err != nil {
		if s.log != nil {
			s.log.Error().Err(err).Msg("generar SQL desde pregunta")
		}
		return nil, fmt.Errorf("generar SQL: %w", err)
	}

	sqlQuery := textToSQL.SQL
	stdlog.Printf("SQL Generado por Anthropic: %s", sqlQuery)
	if s.log != nil {
		s.log.Info().Str("question", question).Str("generated_sql", sqlQuery).Msg("SQL generado (Text-to-SQL)")
	}

	if err := s.sqlGuard.ValidateQuery(sqlQuery); err != nil {
		if s.log != nil {
			s.log.Warn().Err(err).Str("query", sqlQuery).Msg("SQL rechazado por validacion")
		}
		return nil, fmt.Errorf("SQL no seguro: %w", err)
	}
	if err := assertSingleStatement(sqlQuery); err != nil {
		return nil, err
	}
	if err := assertCompanyFilter(sqlQuery, companyID); err != nil {
		return nil, err
	}

	results, err := s.analyticsRepo.ExecuteRawSelect(ctx, sqlQuery)
	if err != nil {
		stdlog.Printf("Error ejecutando SQL generado: %v", err)
		if s.log != nil {
			s.log.Error().Err(err).Str("sql", sqlQuery).Msg("ejecutar SQL generado")
		}
		return nil, fmt.Errorf("ejecutar SQL: %w", err)
	}

	sanitized := s.sqlGuard.SanitizeResult(results)
	answer := textToSQL.Answer
	if answer == "" {
		answer = s.summarizeAnswer(ctx, question, sanitized)
	}

	if s.log != nil {
		s.log.Info().Int("records", len(sanitized)).Msg("text-to-sql ok")
	}

	return &dto.CRMTextToSQLResponse{
		Answer:    answer,
		Data:      sanitized,
		ChartType: normalizeChartType(textToSQL.ChartType),
	}, nil
}

func buildTextToSQLSystemPrompt(companyID string) string {
	return fmt.Sprintf(`Eres un experto en PostgreSQL. Tu única tarea es convertir la pregunta del usuario en una consulta SQL SELECT válida basada en el esquema provisto.

SEGURIDAD CRÍTICA: Siempre debes incluir un filtro WHERE company_id = '%s' (usa exactamente este UUID entre comillas simples) para asegurar que los datos no se mezclen entre empresas. En JOINs, aplica company_id en las tablas que lo tengan.

Responde ÚNICAMENTE un JSON válido con esta estructura exacta:
{"answer":"<respuesta breve en español>","sql":"<consulta SELECT>","chartType":"bar|pie|line|none"}
No incluyas markdown ni texto adicional.

Esquema:
%s`, companyID, TextToSQLSchemaDescription)
}

type textToSQLOutput struct {
	Answer    string `json:"answer"`
	SQL       string `json:"sql"`
	ChartType string `json:"chartType"`
}

func (s *AIAnalystService) generateSQLFromQuestion(ctx context.Context, companyID, question string) (*textToSQLOutput, error) {
	sys := buildTextToSQLSystemPrompt(companyID)
	user := fmt.Sprintf("Pregunta del usuario:\n%s", strings.TrimSpace(question))

	rawText, err := s.llmService.GenerateTextWithSystem(ctx, sys, user)
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}
	rawText = extractJSON(rawText)

	var out textToSQLOutput
	if err := json.Unmarshal([]byte(rawText), &out); err != nil {
		// fallback de compatibilidad: si el modelo devolvió SQL plano.
		sqlText := cleanGeneratedSQL(rawText)
		if sqlText == "" {
			return nil, fmt.Errorf("respuesta de IA inválida (JSON requerido): %w", err)
		}
		return &textToSQLOutput{Answer: "", SQL: sqlText}, nil
	}

	sqlText := cleanGeneratedSQL(out.SQL)
	if sqlText == "" {
		return nil, fmt.Errorf("el modelo devolvio SQL vacio")
	}
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sqlText)), "SELECT") {
		return nil, fmt.Errorf("el modelo no genero un SELECT valido: %s", sqlText)
	}
	out.SQL = sqlText
	out.Answer = strings.TrimSpace(out.Answer)
	out.ChartType = normalizeChartType(out.ChartType)
	return &out, nil
}

func normalizeChartType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bar":
		return "bar"
	case "pie":
		return "pie"
	case "line":
		return "line"
	default:
		return "none"
	}
}

func cleanGeneratedSQL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```sql")
	s = strings.TrimPrefix(s, "```SQL")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	return strings.TrimSpace(s)
}

func assertSingleStatement(sql string) error {
	parts := strings.Split(sql, ";")
	n := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			n++
		}
	}
	if n != 1 {
		return fmt.Errorf("solo se permite una sentencia SQL")
	}
	return nil
}

func assertCompanyFilter(sql, companyID string) error {
	if companyID == "" {
		return fmt.Errorf("company_id invalido")
	}
	if !strings.Contains(sql, companyID) {
		return fmt.Errorf("el SQL debe incluir el company_id de la sesion (%s)", companyID)
	}
	lower := strings.ToLower(sql)
	if !strings.Contains(lower, "company_id") {
		return fmt.Errorf("el SQL debe referenciar la columna company_id")
	}
	return nil
}

func (s *AIAnalystService) summarizeAnswer(ctx context.Context, question string, data []map[string]interface{}) string {
	if len(data) == 0 {
		return "No se encontraron resultados para tu consulta."
	}
	preview, err := json.Marshal(data)
	if err != nil {
		preview = []byte("[]")
	}
	if len(preview) > 2500 {
		preview = preview[:2500]
	}
	user := fmt.Sprintf("Pregunta: %s\n\nResultado (JSON, posiblemente truncado):\n%s", question, string(preview))
	sys := "Eres un analista de datos. Responde en una sola frase breve en español, sin markdown."
	ans, err := s.llmService.GenerateTextWithSystem(ctx, sys, user)
	if err != nil {
		if s.log != nil {
			s.log.Warn().Err(err).Msg("resumen IA (usando fallback)")
		}
		return fmt.Sprintf("Consulta ejecutada: se devolvieron %d filas.", len(data))
	}
	return strings.TrimSpace(ans)
}

// AggregateQuery ejecuta una consulta de agregacion (COUNT, SUM, AVG) sobre analytics (vista legacy).
func (s *AIAnalystService) AggregateQuery(ctx context.Context, companyID, sqlQuery string) (interface{}, error) {
	if err := s.sqlGuard.ValidateQuery(sqlQuery); err != nil {
		return nil, fmt.Errorf("SQL no seguro: %w", err)
	}

	safeSQLQuery, err := s.sqlGuard.InjectCompanyFilter(sqlQuery, companyID)
	if err != nil {
		return nil, fmt.Errorf("inyectar company_id: %w", err)
	}

	result, err := s.analyticsRepo.RunAggregateQuery(ctx, companyID, safeSQLQuery)
	if err != nil {
		return nil, fmt.Errorf("ejecutar aggregate query: %w", err)
	}

	return result, nil
}

// ExportResultsAsJSON exporta resultados de query en formato JSON limpio.
func (s *AIAnalystService) ExportResultsAsJSON(results []map[string]interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

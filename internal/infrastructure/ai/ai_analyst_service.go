package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// AIAnalystService traduce preguntas en lenguaje natural a SQL sobre v_crm_ai_analytics
// con protecciones automaticas de security (SQL Guard + company_id injection).
type AIAnalystService struct {
	llmService    ports.LLMService
	sqlGuard      *SQLGuard
	analyticsRepo repository.AIAnalyticsRepository
	log           *logger.Logger
}

// NewAIAnalystService constructor.
func NewAIAnalystService(
	llmService ports.LLMService,
	analyticsRepo repository.AIAnalyticsRepository,
	log *logger.Logger,
) *AIAnalystService {
	return &AIAnalystService{
		llmService:    llmService,
		sqlGuard:      NewSQLGuard(),
		analyticsRepo: analyticsRepo,
		log:           log,
	}
}

// Ask traduce una pregunta en lenguaje natural a SQL SELECT sobre v_crm_ai_analytics
// y retorna los datos del analisis. company_id se inyecta automaticamente.
func (s *AIAnalystService) Ask(ctx context.Context, companyID, question string) ([]map[string]interface{}, error) {
	if question == "" {
		return nil, fmt.Errorf("pregunta vacia")
	}
	if companyID == "" {
		return nil, fmt.Errorf("company_id requerido")
	}

	// Paso 1: Generar SQL usando LLM con schema context
	sqlQuery, err := s.generateSQLFromQuestion(ctx, question)
	if err != nil {
		s.log.Error().Err(err).Msg("generar SQL desde pregunta")
		return nil, fmt.Errorf("generar SQL: %w", err)
	}

	s.log.Info().Str("question", question).Str("generated_sql", sqlQuery).Msg("SQL generado del LLM")

	// Paso 2: Validar SQL con SQL Guard (bloquea DELETE, DROP, INSERT, etc)
	if err := s.sqlGuard.ValidateQuery(sqlQuery); err != nil {
		s.log.Warn().Err(err).Str("query", sqlQuery).Msg("SQL rechazado por validacion")
		return nil, fmt.Errorf("SQL no seguro: %w", err)
	}

	// Paso 3: Inyectar company_id obligatorio para aislamiento multi-tenancy
	safeSQLQuery, err := s.sqlGuard.InjectCompanyFilter(sqlQuery, companyID)
	if err != nil {
		s.log.Warn().Err(err).Msg("inyectar filtro company_id")
		return nil, fmt.Errorf("inyectar company_id: %w", err)
	}

	s.log.Info().Str("safe_sql", safeSQLQuery).Msg("SQL con filtro company_id inyectado")

	// Paso 4: Ejecutar query sobre la vista (filas completas o agregados; columnas según el SQL generado)
	results, err := s.analyticsRepo.QueryView(ctx, companyID, safeSQLQuery)
	if err != nil {
		s.log.Error().Err(err).Str("safe_sql", safeSQLQuery).Msg("ejecutar query")
		return nil, fmt.Errorf("ejecutar analitytics query: %w", err)
	}

	// Paso 5: Sanitizar resultados (remover company_id del output al frontend)
	sanitized := s.sqlGuard.SanitizeResult(results)
	s.log.Info().Int("records", len(sanitized)).Msg("analitytics query exitosa")

	return sanitized, nil
}

// generateSQLFromQuestion usa el LLM para traducir pregunta a SQL SELECT.
// Incluye schema context para que el LLM genere queries correctas.
func (s *AIAnalystService) generateSQLFromQuestion(ctx context.Context, question string) (string, error) {
	schemaContext := `You are an expert SQL analyst. Convert the user's question into a single, valid PostgreSQL SELECT statement.
Target VIEW: v_crm_ai_analytics

VIEW COLUMNS:
- company_id: UUID of the company (will be auto-injected)
- fecha: DATE of the sale
- cliente_nombre: Customer name
- ciudad: Customer city
- producto: Product name
- categoria: Product category (often used as "segmento" for product-based segments like VIP)
- cantidad: Quantity sold
- precio_unitario: Unit price
- ingreso_neto: Net revenue (line_total)
- costo_total: Total cost
- utilidad: Profit (ingreso_neto - costo_total)
- customer_email: Customer email
- sale_id: Sale ID
- item_id: Item ID

RULES:
1. Always start with SELECT (no DELETE, INSERT, UPDATE, DROP, CREATE)
2. Always reference columns from v_crm_ai_analytics
3. Return ONLY the SQL statement, no explanation
4. Use column aliases in Spanish if needed
5. Do NOT include company_id in WHERE clause (will be auto-injected)
6. Use appropriate aggregations (SUM, COUNT, AVG) if the question implies it
7. Use date ranges if the question mentions time periods
8. For "how many customers in segment X", use COUNT(DISTINCT customer_email) or COUNT(DISTINCT cliente_nombre) filtered by categoria or similar, as appropriate

Example questions and expected SQL:
Q: "Cual es el ingreso total del mes pasado?"
A: SELECT SUM(ingreso_neto) as total_ingreso FROM v_crm_ai_analytics WHERE fecha >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '1 month')

Q: "Cuales son los productos mas vendidos?"
A: SELECT producto, SUM(cantidad) as total_cantidad FROM v_crm_ai_analytics GROUP BY producto ORDER BY total_cantidad DESC LIMIT 10

User question: %s`

	fullPrompt := fmt.Sprintf(schemaContext, question)

	sqlText, err := s.llmService.GenerateText(ctx, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM generateText: %w", err)
	}

	// Limpiar respuesta que podria incluir ajeno
	sqlText = strings.TrimSpace(sqlText)
	// Remover markdown code blocks si el LLM los aniade
	sqlText = strings.TrimPrefix(sqlText, "```sql")
	sqlText = strings.TrimPrefix(sqlText, "```")
	sqlText = strings.TrimSuffix(sqlText, "```")
	sqlText = strings.TrimSpace(sqlText)

	// Validar que empiece con SELECT despues de limpiar
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sqlText)), "SELECT") {
		return "", fmt.Errorf("LLM no genero un SELECT valido: %s", sqlText)
	}

	return sqlText, nil
}

// AggregateQuery ejecuta una consulta de agregacion (COUNT, SUM, AVG) sobre analytics.
// Util para dashboards y resumenes rapidos.
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

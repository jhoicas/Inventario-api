package ai

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLGuard implementacion del servicio de validacion y sanitizacion de queries SQL
// Previene inyeccion SQL, acceso cross-company, y operaciones peligrosas.
type SQLGuard struct {
	// Palabras clave bloqueadas que no deben aparecer en queries generadas
	blockedKeywords []string
	// Expresion regular para detectar parametros peligrosos
	dangerousPatterns []*regexp.Regexp
}

// NewSQLGuard constructor.
func NewSQLGuard() *SQLGuard {
	return &SQLGuard{
		blockedKeywords: []string{
			"DELETE", "INSERT", "UPDATE", "DROP", "TRUNCATE", "ALTER",
			"GRANT", "REVOKE", "CREATE", "REPLACE", "CASCADE",
		},
		dangerousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i);\s*DELETE`),
			regexp.MustCompile(`(?i);\s*INSERT`),
			regexp.MustCompile(`(?i);\s*DROP`),
			regexp.MustCompile(`(?i)--\s*`), // SQL comments
			regexp.MustCompile(`(?i)/\*`),   // Multi-line comments
		},
	}
}

// ValidateQuery verifica que la consulta sea SELECT-only y no contenga palabras clave peligrosas.
func (sg *SQLGuard) ValidateQuery(query string) error {
	if query == "" {
		return fmt.Errorf("consulta vacia")
	}

	// Normalizar espacios y mayusculas para validacion
	normalized := strings.TrimSpace(query)
	upper := strings.ToUpper(normalized)

	// Verificar que comience con SELECT
	if !strings.HasPrefix(upper, "SELECT") {
		return fmt.Errorf("consulta debe comenzar con SELECT; encontrado: %s", string([]rune(upper)[:10]))
	}

	// Verificar palabras clave bloqueadas
	for _, kw := range sg.blockedKeywords {
		// Buscar palabra completa (bounded by space, comma, paren)
		pattern := fmt.Sprintf(`\b%s\b`, kw)
		re := regexp.MustCompile(pattern)
		if re.MatchString(upper) {
			return fmt.Errorf("palabra clave bloqueada detectada: %s", kw)
		}
	}

	// Verificar patrones peligrosos (inyeccion SQL multiples sentencias, comentarios)
	for _, pattern := range sg.dangerousPatterns {
		if pattern.MatchString(normalized) {
			return fmt.Errorf("patron peligroso detectado en consulta: %s", pattern.String())
		}
	}

	return nil
}

// InjectCompanyFilter inyecta un filtro company_id obligatorio en la consulta
// para evitar leakage de datos entre empresas en ambiente multitenancy.
// La funcion busca la clausula WHERE existente y agrega AND company_id = companyID
// Si no hay WHERE, agrega una nueva.
func (sg *SQLGuard) InjectCompanyFilter(query, companyID string) (string, error) {
	if query == "" || companyID == "" {
		return "", fmt.Errorf("query y companyID son obligatorios")
	}

	// Primero valida que sea SELECT-safe
	if err := sg.ValidateQuery(query); err != nil {
		return "", err
	}

	// Buscar patron "FROM ... WHERE" (case-insensitive)
	upperQuery := strings.ToUpper(query)
	whereIdx := strings.LastIndex(upperQuery, "WHERE")

	var result string
	if whereIdx != -1 {
		// Existe WHERE: agregar AND company_id
		result = fmt.Sprintf("%s AND company_id = '%s'", query, escapeSQLString(companyID))
	} else {
		// No existe WHERE: agregar WHERE al final (antes de ORDER BY / LIMIT si existen)
		orderByIdx := strings.LastIndex(upperQuery, "ORDER BY")
		limitIdx := strings.LastIndex(upperQuery, "LIMIT")

		var insertIdx int
		if orderByIdx != -1 {
			insertIdx = orderByIdx
		} else if limitIdx != -1 {
			insertIdx = limitIdx
		} else {
			insertIdx = len(query)
		}

		result = fmt.Sprintf("%s WHERE company_id = '%s'", query[:insertIdx], escapeSQLString(companyID))
		if insertIdx < len(query) {
			result += " " + query[insertIdx:]
		}
	}

	return result, nil
}

// SanitizeResult elimina campos sensibles del resultado antes de retornarlos al usuario.
// Bloquea acceso a campos internos como company_id en niveles de representacion.
func (sg *SQLGuard) SanitizeResult(rows []map[string]interface{}) []map[string]interface{} {
	if len(rows) == 0 {
		return rows
	}

	// Campos sensibles que no se deben enviar al frontend
	sensitive := map[string]bool{
		"company_id": true,
		"id":         false, // IDs estan permitidos (necesarios para referencias)
	}

	sanitized := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		clean := make(map[string]interface{})
		for k, v := range row {
			if !sensitive[k] {
				clean[k] = v
			}
		}
		sanitized[i] = clean
	}

	return sanitized
}

// escapeSQLString escapa caracteres especiales en strings SQL para prevenir inyeccion.
// Nota: En produccion, usar prepared statements con parametros es mas seguro.
// Esta funcion es una capa adicional de defensa.
func escapeSQLString(s string) string {
	// Escapar single quotes duplicandolas (SQL standard)
	escaped := strings.ReplaceAll(s, "'", "''")
	// Bloquear intentos de escape con unicode o caracteres especiales
	filtered := ""
	for _, c := range escaped {
		if c >= 32 && c < 127 {
			filtered += string(c)
		}
	}
	return filtered
}

package ports

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// AIAnalystService define el puerto para traducir preguntas en lenguaje natural a consultas SQL
// (Text-to-SQL sobre el esquema CRM) con protecciones de seguridad incluidas.
type AIAnalystService interface {
	// Ask traduce una pregunta a SQL SELECT, ejecuta en PostgreSQL y devuelve respuesta estructurada.
	Ask(ctx context.Context, companyID, question string) (*dto.CRMTextToSQLResponse, error)
}

// SQLGuardService define el contrato de seguridad para prevenir inyeccion SQL y leakage de datos
// Valida y sanitiza consultas generadas por el LLM antes de ejecutarse.
type SQLGuardService interface {
	// ValidateQuery verifica que la consulta:
	// 1. Comience con SELECT (bloquea DELETE, UPDATE, INSERT, DROP)
	// 2. No contenga palabras clave peligrosas
	// 3. Retorne error si se detectan violaciones
	ValidateQuery(query string) error

	// InjectCompanyFilter inyecta un WHERE company_id = ? obligatorio
	// para evitar filtración de datos entre empresas multitenancy.
	InjectCompanyFilter(query, companyID string) (string, error)

	// SanitizeResult elimina campos sensibles del resultado antes de retornarlos al usuario
	SanitizeResult(rows []map[string]interface{}) []map[string]interface{}
}

// BulkImporterService define el contrato para importar datos en lotes desde CSV/Excel
// hacia las tablas hub con validacion e idempotencia.
type BulkImporterService interface {
	// ImportFromCSV lee un archivo CSV y carga los datos en lotes de 1000 registros
	// hacia crm_products_hub, crm_sales_hub y crm_sale_items_hub.
	// Auto-crea productos que no existan en el hub.
	ImportFromCSV(ctx context.Context, companyID, filePath string, tableName string) (recordsImported int, err error)

	// ImportFromExcel lee un archivo Excel y carga los datos en lotes
	// Soporta multiples hojas (sheets) y mapeo flexible de columnas.
	ImportFromExcel(ctx context.Context, companyID, filePath string, sheetName string, tableName string) (recordsImported int, err error)

	// ValidateImportData valida que los registros cumplan con esquema esperado
	// Retorna lista de errores de validacion o nil si todo es correcto.
	ValidateImportData(records []map[string]interface{}, tableName string) []error
}

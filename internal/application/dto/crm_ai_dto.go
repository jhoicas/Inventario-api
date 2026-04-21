package dto

// CRMTextToSQLResponse respuesta de POST /api/crm/ai/ask (motor Text-to-SQL).
type CRMTextToSQLResponse struct {
	Answer    string                   `json:"answer"`
	Data      []map[string]interface{} `json:"data"`
	ChartType string                   `json:"chartType"`
}

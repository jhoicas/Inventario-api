package dto

import "time"

// ImportCRMProfileRequest es la fila de datos parseada del archivo de importación.
type ImportCRMProfileRequest struct {
	Nombre                string  `json:"nombre"`
	Email                 string  `json:"email"`
	Segmento              string  `json:"segmento"`
	TotalComprado         float64 `json:"total_comprado"`
	CategoriaPrincipal    string  `json:"categoria_principal"`
	ProductosComprados    string  `json:"productos_comprados"`
	AccionRemarketingType string  `json:"accion_remarketing"`
}

// CRMImportResponse resume el resultado de la importación masiva.
type CRMImportResponse struct {
	TotalRows    int           `json:"total_rows"`
	CreatedCount int           `json:"created_count"`
	UpdatedCount int           `json:"updated_count"`
	SkippedCount int           `json:"skipped_count"`
	Errors       []ImportError `json:"errors,omitempty"`
	ProcessedAt  time.Time     `json:"processed_at"`
	CompanyID    string        `json:"company_id"`
}

// ImportError captura los errores por fila durante la importación.
type ImportError struct {
	Row     int    `json:"row"`
	Email   string `json:"email,omitempty"`
	Message string `json:"message"`
}

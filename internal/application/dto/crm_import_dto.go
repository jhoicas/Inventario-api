package dto

import "time"

// ImportCRMProfileRequest es la fila de datos parseada del archivo de importación.
type ImportCRMProfileRequest struct {
	Name                  string     `json:"nombre"`
	Email                 string     `json:"email"`
	Phone                 string     `json:"telefono"`
	TaxID                 string     `json:"documento"`
	FechaNacimiento       string     `json:"fecha_nacimiento,omitempty"`
	BirthDate             *time.Time `json:"-"`
	CategoryName          string     `json:"category_name"`
	Segmento              string     `json:"segmento"`
	VentasTotales         float64    `json:"ventasTotales"`
	Pedidos               int        `json:"pedidos"`
	Productos             int        `json:"productos"`
	UltimaCompra          string     `json:"ultimaCompra"`
	CategoriaProducto     string     `json:"categoriaProducto"`
	DescripcionProductos  string     `json:"descripcionProductos"`
	EstrategiaSeguimiento string     `json:"estrategiaSeguimiento"`
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

// ImportPreviewRow describe una fila validada del archivo de importación.
type ImportPreviewRow struct {
	Row             int      `json:"row"`
	Email           string   `json:"email,omitempty"`
	NormalizedEmail string   `json:"normalized_email,omitempty"`
	FechaNacimiento string   `json:"fecha_nacimiento,omitempty"`
	Valid           bool     `json:"valid"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	IDCliente       string   `json:"id_cliente,omitempty"`
	LastPurchase    string   `json:"last_purchase,omitempty"`
}

// ImportJobRowStatus describe el estado final o intermedio de una fila durante el job.
type ImportJobRowStatus struct {
	Row             int      `json:"row"`
	Email           string   `json:"email,omitempty"`
	NormalizedEmail string   `json:"normalized_email,omitempty"`
	FechaNacimiento string   `json:"fecha_nacimiento,omitempty"`
	Valid           bool     `json:"valid"`
	Action          string   `json:"action"` // skipped|pending|inserted|updated|failed
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	IDCliente       string   `json:"id_cliente,omitempty"`
	LastPurchase    string   `json:"last_purchase,omitempty"`
}

// ImportPreviewSummary resume el resultado del análisis previo al submit.
type ImportPreviewSummary struct {
	TotalRows        int `json:"total_rows"`
	ValidRows        int `json:"valid_rows"`
	InvalidRows      int `json:"invalid_rows"`
	DuplicateRows    int `json:"duplicate_rows"`
	MissingEmailRows int `json:"missing_email_rows"`
	WarningRows      int `json:"warning_rows"`
}

// CRMImportPreviewResponse devuelve el preview del archivo antes de importar.
type CRMImportPreviewResponse struct {
	Summary ImportPreviewSummary `json:"summary"`
	Rows    []ImportPreviewRow   `json:"rows"`
}

// ImportSalesError representa un error encontrado al procesar una orden.
type ImportSalesError struct {
	OrderNumber string `json:"order_number"`
	Message     string `json:"message"`
}

// ImportSalesResponse resume el resultado de la importación masiva de ventas.
type ImportSalesResponse struct {
	TotalOrders       int                `json:"total_orders"`
	SuccessfulOrders  int                `json:"successful_orders"`
	FailedOrders      int                `json:"failed_orders"`
	TotalItems        int                `json:"total_items"`
	CreatedCustomers  int                `json:"created_customers"`
	CreatedCategories int                `json:"created_categories"`
	CreatedProducts   int                `json:"created_products"`
	Errors            []ImportSalesError `json:"errors,omitempty"`
}

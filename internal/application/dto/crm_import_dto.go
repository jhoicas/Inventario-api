package dto

import "time"

// ImportCRMProfileRequest es la fila de datos parseada del archivo de importación.
type ImportCRMProfileRequest struct {
	IDCliente             string  `json:"idCliente"`
	Nombre                string  `json:"nombre"`
	Email                 string  `json:"email"`
	Segmento              string  `json:"segmento"`
	VentasTotales         float64 `json:"ventasTotales"`
	Pedidos               int     `json:"pedidos"`
	Productos             int     `json:"productos"`
	UltimaCompra          string  `json:"ultimaCompra"`
	CategoriaProducto     string  `json:"categoriaProducto"`
	DescripcionProductos  string  `json:"descripcionProductos"`
	EstrategiaSeguimiento string  `json:"estrategiaSeguimiento"`
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

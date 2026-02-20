package dto

// PageRequest paginación para listados.
type PageRequest struct {
	Limit  int `query:"limit" validate:"min=1,max=100"`
	Offset int `query:"offset" validate:"min=0"`
}

// DefaultPage aplica valores por defecto si Limit/Offset son cero.
func (p *PageRequest) DefaultPage() {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// PageResponse metadatos de página en respuestas.
type PageResponse struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total,omitempty"`
}

// ErrorResponse cuerpo de error HTTP.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

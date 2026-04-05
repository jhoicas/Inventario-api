package dto

import "time"

// CreateCompanyRequest entrada para crear una empresa.
type CreateCompanyRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=200"`
	NIT     string `json:"nit" validate:"required,min=1,max=20"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email" validate:"omitempty,email"`
}

// UpdateCompanyRequest entrada para actualizar una empresa (campos opcionales).
type UpdateCompanyRequest struct {
	Name    *string `json:"name" validate:"omitempty,min=1,max=200"`
	NIT     *string `json:"nit" validate:"omitempty,min=1,max=20"`
	Address *string `json:"address"`
	Phone   *string `json:"phone"`
	Email   *string `json:"email" validate:"omitempty,email"`
	Status  *string `json:"status" validate:"omitempty,oneof=active suspended inactive"`
}

// CompanyResponse salida de una empresa (sin datos sensibles).
type CompanyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NIT       string    `json:"nit"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CompanyListResponse lista paginada de empresas.
type CompanyListResponse struct {
	Items []CompanyResponse `json:"items"`
	Page  PageResponse      `json:"page"`
}

// CreateResolutionRequest entrada para crear resolución DIAN por empresa.
type CreateResolutionRequest struct {
	Prefix           string `json:"prefix"`
	ResolutionNumber string `json:"resolution_number"`
	FromNumber       int64  `json:"from_number"`
	ToNumber         int64  `json:"to_number"`
	CurrentNumber    int64  `json:"current_number,omitempty"`  // no usado por ahora; solo para compatibilidad de payload
	ValidFrom        string `json:"valid_from"`                // formato YYYY-MM-DD
	ValidUntil       string `json:"valid_to"`                  // formato YYYY-MM-DD (respetar nombre del frontend)
	AlertThreshold   int    `json:"alert_threshold,omitempty"` // porcentaje; por ahora solo compatibilidad, cálculo interno sigue siendo 10%
	Environment      string `json:"environment,omitempty"`     // test|prod; opcional en este payload
}

// ResolutionResponse salida de resolución con alerta de umbral.
type ResolutionResponse struct {
	ID               string    `json:"id"`
	CompanyID        string    `json:"company_id"`
	Prefix           string    `json:"prefix"`
	ResolutionNumber string    `json:"resolution_number"`
	FromNumber       int64     `json:"from_number"`
	ToNumber         int64     `json:"to_number"`
	ValidFrom        time.Time `json:"valid_from"`
	ValidUntil       time.Time `json:"valid_until"`
	Environment      string    `json:"environment"`
	AlertThreshold   bool      `json:"alert_threshold"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CompanyModuleResponse salida ligera de módulos SaaS por empresa.
type CompanyModuleResponse struct {
	ID          string     `json:"id,omitempty"`
	ModuleName  string     `json:"module_name"`
	IsActive    bool       `json:"is_active"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// CreateCompanyModuleRequest entrada para crear/asignar un módulo a una empresa.
type CreateCompanyModuleRequest struct {
	ModuleName string `json:"module_name"`
	IsActive   *bool  `json:"is_active,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"` // RFC3339 opcional
}

// UpdateCompanyModuleRequest entrada para actualizar estado/vencimiento de un módulo.
type UpdateCompanyModuleRequest struct {
	IsActive  *bool  `json:"is_active,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"` // RFC3339 opcional; "" limpia vencimiento
}

// ToggleCompanyModuleRequest entrada para activar/desactivar rápidamente un módulo.
type ToggleCompanyModuleRequest struct {
	IsActive bool `json:"is_active"`
}

// CompanyModulesResponse respuesta de GET /api/companies/{id}/modules.
type CompanyModulesResponse struct {
	CompanyID string                  `json:"company_id"`
	Modules   []CompanyModuleResponse `json:"modules"`
}

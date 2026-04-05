package dto

import "time"

// CreateCompanyScreenRequest habilita o crea una pantalla para una empresa.
type CreateCompanyScreenRequest struct {
	ScreenID string `json:"screen_id" validate:"required,uuid"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// UpdateCompanyScreenRequest actualiza el estado de una pantalla para una empresa.
type UpdateCompanyScreenRequest struct {
	IsActive *bool `json:"is_active,omitempty"`
}

// ReplaceCompanyScreensRequest reemplaza en bloque las pantallas de una empresa.
type ReplaceCompanyScreensRequest struct {
	ScreenIDs []string `json:"screen_ids"`
}

// CompanyScreenResponse representa una pantalla habilitada para una empresa.
type CompanyScreenResponse struct {
	ID            string     `json:"id,omitempty"`
	CompanyID     string     `json:"company_id"`
	ScreenID      string     `json:"screen_id"`
	ScreenKey     string     `json:"screen_key"`
	ScreenName    string     `json:"screen_name"`
	ModuleKey     string     `json:"module_key,omitempty"`
	ModuleName    string     `json:"module_name,omitempty"`
	FrontendRoute string     `json:"frontend_route,omitempty"`
	ApiEndpoint   string     `json:"api_endpoint,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

// CompanyScreensResponse lista pantallas habilitadas para una empresa.
type CompanyScreensResponse struct {
	CompanyID string                  `json:"company_id"`
	Screens   []CompanyScreenResponse `json:"screens"`
}

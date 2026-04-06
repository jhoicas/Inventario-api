package dto

// CreateScreenRequest representa la creación de una pantalla del catálogo.
type CreateScreenRequest struct {
	ModuleID          string `json:"module_id"`
	Key               string `json:"key"`
	Name              string `json:"name"`
	FrontendRoute     string `json:"frontend_route"`
	ApiEndpoint       string `json:"api_endpoint"`
	Order             int    `json:"order"`
	IsActive          *bool  `json:"is_active,omitempty"`
	ModuleKeySnapshot string `json:"module_key_snapshot"`
}

// UpdateScreenRequest representa la actualización de una pantalla del catálogo.
type UpdateScreenRequest struct {
	ModuleID      string `json:"module_id"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	FrontendRoute string `json:"frontend_route"`
	ApiEndpoint   string `json:"api_endpoint"`
	Order         int    `json:"order"`
	IsActive      *bool  `json:"is_active,omitempty"`
}

// ScreenAdminResponse representa una pantalla para administración de super admin.
type ScreenAdminResponse struct {
	ID                string `json:"id"`
	ModuleID          string `json:"module_id"`
	Key               string `json:"key"`
	Name              string `json:"name"`
	FrontendRoute     string `json:"frontend_route"`
	ApiEndpoint       string `json:"api_endpoint"`
	Order             int    `json:"order"`
	IsActive          bool   `json:"is_active"`
	ModuleKeySnapshot string `json:"module_key_snapshot"`
}

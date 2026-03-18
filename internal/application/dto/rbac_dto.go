package dto

import "time"

// RoleResponse representa un rol expuesto al frontend.
type RoleResponse struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ScreenResponse representa una pantalla dentro de un módulo.
type ScreenResponse struct {
	ID            string `json:"id"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	FrontendRoute string `json:"frontend_route"`
	ApiEndpoint   string `json:"api_endpoint"`
	Order         int    `json:"order"`
}

// ModuleResponse representa un módulo con sus pantallas permitidas.
type ModuleResponse struct {
	ID      string           `json:"id"`
	Key     string           `json:"key"`
	Name    string           `json:"name"`
	Icon    string           `json:"icon"`
	Order   int              `json:"order"`
	Screens []ScreenResponse `json:"screens"`
}

// RBACMenuResponse devuelve el menú jerárquico permitido para un rol.
type RBACMenuResponse struct {
	RoleID   string           `json:"role_id"`
	RoleKey  string           `json:"role_key"`
	RoleName string           `json:"role_name"`
	Modules  []ModuleResponse `json:"modules"`
}

// RBACCatalogResponse devuelve el catálogo completo de módulos y pantallas.
type RBACCatalogResponse struct {
	Modules []ModuleResponse `json:"modules"`
}

// UpdateRoleScreensRequest reemplaza las pantallas asignadas a un rol.
type UpdateRoleScreensRequest struct {
	ScreenIDs []string `json:"screen_ids"`
}


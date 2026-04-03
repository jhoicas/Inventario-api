package entity

import "time"

// Role representa un rol funcional del sistema.
type Role struct {
	ID          string
	Key         string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Module representa un módulo visible en el menú principal.
type Module struct {
	ID        string
	Key       string
	Name      string
	Icon      string
	Order     int
	IsActive  bool
	Screens   []Screen
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Screen representa una pantalla/entrada navegable y su endpoint asociado.
type Screen struct {
	ID                   string
	ModuleID             string
	ModuleKey            string
	ModuleName           string
	ModuleKeySnapshot    string
	Key                  string
	Name                 string
	FrontendRoute        string
	ApiEndpoint          string
	Order                int
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ModuleClassification string
}

// RoleScreen es la tabla pivote entre roles y pantallas.
type RoleScreen struct {
	RoleID    string
	ScreenID  string
	CreatedAt time.Time
}

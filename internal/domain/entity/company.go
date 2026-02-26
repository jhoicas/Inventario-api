package entity

import "time"

// Company representa una organización/tenant del sistema (multi-tenant, enfoque Colombia).
type Company struct {
	ID        string
	Name      string
	NIT       string    // NIT colombiano (con o sin dígito de verificación)
	Address   string
	Phone     string
	Email     string
	Status    string    // active, suspended, inactive
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Módulos SaaS disponibles (deben coincidir con el CHECK de la tabla company_modules).
const (
	ModuleInventory  = "inventory"
	ModuleBilling    = "billing"
	ModuleCRM        = "crm"
	ModuleAnalytics  = "analytics"
	ModulePurchasing = "purchasing"
)

// CompanyModule representa la activación de un módulo SaaS en una empresa.
type CompanyModule struct {
	ID          string
	CompanyID   string
	ModuleName  string    // ver constantes Module*
	IsActive    bool
	ActivatedAt time.Time
	ExpiresAt   *time.Time // nil = sin vencimiento
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

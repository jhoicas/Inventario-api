package entity

import "time"

// Company representa una organización/tenant del sistema (multi-tenant, enfoque Colombia).
type Company struct {
	ID          string
	Name        string
	NIT         string // NIT colombiano (con o sin dígito de verificación)
	Address     string
	Phone       string
	Email       string
	Status      string // active, suspended, inactive
	Environment string // "habilitacion" o "produccion" para DIAN
	CertHab     string // Certificado para ambiente de habilitación (PEM o base64)
	CertProd    string // Certificado para ambiente de producción (PEM o base64)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DianConfig retorna la configuración de DIAN según el environment de la empresa.
func (c *Company) DianConfig() (url, cert string) {
	switch c.Environment {
	case "produccion":
		return "https://vpfe.dian.gov.co/WcfDianCustomerServices.svc", c.CertProd
	default:
		return "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc", c.CertHab
	}
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
	ModuleName  string // ver constantes Module*
	IsActive    bool
	ActivatedAt time.Time
	ExpiresAt   *time.Time // nil = sin vencimiento
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CompanyScreen representa la habilitación de una pantalla para una empresa.
type CompanyScreen struct {
	ID            string
	CompanyID     string
	ScreenID      string
	ScreenKey     string
	ScreenName    string
	ModuleKey     string
	ModuleName    string
	FrontendRoute string
	ApiEndpoint   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

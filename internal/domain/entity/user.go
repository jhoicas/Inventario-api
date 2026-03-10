package entity

import "time"

// Roles válidos para User.
const (
	RoleAdmin     = "admin"
	RoleBodeguero = "bodeguero" // legado inventario
	RoleVendedor  = "vendedor"  // legado facturación

	RoleMarketing = "marketing"
	RoleSales     = "sales"
	RoleSupport   = "support"
)

// User representa un usuario del sistema (pertenece a una Company).
type User struct {
	ID           string
	CompanyID    string
	Email        string
	PasswordHash string    // bcrypt hash, nunca plano en dominio después de persistir
	Name         string
	Roles        []string  // lista de roles (p.ej. ["admin","marketing"])
	Status       string    // active, inactive, suspended
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

package entity

import "time"

// Roles válidos para User.
const (
	RoleAdmin     = "admin"
	RoleBodeguero = "bodeguero"
	RoleVendedor  = "vendedor"
)

// User representa un usuario del sistema (pertenece a una Company).
type User struct {
	ID           string
	CompanyID    string
	Email        string
	PasswordHash string    // bcrypt hash, nunca plano en dominio después de persistir
	Name         string
	Role         string    // admin, bodeguero, vendedor
	Status       string    // active, inactive, suspended
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

package dto

import "time"

// CreateUserRequest entrada para crear un usuario (password en texto, se hashea en use case).
type CreateUserRequest struct {
	CompanyID string `json:"company_id" validate:"required,uuid"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Name      string `json:"name" validate:"required,min=1,max=200"`
	Role      string `json:"role" validate:"required,oneof=admin bodeguero vendedor"`
}

// RegisterRequest entrada para registro (auth): email, password, company_id.
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	CompanyID string `json:"company_id" validate:"required,uuid"`
	Name      string `json:"name" validate:"omitempty,max=200"`
	Role      string `json:"role" validate:"omitempty,oneof=admin bodeguero vendedor"`
}

// UserResponse salida de un usuario (sin password).
type UserResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest entrada para login (email + company context opcional después).
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse salida con token JWT (se implementará cuando tengamos JWT).
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

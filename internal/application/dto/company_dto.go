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
	Page  PageResponse     `json:"page"`
}

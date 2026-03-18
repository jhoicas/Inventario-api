package entity

import "time"

// CRMBenefit representa un beneficio asociado a una categoría de fidelización.
type CRMBenefit struct {
	ID          string
	CompanyID   string
	CategoryID  string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

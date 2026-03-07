package entity

import "time"

// CRMBenefit representa un beneficio asociado a una categoría de fidelización.
type CRMBenefit struct {
	ID          string
	CompanyID   string
	CategoryID  string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

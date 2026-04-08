package entity

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// ProfileMetadata guarda datos agregados para importaciones avanzadas y segmentación.
type ProfileMetadata struct {
	OrdersCount      int    `json:"ordersCount"`
	DistinctProducts int    `json:"distinctProducts"`
	LastPurchaseDate string `json:"lastPurchaseDate"`
	MainCategory     string `json:"mainCategory"`
	ProductsList     string `json:"productsList"`
	FollowUpStrategy string `json:"followUpStrategy"`
}

// CRMCustomerProfile extiende al cliente con datos de fidelización (categoría y LTV).
type CRMCustomerProfile struct {
	ID         string
	CustomerID string
	CompanyID  string
	CategoryID string          // nullable si no asignado
	LTV        decimal.Decimal // lifetime value
	Metadata   ProfileMetadata
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MarshalJSON asegura compatibilidad en caso de necesitar serialización directa.
func (m ProfileMetadata) MarshalJSON() ([]byte, error) {
	type alias ProfileMetadata
	return json.Marshal(alias(m))
}

// Profile360 agrupa datos del cliente y su perfil CRM para vista 360 (JOIN customers + crm_customer_profiles).
type Profile360 struct {
	Customer   Customer
	ProfileID  string
	CategoryID string
	LTV        decimal.Decimal
	Metadata   ProfileMetadata
}

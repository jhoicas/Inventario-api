package entity

import "time"

// InteractionType tipo de interacción CRM.
type InteractionType string

const (
	InteractionTypeCall    InteractionType = "call"
	InteractionTypeEmail   InteractionType = "email"
	InteractionTypeMeeting InteractionType = "meeting"
	InteractionTypeOther   InteractionType = "other"
)

// CRMInteraction representa una interacción con el cliente (llamada, email, etc.).
type CRMInteraction struct {
	ID         string
	CompanyID  string
	CustomerID string
	Type       InteractionType
	Subject    string
	Body       string
	CreatedBy  string
	CreatedAt  time.Time
}

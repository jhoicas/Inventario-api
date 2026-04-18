package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	CampaignStatusPending   = "pending"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusSending   = "sending"
	CampaignStatusCompleted = "completed"

	// CampaignRecipientStatusQueued indica que el destinatario quedó encolado para envío.
	CampaignRecipientStatusQueued = "QUEUED"
)

// CRMCampaign representa una campaña de marketing CRM.
type CRMCampaign struct {
	ID          string
	CompanyID   string
	Name        string
	Description string
	Status      string
	Channel     string
	ScheduledAt *time.Time
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Campaign se mantiene como alias por compatibilidad con el código existente.
type Campaign = CRMCampaign

// CampaignMetrics métricas de envío y conversión de una campaña.
type CampaignMetrics struct {
	CampaignID string
	Sent       int
	Opened     int
	Clicked    int
	Converted  int
	Revenue    decimal.Decimal
}

// CampaignTemplate representa una plantilla de campaña de email CRM.
type CampaignTemplate struct {
	ID        string
	CompanyID string
	Name      string
	Subject   string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CampaignRecipient representa un destinatario encolado para envío masivo.
type CampaignRecipient struct {
	ID          string
	CampaignID  string
	CustomerID  string
	CompanyID   string
	Email       string
	Subject     string
	Body        string
	Status      string
	Error       string
	QueuedAt    time.Time
	SentAt      *time.Time
	ProcessedAt *time.Time
}

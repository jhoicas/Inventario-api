package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// CampaignStatus estado de una campaña de marketing CRM.
type CampaignStatus string

const (
	CampaignStatusBorrador   CampaignStatus = "BORRADOR"
	CampaignStatusProgramada CampaignStatus = "PROGRAMADA"
	CampaignStatusEnviando   CampaignStatus = "ENVIANDO"
	CampaignStatusCompletada CampaignStatus = "COMPLETADA"
)

// Campaign representa una campaña de marketing CRM.
type Campaign struct {
	ID          string
	CompanyID   string
	Name        string
	Description string
	Status      CampaignStatus
	ScheduledAt *time.Time
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

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

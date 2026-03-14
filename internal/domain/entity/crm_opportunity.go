package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// OpportunityStage etapa del embudo de ventas de una oportunidad CRM.
type OpportunityStage string

const (
	OpportunityStageProspecto   OpportunityStage = "prospecto"
	OpportunityStageCalificado  OpportunityStage = "calificado"
	OpportunityStagePropuesta   OpportunityStage = "propuesta"
	OpportunityStageNegociacion OpportunityStage = "negociacion"
	OpportunityStageGanado      OpportunityStage = "ganado"
	OpportunityStagePerdido     OpportunityStage = "perdido"
)

// Opportunity representa una oportunidad de negocio en el CRM.
type Opportunity struct {
	ID                string
	CompanyID         string
	CustomerID        string
	Title             string
	Amount            decimal.Decimal
	Probability       int // 0–100
	Stage             OpportunityStage
	ExpectedCloseDate time.Time
	CreatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

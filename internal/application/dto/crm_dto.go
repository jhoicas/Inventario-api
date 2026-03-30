package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// CreateTaskRequest body para crear una tarea CRM.
type CreateTaskRequest struct {
	CustomerID  string     `json:"customer_id"`
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
}

// CreateTaskFromEmailRequest body para crear una tarea CRM desde un correo.
type CreateTaskFromEmailRequest struct {
	EmailID     string    `json:"email_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	AssignedTo  string    `json:"assigned_to"`
}

// UpdateTaskRequest body para actualizar una tarea.
type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	DueAt       *time.Time `json:"due_at"`
	Status      *string    `json:"status"` // pending, done, cancelled
}

// CreateInteractionRequest body para registrar una interacción.
type CreateInteractionRequest struct {
	CustomerID string `json:"customer_id" validate:"required"`
	Type       string `json:"type" validate:"required"` // call, email, meeting, other
	Subject    string `json:"subject"`
	Body       string `json:"body"`
}

// CreateTicketRequest body para radicar un ticket PQR.
type CreateTicketRequest struct {
	CustomerID  string `json:"customer_id" validate:"required"`
	Subject     string `json:"subject" validate:"required"`
	Description string `json:"description" validate:"required"`
}

// CreateTicketFromEmailRequest body para crear un ticket CRM desde un correo.
type CreateTicketFromEmailRequest struct {
	EmailID     string `json:"email_id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	SenderEmail string `json:"sender_email"`
	Priority    string `json:"priority"`
	AssignedTo  string `json:"assigned_to,omitempty"`
}

// UpdateTicketRequest body para actualizar un ticket.
type UpdateTicketRequest struct {
	Subject     *string `json:"subject"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Sentiment   *string `json:"sentiment"`
}

// AssignCategoryRequest asigna o actualiza la categoría de fidelización del cliente.
type AssignCategoryRequest struct {
	CategoryID string          `json:"category_id"`
	LTV        decimal.Decimal `json:"ltv"`
}

// AwardPointsRequest body para acreditar puntos al cliente.
type AwardPointsRequest struct {
	CustomerID  string `json:"customer_id"`
	Points      int    `json:"points"`
	Reason      string `json:"reason"`
	ReferenceID string `json:"reference_id"`
}

// RedeemPointsRequest body para redimir puntos del cliente.
type RedeemPointsRequest struct {
	CustomerID string `json:"customer_id"`
	Points     int    `json:"points"`
	Reason     string `json:"reason"`
}

// CreateBenefitRequest body para crear un beneficio en una categoría.
type CreateBenefitRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Description string `json:"description"`
}

// UpdateBenefitRequest body para actualizar un beneficio.
type UpdateBenefitRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Description string `json:"description"`
}

// Profile360Response vista 360 del cliente (datos base + perfil CRM).
type Profile360Response struct {
	Customer     CustomerResponse  `json:"customer"`
	ProfileID    string            `json:"profile_id"`
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name,omitempty"`
	LTV          decimal.Decimal   `json:"ltv"`
	Benefits     []BenefitResponse `json:"benefits,omitempty"`
}

// TaskResponse tarea en respuestas.
type TaskResponse struct {
	ID          string     `json:"id"`
	CompanyID   string     `json:"company_id"`
	CustomerID  string     `json:"customer_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
	Status      string     `json:"status"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TaskResponseList lista paginada de tareas.
type TaskResponseList struct {
	Items  []TaskResponse `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// InteractionResponse interacción en respuestas.
type InteractionResponse struct {
	ID         string    `json:"id"`
	CompanyID  string    `json:"company_id"`
	CustomerID string    `json:"customer_id"`
	Type       string    `json:"type"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// InteractionListResponse lista paginada de interacciones.
type InteractionListResponse struct {
	Items []InteractionResponse `json:"items"`
	Total int64                 `json:"total"`
}

// PointEventDTO evento de puntos de fidelización.
type PointEventDTO struct {
	Points      int       `json:"points"`
	Reason      string    `json:"reason"`
	ReferenceID string    `json:"reference_id,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// LoyaltyBalanceDTO balance actual de puntos, tier y umbral siguiente.
type LoyaltyBalanceDTO struct {
	Balance           int             `json:"balance"`
	Tier              string          `json:"tier"`
	NextTierThreshold int             `json:"next_tier_threshold"`
	History           []PointEventDTO `json:"history"`
}

// TicketResponse ticket PQR en respuestas.
type TicketResponse struct {
	ID               string    `json:"id"`
	CompanyID        string    `json:"company_id"`
	CustomerID       string    `json:"customer_id"`
	Subject          string    `json:"subject"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	Sentiment        string    `json:"sentiment"`
	EscalationReason string    `json:"escalation_reason,omitempty"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TicketResponseList lista paginada de tickets.
type TicketResponseList struct {
	Items  []TicketResponse `json:"items"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// CategoryResponse categoría de fidelización.
type CategoryResponse struct {
	ID        string          `json:"id"`
	CompanyID string          `json:"company_id"`
	Name      string          `json:"name"`
	MinLTV    decimal.Decimal `json:"min_ltv"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CreateCategoryRequest body para crear una categoría CRM (solo admin).
type CreateCategoryRequest struct {
	Name   string          `json:"name" validate:"required"`
	MinLTV decimal.Decimal `json:"min_ltv"`
}

// UpdateCategoryRequest body para actualizar una categoría CRM (solo admin).
type UpdateCategoryRequest struct {
	Name   *string          `json:"name,omitempty"`
	MinLTV *decimal.Decimal `json:"min_ltv,omitempty"`
}

// BenefitResponse beneficio por categoría.
type BenefitResponse struct {
	ID          string    `json:"id"`
	CompanyID   string    `json:"company_id"`
	CategoryID  string    `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskAlert sugerencia de tarea de recompra (GenerateReorderAlerts).
type TaskAlert struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Reason      string `json:"reason"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Opportunities
// ──────────────────────────────────────────────────────────────────────────────

// CreateOpportunityRequest body para crear una oportunidad.
type CreateOpportunityRequest struct {
	CustomerID        string          `json:"customer_id"`
	Title             string          `json:"title" validate:"required"`
	Amount            decimal.Decimal `json:"amount"`
	Probability       int             `json:"probability"` // 0–100
	Stage             string          `json:"stage"`       // defaults to "prospecto"
	ExpectedCloseDate *time.Time      `json:"expected_close_date"`
}

// OpportunityResponse oportunidad en respuestas.
type OpportunityResponse struct {
	ID                string          `json:"id"`
	CompanyID         string          `json:"company_id"`
	CustomerID        string          `json:"customer_id"`
	Title             string          `json:"title"`
	Amount            decimal.Decimal `json:"amount"`
	Probability       int             `json:"probability"`
	Stage             string          `json:"stage"`
	ExpectedCloseDate *time.Time      `json:"expected_close_date,omitempty"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// FunnelStageDTO resumen de una etapa del embudo de ventas.
type FunnelStageDTO struct {
	Stage       string          `json:"stage"`
	Count       int             `json:"count"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Campaigns
// ──────────────────────────────────────────────────────────────────────────────

// CreateCampaignRequest body para crear una campaña de marketing.
type CreateCampaignRequest struct {
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// CampaignResponse campaña en respuestas.
type CampaignResponse struct {
	ID          string     `json:"id"`
	CompanyID   string     `json:"company_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CampaignMetricsResponse métricas de envío y conversión de una campaña.
type CampaignMetricsResponse struct {
	CampaignID string          `json:"campaign_id"`
	Sent       int             `json:"sent"`
	Opened     int             `json:"opened"`
	Clicked    int             `json:"clicked"`
	Converted  int             `json:"converted"`
	Revenue    decimal.Decimal `json:"revenue"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Campaign Templates
// ──────────────────────────────────────────────────────────────────────────────

type CreateCampaignTemplateRequest struct {
	Name    string `json:"name" validate:"required"`
	Subject string `json:"subject" validate:"required"`
	Body    string `json:"body" validate:"required"`
}

type CampaignTemplateResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Campaign recipient resolution
// ──────────────────────────────────────────────────────────────────────────────

type CampaignRecipientStrategy struct {
	Type       string `json:"type"` // "category"
	CategoryID string `json:"category_id"`
}

type ResolveCampaignRecipientsRequest struct {
	Strategies []CampaignRecipientStrategy `json:"strategies"`
}

type CampaignRecipientDTO struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Segment    string `json:"segment"`
}

type ResolveCampaignRecipientsResponse struct {
	Recipients []CampaignRecipientDTO `json:"recipients"`
}

// ──────────────────────────────────────────────────────────────────────────────
// CRM Analytics Dashboard
// ──────────────────────────────────────────────────────────────────────────────

type CRMAnalyticsKPIsResponse struct {
	TotalCustomers int64           `json:"total_customers"`
	TotalSales     decimal.Decimal `json:"total_sales"`
	AverageTicket  decimal.Decimal `json:"average_ticket"`
}

type CRMAnalyticsSegmentationItem struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type CRMAnalyticsMonthlyEvolutionItem struct {
	Month string          `json:"month"`
	Sales decimal.Decimal `json:"sales"`
}

// SendCampaignRequest body para enviar una campaña de email masiva.
type SendCampaignRequest struct {
	Subject    string `json:"subject"`               // requerido
	Body       string `json:"body"`                  // requerido (texto generado por IA o HTML simple)
	CategoryID string `json:"category_id,omitempty"` // opcional: filtrar por categoría CRM
}

// SendTestCampaignRequest body para enviar un correo de prueba (una sola dirección).
type SendTestCampaignRequest struct {
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	Email      string `json:"email,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
}

// ImportCRMResponse resumen de la importación de clientes CRM desde Excel.
type ImportCRMResponse struct {
	ProcessedRows int      `json:"processed_rows"`
	Created       int      `json:"created"`
	Updated       int      `json:"updated"`
	Errors        []string `json:"errors,omitempty"`
}

// SendBulkCampaignRequest body para encolar envío masivo por customer IDs.
type SendBulkCampaignRequest struct {
	CustomerIDs []string `json:"customer_ids"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
}

// SendBulkCampaignResponse respuesta de encolado masivo.
type SendBulkCampaignResponse struct {
	CampaignID string `json:"campaign_id"`
	Queued     int    `json:"queued"`
	Status     string `json:"status"`
}

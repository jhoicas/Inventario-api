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

// UpdateTaskRequest body para actualizar una tarea.
type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	DueAt       *time.Time `json:"due_at"`
	Status      *string `json:"status"` // pending, done, cancelled
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

// Profile360Response vista 360 del cliente (datos base + perfil CRM).
type Profile360Response struct {
	Customer   CustomerResponse `json:"customer"`
	ProfileID  string          `json:"profile_id"`
	CategoryID string          `json:"category_id"`
	CategoryName string        `json:"category_name,omitempty"`
	LTV        decimal.Decimal `json:"ltv"`
	Benefits   []BenefitResponse `json:"benefits,omitempty"`
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

// TicketResponse ticket PQR en respuestas.
type TicketResponse struct {
	ID          string    `json:"id"`
	CompanyID   string    `json:"company_id"`
	CustomerID  string    `json:"customer_id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Sentiment   string    `json:"sentiment"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

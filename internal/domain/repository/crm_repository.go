package repository

import (
	"context"
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CRMCategoryRepository puerto de persistencia para categorías de fidelización.
type CRMCategoryRepository interface {
	Create(category *entity.CRMCategory) error
	GetByID(id string) (*entity.CRMCategory, error)
	ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCategory, error)
	Update(category *entity.CRMCategory) error
	Delete(id string) error
	SetActive(companyID, id string, isActive bool, updatedAt time.Time) error
}

// CRMBenefitRepository puerto de persistencia para beneficios por categoría.
type CRMBenefitRepository interface {
	Create(benefit *entity.CRMBenefit) error
	GetByID(id string) (*entity.CRMBenefit, error)
	ListByCategory(categoryID string, limit, offset int) ([]*entity.CRMBenefit, error)
	Update(benefit *entity.CRMBenefit) error
	SetActive(companyID, id string, isActive bool) error
}

// CRMProfileRepository puerto de persistencia para perfiles CRM (y vista 360).
type CRMProfileRepository interface {
	GetByCustomerID(customerID string) (*entity.CRMCustomerProfile, error)
	GetProfile360(ctx context.Context, companyID, customerID string) (*entity.Profile360, error)
	Upsert(profile *entity.CRMCustomerProfile) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCustomerProfile, error)
}

// InteractionFilters filtros opcionales para ListInteractions.
type InteractionFilters struct {
	Type      string    // filtra por tipo exacto (call, email, meeting, other); vacío = todos
	StartDate time.Time // filtra created_at >= StartDate cuando no es zero
	EndDate   time.Time // filtra created_at <= EndDate cuando no es zero
	Limit     int
	Offset    int
}

// CRMInteractionRepository puerto de persistencia para interacciones.
type CRMInteractionRepository interface {
	Create(interaction *entity.CRMInteraction) error
	GetByID(id string) (*entity.CRMInteraction, error)
	ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error)
	// ListInteractions lista interacciones de un cliente con filtros opcionales.
	// Devuelve el slice de resultados y el total (sin paginación) para el header X-Total-Count.
	ListInteractions(customerID string, f InteractionFilters) ([]*entity.CRMInteraction, int64, error)
}

// CRMTaskRepository puerto de persistencia para tareas.
type CRMTaskRepository interface {
	Create(task *entity.CRMTask) error
	GetByID(id string) (*entity.CRMTask, error)
	Update(task *entity.CRMTask) error
	ListByCompany(companyID string, status string, limit, offset int) ([]*entity.CRMTask, error)
}

// CRMTicketRepository puerto de persistencia para tickets PQR.
type CRMTicketRepository interface {
	Create(ticket *entity.CRMTicket) error
	GetByID(id string) (*entity.CRMTicket, error)
	Update(ticket *entity.CRMTicket) error
	// ListByCompany lista tickets por empresa con filtros opcionales.
	// search: busca por asunto (subject) usando ILIKE.
	// status: filtra por status exacto.
	// sort: orden por created_at ("asc" | "desc"). Cualquier otro valor usa "desc".
	ListByCompany(companyID string, search string, status string, sort string, limit, offset int) ([]*entity.CRMTicket, error)
	// UpdateStatus actualiza solo el status y updated_at de un ticket.
	UpdateStatus(id, status string, updatedAt time.Time) error
	// ListOverdue retorna los tickets en estado OVERDUE de una empresa.
	ListOverdue(companyID string) ([]*entity.CRMTicket, error)
	// MarkOverdueTickets marca como OVERDUE todos los tickets activos cuyo
	// created_at + sla_config.max_hours ha expirado. Devuelve el total marcado.
	MarkOverdueTickets(ctx context.Context) (int64, error)
}

// SLAConfigRepository puerto para configuración de SLA por empresa y tipo de ticket.
type SLAConfigRepository interface {
	// Upsert inserta o actualiza la configuración SLA.
	Upsert(ctx context.Context, cfg *entity.SLAConfig) error
	// GetByCompanyAndType obtiene la configuración SLA para una empresa y tipo.
	GetByCompanyAndType(ctx context.Context, companyID, ticketType string) (*entity.SLAConfig, error)
	// ListByCompany lista todas las configuraciones SLA de una empresa.
	ListByCompany(ctx context.Context, companyID string) ([]*entity.SLAConfig, error)
}

// CRMOpportunityRepository puerto de persistencia para oportunidades CRM.
type CRMOpportunityRepository interface {
	Create(ctx context.Context, opp *entity.Opportunity) error
	GetByID(ctx context.Context, id string) (*entity.Opportunity, error)
	UpdateStage(ctx context.Context, id string, stage entity.OpportunityStage, updatedAt time.Time) error
	ListByCompany(ctx context.Context, companyID string) ([]*entity.Opportunity, error)
}

// CRMCampaignRepository puerto de persistencia para campañas CRM.
type CRMCampaignRepository interface {
	Create(ctx context.Context, c *entity.Campaign) error
	GetByID(ctx context.Context, id string) (*entity.Campaign, error)
	GetMetrics(ctx context.Context, campaignID string) (*entity.CampaignMetrics, error)
}

// CRMCampaignTemplateRepository puerto de persistencia para plantillas de campañas.
type CRMCampaignTemplateRepository interface {
	Create(ctx context.Context, t *entity.CampaignTemplate) error
	FindAllByCompany(ctx context.Context, companyID string) ([]*entity.CampaignTemplate, error)
	Delete(ctx context.Context, id, companyID string) error
}

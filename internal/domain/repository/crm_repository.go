package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// CRMCategoryRepository puerto de persistencia para categorías de fidelización.
type CRMCategoryRepository interface {
	Create(category *entity.CRMCategory) error
	GetByID(id string) (*entity.CRMCategory, error)
	ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCategory, int64, error)
	Update(category *entity.CRMCategory) error
	Delete(id string) error
	SetActive(companyID, id string, isActive bool, updatedAt time.Time) error
}

// CRMBenefitRepository puerto de persistencia para beneficios por categoría.
type CRMBenefitRepository interface {
	Create(benefit *entity.CRMBenefit) error
	GetByID(id string) (*entity.CRMBenefit, error)
	ListByCategory(categoryID string, limit, offset int) ([]*entity.CRMBenefit, int64, error)
	Update(benefit *entity.CRMBenefit) error
	SetActive(companyID, id string, isActive bool) error
}

// CRMProfileRepository puerto de persistencia para perfiles CRM (y vista 360).
type CRMProfileRepository interface {
	GetByCustomerID(customerID string) (*entity.CRMCustomerProfile, error)
	GetProfile360(ctx context.Context, companyID, customerID string) (*entity.Profile360, error)
	Upsert(profile *entity.CRMCustomerProfile) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCustomerProfile, error)
	ResolveCampaignRecipientsByCategory(ctx context.Context, companyID, categoryID string) ([]dto.CampaignRecipientDTO, error)
	ResolveCampaignRecipients(ctx context.Context, companyID, categoryID string) ([]dto.CampaignRecipientDTO, error)
	GetAnalytics(ctx context.Context, companyID string) (*dto.CRMAnalyticsResponse, error)
	GetRemarketingProspects(ctx context.Context, companyID string) ([]dto.RemarketingProspect, error)
	GetRemarketingAudience(ctx context.Context, companyID, segmento, estrategia string) ([]dto.RemarketingAudienceDTO, error)
	GetRemarketingTargetsByCustomerIDs(ctx context.Context, companyID string, customerIDs []string) ([]dto.RemarketingAudienceDTO, error)

	// GetDashboardKPIs retorna KPIs agregados del dashboard CRM para una empresa.
	GetDashboardKPIs(companyID string) (*CRMDashboardKPIs, error)
	// GetDashboardSegmentation retorna distribución de clientes por categoría.
	GetDashboardSegmentation(companyID string) ([]*CRMSegmentDistribution, error)
	// GetDashboardMonthlyEvolution retorna ventas mensuales de los últimos N meses.
	GetDashboardMonthlyEvolution(companyID string, months int) ([]*CRMMonthlySales, error)
}

type CRMDashboardKPIs struct {
	TotalCustomers int64
	TotalSales     decimal.Decimal
	AverageTicket  decimal.Decimal
}

type CRMSegmentDistribution struct {
	Category   string
	Count      int64
	TotalSales decimal.Decimal
}

type CRMMonthlySales struct {
	Month string
	Sales decimal.Decimal
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
	ListByCompany(companyID string, status string, limit, offset int) ([]*entity.CRMTask, int64, error)
	CheckTaskExistsForToday(ctx context.Context, companyID, customerName, titlePrefix string) (bool, error)
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
	ListByCompany(companyID string, search string, status string, sort string, limit, offset int) ([]*entity.CRMTicket, int64, error)
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
	ListByCompanyPage(ctx context.Context, companyID string, limit, offset int) ([]*entity.Opportunity, error)
	CountByCompany(ctx context.Context, companyID string) (int64, error)
}

// CRMCampaignRepository puerto de persistencia para campañas CRM.
type CRMCampaignRepository interface {
	Create(ctx context.Context, c *entity.Campaign) error
	CreateWithRecipients(ctx context.Context, c *entity.Campaign, recipients []*entity.CampaignRecipient) error
	Update(ctx context.Context, c *entity.Campaign) error
	GetByID(ctx context.Context, id string) (*entity.Campaign, error)
	GetMetrics(ctx context.Context, campaignID string) (*entity.CampaignMetrics, error)
	QueueRecipients(ctx context.Context, campaignID string, recipients []*entity.CampaignRecipient) (int, error)
	BatchInsertCampaignRecipients(ctx context.Context, recipients []domain.CampaignRecipient) error
	ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]*entity.Campaign, int64, error)
}

// CRMAutomationRepository puerto de persistencia para automatizaciones CRM.
type CRMAutomationRepository interface {
	Create(ctx context.Context, automation *entity.CRMAutomation) error
	ListByCompany(ctx context.Context, companyID string) ([]*entity.CRMAutomation, error)
	GetByID(ctx context.Context, id string) (*entity.CRMAutomation, error)
	Update(ctx context.Context, automation *entity.CRMAutomation) error
	Delete(ctx context.Context, id, companyID string) error
	GetActiveAutomations(ctx context.Context) ([]*entity.CRMAutomation, error)
	GetCustomersForBirthday(ctx context.Context, companyID uuid.UUID) ([]*entity.Customer, error)
	GetCustomersForRepurchase(ctx context.Context, companyID uuid.UUID, productID uuid.UUID, daysSincePurchase int) ([]*entity.Customer, error)
}

// CRMCampaignTemplateRepository puerto de persistencia para plantillas de campañas.
type CRMCampaignTemplateRepository interface {
	Create(ctx context.Context, t *entity.CampaignTemplate) error
	GetByID(ctx context.Context, id string) (*entity.CampaignTemplate, error)
	FindAllByCompany(ctx context.Context, companyID string) ([]*entity.CampaignTemplate, error)
	Delete(ctx context.Context, id, companyID string) error
}

// CRMProductHubRepository puerto de persistencia para productos en el hub de analytics.
type CRMProductHubRepository interface {
	CreateBatch(ctx context.Context, products []*entity.ProductHub) error
	GetByCompanyAndCode(ctx context.Context, companyID, productCode string) (*entity.ProductHub, error)
	ListByCompany(ctx context.Context, companyID string) ([]*entity.ProductHub, error)
	Upsert(ctx context.Context, product *entity.ProductHub) error
}

// CRMSalesHubRepository puerto de persistencia para ventas en el hub de analytics.
type CRMSalesHubRepository interface {
	CreateBatch(ctx context.Context, sales []*entity.SaleHub) error
	GetByID(ctx context.Context, saleID string) (*entity.SaleHub, error)
	ListByCompanyAndDateRange(ctx context.Context, companyID string, startDate, endDate time.Time) ([]*entity.SaleHub, error)
}

// CRMSaleItemHubRepository puerto de persistencia para items de venta en el hub de analytics.
type CRMSaleItemHubRepository interface {
	CreateBatch(ctx context.Context, items []*entity.SaleItemHub) error
	GetBySaleID(ctx context.Context, saleID string) ([]*entity.SaleItemHub, error)
}

// AIAnalyticsRepository puerto de persistencia para queries en la vista de analytics.
type AIAnalyticsRepository interface {
	QueryView(ctx context.Context, companyID, sqlQuery string) ([]*entity.AIAnalyticsRow, error)
	RunAggregateQuery(ctx context.Context, companyID, sqlQuery string) (interface{}, error)
	GetCustomersAtRiskOfChurn(ctx context.Context, daysThreshold int) ([]*entity.CustomerChurnRisk, error)
}

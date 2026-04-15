package crm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// AutomationStrategy define el contrato para generar recipients según la automatización.
type AutomationStrategy interface {
	GenerateRecipients(ctx context.Context, automation domain.Automation) ([]domain.CampaignRecipient, error)
}

// AutomationStrategyFactory resuelve la estrategia según el tipo de automatización.
type AutomationStrategyFactory struct {
	birthday   AutomationStrategy
	repurchase AutomationStrategy
}

// NewAutomationStrategyFactory construye la factory de estrategias CRM.
func NewAutomationStrategyFactory(repo repository.CRMAutomationRepository) *AutomationStrategyFactory {
	return &AutomationStrategyFactory{
		birthday:   &BirthdayStrategy{repo: repo},
		repurchase: &RepurchaseStrategy{repo: repo},
	}
}

// GetStrategy retorna la estrategia asociada al tipo de automatización.
func (f *AutomationStrategyFactory) GetStrategy(automationType domain.AutomationType) (AutomationStrategy, error) {
	switch automationType {
	case domain.AutomationTypeBirthday:
		return f.birthday, nil
	case domain.AutomationTypeRepurchase:
		return f.repurchase, nil
	default:
		return nil, fmt.Errorf("tipo de automatización no soportado: %s", automationType)
	}
}

// BirthdayStrategy genera recipients para automatizaciones de cumpleaños.
type BirthdayStrategy struct {
	repo repository.CRMAutomationRepository
}

// GenerateRecipients obtiene los clientes con cumpleaños hoy y los transforma a recipients en QUEUED.
func (s *BirthdayStrategy) GenerateRecipients(ctx context.Context, automation domain.Automation) ([]domain.CampaignRecipient, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("repositorio de automatización no disponible")
	}
	companyID, err := uuid.Parse(strings.TrimSpace(automation.CompanyID))
	if err != nil {
		return nil, fmt.Errorf("company_id inválido: %w", err)
	}

	customers, err := s.repo.GetCustomersForBirthday(ctx, companyID)
	if err != nil {
		return nil, err
	}

	recipients := make([]domain.CampaignRecipient, 0, len(customers))
	for _, customer := range customers {
		if customer == nil {
			continue
		}
		email := strings.TrimSpace(customer.Email)
		if email == "" {
			continue
		}
		recipients = append(recipients, domain.CampaignRecipient{
			CustomerID: customer.ID,
			CompanyID:  automation.CompanyID,
			Email:      email,
			Status:     entity.CampaignRecipientStatusQueued,
			QueuedAt:   time.Now().UTC(),
		})
	}

	return recipients, nil
}

type repurchaseAutomationConfig struct {
	ProductID         string `json:"product_id"`
	DaysSincePurchase int    `json:"days_since_purchase"`
}

// RepurchaseStrategy genera recipients para automatizaciones de recompra.
type RepurchaseStrategy struct {
	repo repository.CRMAutomationRepository
}

// GenerateRecipients obtiene los clientes elegibles para recompra según el JSON config.
func (s *RepurchaseStrategy) GenerateRecipients(ctx context.Context, automation domain.Automation) ([]domain.CampaignRecipient, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("repositorio de automatización no disponible")
	}
	companyID, err := uuid.Parse(strings.TrimSpace(automation.CompanyID))
	if err != nil {
		return nil, fmt.Errorf("company_id inválido: %w", err)
	}

	var cfg repurchaseAutomationConfig
	if len(automation.Config) == 0 {
		return nil, fmt.Errorf("config de automatización vacío")
	}
	if err := json.Unmarshal(automation.Config, &cfg); err != nil {
		return nil, fmt.Errorf("config de automatización inválido: %w", err)
	}
	if strings.TrimSpace(cfg.ProductID) == "" || cfg.DaysSincePurchase <= 0 {
		return nil, fmt.Errorf("config de recompra incompleto")
	}

	productID, err := uuid.Parse(strings.TrimSpace(cfg.ProductID))
	if err != nil {
		return nil, fmt.Errorf("product_id inválido: %w", err)
	}

	customers, err := s.repo.GetCustomersForRepurchase(ctx, companyID, productID, cfg.DaysSincePurchase)
	if err != nil {
		return nil, err
	}

	recipients := make([]domain.CampaignRecipient, 0, len(customers))
	for _, customer := range customers {
		if customer == nil {
			continue
		}
		email := strings.TrimSpace(customer.Email)
		if email == "" {
			continue
		}
		recipients = append(recipients, domain.CampaignRecipient{
			CustomerID: customer.ID,
			CompanyID:  automation.CompanyID,
			Email:      email,
			Status:     entity.CampaignRecipientStatusQueued,
			QueuedAt:   time.Now().UTC(),
		})
	}

	return recipients, nil
}

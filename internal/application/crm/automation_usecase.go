package crm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// AutomationUseCase orquesta la ejecución diaria de automatizaciones CRM.
type AutomationUseCase struct {
	automationRepo repository.CRMAutomationRepository
	campaignRepo   repository.CRMCampaignRepository
	templateRepo   repository.CRMCampaignTemplateRepository
	factory        *AutomationStrategyFactory
	log            *logger.Logger
	auditUC        *AuditLogUseCase
}

// NewAutomationUseCase construye el caso de uso de automatizaciones.
func NewAutomationUseCase(
	automationRepo repository.CRMAutomationRepository,
	campaignRepo repository.CRMCampaignRepository,
	templateRepo repository.CRMCampaignTemplateRepository,
	factory *AutomationStrategyFactory,
	log *logger.Logger,
	auditUC *AuditLogUseCase,
) *AutomationUseCase {
	if factory == nil && automationRepo != nil {
		factory = NewAutomationStrategyFactory(automationRepo)
	}
	return &AutomationUseCase{
		automationRepo: automationRepo,
		campaignRepo:   campaignRepo,
		templateRepo:   templateRepo,
		factory:        factory,
		log:            log,
		auditUC:        auditUC,
	}
}

// RunDailyAutomations ejecuta las automatizaciones activas del día.
func (uc *AutomationUseCase) RunDailyAutomations(ctx context.Context) error {
	if uc == nil || uc.automationRepo == nil || uc.campaignRepo == nil || uc.templateRepo == nil || uc.factory == nil {
		return domain.ErrInvalidInput
	}

	automations, err := uc.automationRepo.GetActiveAutomations(ctx)
	if err != nil {
		return err
	}
	if len(automations) == 0 {
		if uc.log != nil {
			uc.log.Info().Msg("automatizaciones diarias: no hay automatizaciones activas")
		}
		return nil
	}

	if uc.log != nil {
		uc.log.Info().Int("count", len(automations)).Msg("automatizaciones diarias: automatizaciones activas cargadas")
	}

	var errs []error
	for _, automation := range automations {
		if automation == nil {
			continue
		}

		strategy, err := uc.factory.GetStrategy(domain.AutomationType(automation.Type))
		if err != nil {
			err = fmt.Errorf("automatización %s: %w", automation.ID, err)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Msg("automatización diaria: estrategia no disponible")
			}
			continue
		}

		recipients, err := strategy.GenerateRecipients(ctx, domain.Automation(*automation))
		if err != nil {
			err = fmt.Errorf("automatización %s: %w", automation.ID, err)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Msg("automatización diaria: error generando recipients")
			}
			continue
		}

		if len(recipients) == 0 {
			if uc.log != nil {
				uc.log.Info().Str("automation_id", automation.ID).Msg("automatización diaria: sin recipients elegibles")
			}
			continue
		}

		if automation.TemplateID == nil || strings.TrimSpace(*automation.TemplateID) == "" {
			err = fmt.Errorf("automatización %s: template_id requerido", automation.ID)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Msg("automatización diaria: plantilla faltante")
			}
			continue
		}

		template, err := uc.templateRepo.GetByID(ctx, *automation.TemplateID)
		if err != nil {
			err = fmt.Errorf("automatización %s: %w", automation.ID, err)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Msg("automatización diaria: no se pudo cargar la plantilla")
			}
			continue
		}
		if template == nil {
			err = fmt.Errorf("automatización %s: plantilla no encontrada", automation.ID)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Msg("automatización diaria: plantilla inexistente")
			}
			continue
		}

		now := time.Now().UTC()
		campaignName := fmt.Sprintf("Campaña Automática - %s - %s", automation.Name, now.Format("2006-01-02"))
		campaign := &entity.Campaign{
			ID:          uuid.New().String(),
			CompanyID:   automation.CompanyID,
			Name:        campaignName,
			Description: fmt.Sprintf("Automatización %s", automation.Type),
			Status:      entity.CampaignStatusScheduled,
			ScheduledAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		payload := make([]*entity.CampaignRecipient, 0, len(recipients))
		for _, recipient := range recipients {
			rec := entity.CampaignRecipient(recipient)
			rec.CampaignID = campaign.ID
			if strings.TrimSpace(rec.Subject) == "" {
				rec.Subject = template.Subject
			}
			if strings.TrimSpace(rec.Body) == "" {
				rec.Body = template.Body
			}
			recCopy := rec
			payload = append(payload, &recCopy)
		}

		if err := uc.campaignRepo.CreateWithRecipients(ctx, campaign, payload); err != nil {
			err = fmt.Errorf("automatización %s: %w", automation.ID, err)
			errs = append(errs, err)
			if uc.log != nil {
				uc.log.Error().Err(err).Str("automation_id", automation.ID).Int("recipients", len(payload)).Msg("automatización diaria: error guardando campaña y recipients")
			}
			continue
		}

		if uc.log != nil {
			uc.log.Info().
				Str("automation_id", automation.ID).
				Str("campaign_id", campaign.ID).
				Int("recipients", len(payload)).
				Msg("automatización diaria: campaña y recipients encolados")
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// CreateAutomation crea una automatización CRM para una empresa.
func (uc *AutomationUseCase) CreateAutomation(ctx context.Context, companyID string, req dto.CreateAutomationRequest) (*dto.AutomationResponse, error) {
	if uc == nil || uc.automationRepo == nil || strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrInvalidInput
	}
	name := strings.TrimSpace(req.Name)
	typ, err := parseAutomationType(req.Type)
	if name == "" || err != nil {
		return nil, domain.ErrInvalidInput
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	a := &entity.CRMAutomation{
		ID:           uuid.New().String(),
		CompanyID:    companyID,
		Name:         name,
		Type:         typ,
		TemplateID:   trimPtr(req.TemplateID),
		ScheduleCron: trimPtr(req.ScheduleCron),
		Config:       normalizeConfig(req.Config),
		IsActive:     isActive,
	}
	if err := uc.automationRepo.Create(ctx, a); err != nil {
		return nil, err
	}
	return toAutomationResponse(a), nil
}

// ListAutomations lista automatizaciones CRM de la empresa autenticada.
func (uc *AutomationUseCase) ListAutomations(ctx context.Context, companyID string) ([]dto.AutomationResponse, error) {
	if uc == nil || uc.automationRepo == nil || strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrInvalidInput
	}
	list, err := uc.automationRepo.ListByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AutomationResponse, 0, len(list))
	for _, item := range list {
		out = append(out, *toAutomationResponse(item))
	}
	return out, nil
}

// UpdateAutomation actualiza una automatización CRM existente.
func (uc *AutomationUseCase) UpdateAutomation(ctx context.Context, companyID, id string, req dto.UpdateAutomationRequest) (*dto.AutomationResponse, error) {
	return uc.UpdateAutomationByUser(ctx, companyID, "", id, req)
}

func (uc *AutomationUseCase) UpdateAutomationByUser(ctx context.Context, companyID, userID, id string, req dto.UpdateAutomationRequest) (*dto.AutomationResponse, error) {
	if uc == nil || uc.automationRepo == nil || strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	current, err := uc.automationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	before := *current
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		current.Name = name
	}
	if req.Type != nil {
		typ, err := parseAutomationType(*req.Type)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		current.Type = typ
	}
	if req.TemplateID != nil {
		current.TemplateID = trimPtr(req.TemplateID)
	}
	if req.ScheduleCron != nil {
		current.ScheduleCron = trimPtr(req.ScheduleCron)
	}
	if req.Config != nil {
		current.Config = normalizeConfig(req.Config)
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	if err := uc.automationRepo.Update(ctx, current); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	_ = uc.auditUC.RegisterChange(ctx, companyID, userID, "UPDATE", "AUTOMATION", current.ID, before, current)
	return toAutomationResponse(current), nil
}

// DeleteAutomation elimina una automatización CRM de la empresa autenticada.
func (uc *AutomationUseCase) DeleteAutomation(ctx context.Context, companyID, id string) error {
	return uc.DeleteAutomationByUser(ctx, companyID, "", id)
}

func (uc *AutomationUseCase) DeleteAutomationByUser(ctx context.Context, companyID, userID, id string) error {
	if uc == nil || uc.automationRepo == nil || strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return domain.ErrInvalidInput
	}
	current, err := uc.automationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return domain.ErrForbidden
	}
	if err := uc.automationRepo.Delete(ctx, id, companyID); err != nil {
		return err
	}
	_ = uc.auditUC.RegisterChange(ctx, companyID, userID, "DELETE", "AUTOMATION", id, current, nil)
	return nil
}

func parseAutomationType(raw string) (entity.CRMAutomationType, error) {
	s := entity.CRMAutomationType(strings.ToUpper(strings.TrimSpace(raw)))
	if s != entity.CRMAutomationTypeBirthday && s != entity.CRMAutomationTypeRepurchase {
		return "", domain.ErrInvalidInput
	}
	return s, nil
}

func trimPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeConfig(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}

func toAutomationResponse(a *entity.CRMAutomation) *dto.AutomationResponse {
	if a == nil {
		return nil
	}
	return &dto.AutomationResponse{
		ID:           a.ID,
		CompanyID:    a.CompanyID,
		Name:         a.Name,
		Type:         string(a.Type),
		TemplateID:   a.TemplateID,
		ScheduleCron: a.ScheduleCron,
		Config:       append([]byte(nil), a.Config...),
		IsActive:     a.IsActive,
		LastRunAt:    a.LastRunAt,
	}
}

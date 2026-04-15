package crm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
}

// NewAutomationUseCase construye el caso de uso de automatizaciones.
func NewAutomationUseCase(
	automationRepo repository.CRMAutomationRepository,
	campaignRepo repository.CRMCampaignRepository,
	templateRepo repository.CRMCampaignTemplateRepository,
	factory *AutomationStrategyFactory,
	log *logger.Logger,
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

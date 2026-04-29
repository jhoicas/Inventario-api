package crm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	inframail "github.com/jhoicas/Inventario-api/internal/infrastructure/mail"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// AutomationUseCase orquesta la ejecución diaria de automatizaciones CRM.
type AutomationUseCase struct {
	automationRepo   repository.CRMAutomationRepository
	campaignRepo     repository.CRMCampaignRepository
	templateRepo     repository.CRMCampaignTemplateRepository
	invoiceRepo      repository.InvoiceRepository
	customerRepo     repository.CustomerRepository
	profileRepo      repository.CRMProfileRepository
	benefitRepo      repository.CRMBenefitRepository
	notificationRepo repository.NotificationLogRepository
	llm              ports.LLMService
	mailSender       *inframail.SMTPSender
	factory          *AutomationStrategyFactory
	log              *logger.Logger
	auditUC          *AuditLogUseCase
}

const defaultAutomationScheduleCron = "0 0 * * *"

// NewAutomationUseCase construye el caso de uso de automatizaciones.
func NewAutomationUseCase(
	automationRepo repository.CRMAutomationRepository,
	campaignRepo repository.CRMCampaignRepository,
	templateRepo repository.CRMCampaignTemplateRepository,
	invoiceRepo repository.InvoiceRepository,
	customerRepo repository.CustomerRepository,
	profileRepo repository.CRMProfileRepository,
	benefitRepo repository.CRMBenefitRepository,
	notificationRepo repository.NotificationLogRepository,
	llm ports.LLMService,
	mailSender *inframail.SMTPSender,
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
		invoiceRepo:    invoiceRepo,
		customerRepo:   customerRepo,
		profileRepo:    profileRepo,
		benefitRepo:    benefitRepo,
		notificationRepo: notificationRepo,
		llm:              llm,
		mailSender:       mailSender,
		factory:        factory,
		log:            log,
		auditUC:        auditUC,
	}
}

var discountPercentRegex = regexp.MustCompile(`(?i)(\d{1,3})(?:[\.,]\d+)?\s*%`)

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
		ScheduleCron: ensureScheduleCron(trimPtr(req.ScheduleCron)),
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
		current.ScheduleCron = ensureScheduleCron(trimPtr(req.ScheduleCron))
	}
	if current.ScheduleCron == nil {
		current.ScheduleCron = ensureScheduleCron(nil)
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

func ensureScheduleCron(v *string) *string {
	if v != nil {
		trimmed := strings.TrimSpace(*v)
		if trimmed != "" {
			return &trimmed
		}
	}
	cron := defaultAutomationScheduleCron
	return &cron
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

func (uc *AutomationUseCase) TriggerBirthdays(ctx context.Context, companyID string) (*dto.TriggerBirthdayResultResponse, error) {
	if uc == nil || uc.automationRepo == nil || uc.mailSender == nil || uc.llm == nil || uc.notificationRepo == nil {
		return nil, domain.ErrInvalidInput
	}
	companyUUID, err := uuid.Parse(strings.TrimSpace(companyID))
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	customers, err := uc.automationRepo.GetCustomersForBirthday(ctx, companyUUID)
	if err != nil {
		return nil, err
	}

	result := &dto.TriggerBirthdayResultResponse{Processed: len(customers)}
	for _, c := range customers {
		if c == nil || strings.TrimSpace(c.Email) == "" {
			continue
		}

		customer, err := uc.customerRepo.GetByID(c.ID)
		if err != nil || customer == nil || customer.CompanyID != companyID {
			continue
		}

		topProducts, err := uc.invoiceRepo.GetTopProductNamesByCustomer(customer.ID, 3)
		if err != nil {
			topProducts = []string{}
		}
		maxDiscount := uc.resolveMaxDiscountPercent(customer.ID)
		body, subject, aiErr := uc.generateBirthdayHTML(ctx, customer, topProducts, maxDiscount)

		status := "SENT"
		errorMessage := ""
		if aiErr != nil {
			status = "FAILED"
			errorMessage = aiErr.Error()
		} else if err := uc.mailSender.Send(customer.Email, subject, body); err != nil {
			status = "FAILED"
			errorMessage = err.Error()
		}

		if status == "SENT" {
			result.Sent++
		} else {
			result.Failed++
		}

		_ = uc.notificationRepo.Create(ctx, &entity.NotificationLog{
			CompanyID:    companyID,
			CustomerID:   customer.ID,
			Type:         "BIRTHDAY",
			Channel:      "EMAIL",
			Subject:      subject,
			Body:         body,
			SentAt:       time.Now().UTC(),
			Status:       status,
			ErrorMessage: errorMessage,
		})
	}
	return result, nil
}

func (uc *AutomationUseCase) resolveMaxDiscountPercent(customerID string) int {
	if uc.profileRepo == nil || uc.benefitRepo == nil {
		return 0
	}
	profile, err := uc.profileRepo.GetByCustomerID(customerID)
	if err != nil || profile == nil || strings.TrimSpace(profile.CategoryID) == "" {
		return 0
	}
	benefits, _, err := uc.benefitRepo.ListByCategory(profile.CategoryID, 200, 0)
	if err != nil {
		return 0
	}
	maxDiscount := 0
	for _, b := range benefits {
		if b == nil {
			continue
		}
		text := strings.TrimSpace(b.Name + " " + b.Description)
		match := discountPercentRegex.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		v, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if v > maxDiscount {
			maxDiscount = v
		}
	}
	return maxDiscount
}

func (uc *AutomationUseCase) generateBirthdayHTML(ctx context.Context, customer *entity.Customer, topProducts []string, maxDiscount int) (string, string, error) {
	firstName := strings.TrimSpace(customer.Name)
	if firstName == "" {
		firstName = "Cliente"
	}
	parts := strings.Fields(firstName)
	if len(parts) > 0 {
		firstName = parts[0]
	}
	productList := "nuestros productos favoritos"
	if len(topProducts) > 0 {
		productList = strings.Join(topProducts, ", ")
	}
	discountText := "0"
	if maxDiscount > 0 {
		discountText = strconv.Itoa(maxDiscount)
	}
	systemPrompt := "Eres un experto en fidelización. Escribe un correo de feliz cumpleaños cálido para [Nombre del Cliente]. Menciona sutilmente que sabemos que le encantan los productos como [Lista de Productos]. Ofrécele un descuento exclusivo de cumpleaños de [X]% (solo si X > 0). El mensaje debe ser corto, profesional pero muy cercano, y generar ventas sin sonar desesperado. Formato HTML ligero."
	userPrompt := fmt.Sprintf("Nombre del cliente: %s\nLista de productos: %s\nDescuento maximo permitido: %s", firstName, productList, discountText)
	body, err := uc.llm.GenerateTextWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", "", err
	}
	return body, fmt.Sprintf("Feliz cumpleaños, %s", firstName), nil
}

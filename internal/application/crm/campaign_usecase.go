package crm

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	inframail "github.com/jhoicas/Inventario-api/internal/infrastructure/mail"
)

// CampaignUseCase gestión de campañas de marketing CRM.
type CampaignUseCase struct {
	repo            repository.CRMCampaignRepository
	customerRepo    repository.CustomerRepository
	profileRepo     repository.CRMProfileRepository
	interactionRepo repository.CRMInteractionRepository
	providers       map[string]repository.MessageProvider
	mailSender      *inframail.SMTPSender
}

// NewCampaignUseCase construye el caso de uso.
func NewCampaignUseCase(
	repo repository.CRMCampaignRepository,
	customerRepo repository.CustomerRepository,
	profileRepo repository.CRMProfileRepository,
	interactionRepo repository.CRMInteractionRepository,
	providers map[string]repository.MessageProvider,
	mailSender *inframail.SMTPSender,
) *CampaignUseCase {
	return &CampaignUseCase{
		repo:            repo,
		customerRepo:    customerRepo,
		profileRepo:     profileRepo,
		interactionRepo: interactionRepo,
		providers:       providers,
		mailSender:      mailSender,
	}
}

// Create crea una campaña en estado pending.
func (uc *CampaignUseCase) Create(ctx context.Context, companyID, userID string, req dto.CreateCampaignRequest) (*dto.CampaignResponse, error) {
	if req.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	channel := strings.ToUpper(strings.TrimSpace(req.Channel))
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = entity.CampaignStatusPending
	}
	scheduledAt := normalizeTimePtrToUTC(req.ScheduledAt)
	if scheduledAt == nil {
		nowUTC := time.Now().UTC()
		scheduledAt = &nowUTC
		status = entity.CampaignStatusScheduled
	}

	if strings.TrimSpace(req.Body) == "" {
		return nil, domain.ErrInvalidInput
	}
	if channel == "EMAIL" && strings.TrimSpace(req.Subject) == "" {
		return nil, domain.ErrInvalidInput
	}

	isScheduled := true
	if req.ScheduledAt != nil {
		status = entity.CampaignStatusScheduled
	}

	now := time.Now()
	c := &entity.Campaign{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		Name:        req.Name,
		Description: req.Description,
		Subject:     req.Subject,
		Body:        req.Body,
		Status:      status,
		Channel:     req.Channel,
		ScheduledAt: scheduledAt,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if isScheduled {
		recipientsDTO, err := uc.profileRepo.ResolveCampaignRecipients(ctx, companyID, strings.TrimSpace(req.CategoryID))
		if err != nil {
			return nil, err
		}
		if len(recipientsDTO) == 0 {
			return nil, domain.ErrInvalidInput
		}

		recipients := make([]*entity.CampaignRecipient, 0, len(recipientsDTO))
		for _, recipient := range recipientsDTO {
			email := strings.TrimSpace(recipient.Email)
			phone := strings.TrimSpace(recipient.Phone)

			if channel == "EMAIL" && email == "" {
				continue
			}
			if (channel == "WHATSAPP" || channel == "SMS") && phone == "" {
				continue
			}
			custName := "Cliente"
			cust, err := uc.customerRepo.GetByID(recipient.CustomerID)
			if err == nil && cust != nil && strings.TrimSpace(cust.Name) != "" {
				custName = strings.TrimSpace(cust.Name)
			}
			customBody := strings.ReplaceAll(req.Body, "[Nombre]", custName)
			recipients = append(recipients, &entity.CampaignRecipient{
				CustomerID: recipient.CustomerID,
				CompanyID:  companyID,
				Email:      email,
				Phone:      phone,
				Subject:    req.Subject,
				Body:       customBody,
				Status:     "QUEUED",
				QueuedAt:   now,
			})
		}
		if len(recipients) == 0 {
			return nil, domain.ErrInvalidInput
		}
		if err := uc.repo.CreateWithRecipients(ctx, c, recipients); err != nil {
			return nil, err
		}
	} else {
		if err := uc.repo.Create(ctx, c); err != nil {
			return nil, err
		}
	}
	return toCampaignResponse(c), nil
}

// GetMetrics devuelve las métricas de envío y conversión de una campaña.
func (uc *CampaignUseCase) GetMetrics(ctx context.Context, campaignID string) (*dto.CampaignMetricsResponse, error) {
	if campaignID == "" {
		return nil, domain.ErrInvalidInput
	}

	m, err := uc.repo.GetMetrics(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, domain.ErrNotFound
	}

	return &dto.CampaignMetricsResponse{
		CampaignID: m.CampaignID,
		Sent:       m.Sent,
		Opened:     m.Opened,
		Clicked:    m.Clicked,
		Converted:  m.Converted,
		Revenue:    m.Revenue,
	}, nil
}

// ListCampaigns devuelve la lista de campañas de la empresa.
func (uc *CampaignUseCase) ListCampaigns(ctx context.Context, companyID string, limit, offset int) (*dto.CampaignListResponse, error) {
	campaigns, total, err := uc.repo.ListByCompany(ctx, companyID, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CampaignResponse, 0, len(campaigns))
	for _, c := range campaigns {
		items = append(items, *toCampaignResponse(c))
	}

	return &dto.CampaignListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ExecuteCampaign procesa una campaña pendiente o programada de forma manual.
func (uc *CampaignUseCase) ExecuteCampaign(ctx context.Context, companyID, userID, campaignID string) error {
	c, err := uc.repo.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if c == nil || c.CompanyID != companyID {
		return domain.ErrNotFound
	}

	// Solo permitir ejecutar si está pendiente o programada
	if c.Status != entity.CampaignStatusPending && c.Status != entity.CampaignStatusScheduled {
		return domain.ErrConflict
	}

	// Reutilizamos la lógica de envío masivo
	req := dto.SendCampaignRequest{
		Channel:    c.Channel,
		Subject:    c.Subject,
		Body:       c.Body,
		CategoryID: "", // Podríamos guardar la categoría en la entidad si fuera necesario
	}

	err = uc.SendCampaign(ctx, companyID, userID, req)
	if err != nil {
		return err
	}

	// Marcar como completada
	c.Status = entity.CampaignStatusCompleted
	c.UpdatedAt = time.Now()
	return uc.repo.Update(ctx, c)
}

// SendTestMessage envía un mensaje directo a un destino sin guardarlo en la base de datos.
func (uc *CampaignUseCase) SendTestMessage(ctx context.Context, companyID string, req dto.SendTestMessageRequest) error {
	channel := strings.ToUpper(req.Channel)
	provider, exists := uc.providers[channel]
	if !exists || isNil(provider) {
		return domain.ErrConflict
	}

	return provider.Send(ctx, req.DestinationPhone, req.Content)
}

// SendCampaign envía una campaña de email a los clientes filtrados por categoría (opcional)
// y registra una interacción de tipo "email" por cada envío exitoso.
func (uc *CampaignUseCase) SendCampaign(ctx context.Context, companyID, userID string, req dto.SendCampaignRequest) error {
	if req.Channel == "EMAIL" && strings.TrimSpace(req.Subject) == "" {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(req.Body) == "" {
		return domain.ErrInvalidInput
	}
	channel := strings.ToUpper(req.Channel)
	var provider repository.MessageProvider
	if channel != "EMAIL" {
		p, exists := uc.providers[channel]
		if !exists || isNil(p) {
			return domain.ErrConflict
		}
		provider = p
	} else {
		if uc.mailSender == nil {
			return domain.ErrConflict
		}
	}

	// Resolver destinatarios
	var customerIDs []string

	if req.CategoryID != "" {
		// Filtrar por categoría CRM: buscar perfiles con esa CategoryID.
		profiles, err := uc.profileRepo.ListByCompany(companyID, 2000, 0)
		if err != nil {
			return err
		}
		for _, p := range profiles {
			if p.CategoryID == req.CategoryID {
				customerIDs = append(customerIDs, p.CustomerID)
			}
		}
	} else {
		// Sin filtro: enviar a todos los clientes de la empresa (paginando).
		offset := 0
		limit := 200
		for {
			customers, _, err := uc.customerRepo.ListByCompany(companyID, "", limit, offset)
			if err != nil {
				return err
			}
			if len(customers) == 0 {
				break
			}
			for _, c := range customers {
				customerIDs = append(customerIDs, c.ID)
			}
			if len(customers) < limit {
				break
			}
			offset += limit
		}
	}

	now := time.Now()

	for _, cid := range customerIDs {
		cust, err := uc.customerRepo.GetByID(cid)
		if err != nil || cust == nil || cust.CompanyID != companyID {
			continue
		}
		email := strings.TrimSpace(cust.Email)
		if email == "" {
			continue
		}

		// Intentar enviar mensaje
		if channel == "EMAIL" {
			if err := uc.mailSender.Send(email, req.Subject, req.Body); err != nil {
				continue
			}
		} else {
			if err := provider.Send(ctx, email, req.Body); err != nil {
				continue
			}
		}

		if uc.interactionRepo == nil {
			continue
		}

		interaction := &entity.CRMInteraction{
			ID:         uuid.New().String(),
			CompanyID:  companyID,
			CustomerID: cust.ID,
			Type:       entity.InteractionTypeEmail,
			Subject:    req.Subject,
			Body:       "Campaña enviada",
			CreatedBy:  userID,
			CreatedAt:  now,
		}
		_ = uc.interactionRepo.Create(interaction)
	}

	return nil
}

// SendTest envía un correo de prueba a una dirección específica.
func (uc *CampaignUseCase) SendTest(ctx context.Context, companyID, userID string, req dto.SendTestCampaignRequest) error {
	if req.Channel == "EMAIL" && strings.TrimSpace(req.Subject) == "" {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(req.Body) == "" {
		return domain.ErrInvalidInput
	}
	channel := strings.ToUpper(req.Channel)
	var provider repository.MessageProvider
	if channel != "EMAIL" {
		p, exists := uc.providers[channel]
		if !exists || isNil(p) {
			return domain.ErrConflict
		}
		provider = p
	} else {
		if uc.mailSender == nil {
			return domain.ErrConflict
		}
	}

	toEmail := strings.TrimSpace(req.Email)
	body := req.Body

	if strings.TrimSpace(req.CustomerID) != "" {
		cust, err := uc.customerRepo.GetByID(req.CustomerID)
		if err != nil {
			return err
		}
		if cust == nil {
			return domain.ErrNotFound
		}
		if cust.CompanyID != companyID {
			return domain.ErrForbidden
		}
		toEmail = strings.TrimSpace(cust.Email)
		if toEmail == "" {
			return domain.ErrInvalidInput
		}
		name := strings.TrimSpace(cust.Name)
		if name != "" {
			body = strings.ReplaceAll(body, "[Nombre]", name)
		}
	}

	if toEmail == "" {
		return domain.ErrInvalidInput
	}

	if channel == "EMAIL" {
		return uc.mailSender.Send(toEmail, req.Subject, body)
	}
	return provider.Send(ctx, toEmail, body)
}

// isNil detecta interfaces con puntero interno nil (evita panics).
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}

func toCampaignResponse(c *entity.Campaign) *dto.CampaignResponse {
	resp := &dto.CampaignResponse{
		ID:          c.ID,
		CompanyID:   c.CompanyID,
		Name:        c.Name,
		Description: c.Description,
		Status:      c.Status,
		Channel:     c.Channel,
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
	if c.ScheduledAt != nil {
		s := c.ScheduledAt.UTC()
		resp.ScheduledAt = &s
	}
	return resp
}

func normalizeTimePtrToUTC(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

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
	repo           repository.CRMCampaignRepository
	customerRepo   repository.CustomerRepository
	profileRepo    repository.CRMProfileRepository
	interactionRepo repository.CRMInteractionRepository
	mailSender     inframail.Sender
}

// NewCampaignUseCase construye el caso de uso.
func NewCampaignUseCase(
	repo repository.CRMCampaignRepository,
	customerRepo repository.CustomerRepository,
	profileRepo repository.CRMProfileRepository,
	interactionRepo repository.CRMInteractionRepository,
	mailSender inframail.Sender,
) *CampaignUseCase {
	return &CampaignUseCase{
		repo:            repo,
		customerRepo:    customerRepo,
		profileRepo:     profileRepo,
		interactionRepo: interactionRepo,
		mailSender:      mailSender,
	}
}

// Create crea una campaña en estado BORRADOR.
func (uc *CampaignUseCase) Create(ctx context.Context, companyID, userID string, req dto.CreateCampaignRequest) (*dto.CampaignResponse, error) {
	if req.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	c := &entity.Campaign{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		Name:        req.Name,
		Description: req.Description,
		Status:      entity.CampaignStatusBorrador,
		ScheduledAt: req.ScheduledAt,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(ctx, c); err != nil {
		return nil, err
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

// SendCampaign envía una campaña de email a los clientes filtrados por categoría (opcional)
// y registra una interacción de tipo "email" por cada envío exitoso.
func (uc *CampaignUseCase) SendCampaign(ctx context.Context, companyID, userID string, req dto.SendCampaignRequest) error {
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Body) == "" {
		return domain.ErrInvalidInput
	}
	if isNil(uc.mailSender) {
		return domain.ErrConflict
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
			customers, err := uc.customerRepo.ListByCompany(companyID, "", limit, offset)
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

		// Intentar enviar email; si falla uno, continuar con el siguiente.
		if err := uc.mailSender.Send(email, req.Subject, req.Body); err != nil {
			continue
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
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Body) == "" {
		return domain.ErrInvalidInput
	}
	if isNil(uc.mailSender) {
		return domain.ErrConflict
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
	return uc.mailSender.Send(toEmail, req.Subject, body)
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
		Status:      string(c.Status),
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
	if c.ScheduledAt != nil {
		resp.ScheduledAt = c.ScheduledAt
	}
	return resp
}

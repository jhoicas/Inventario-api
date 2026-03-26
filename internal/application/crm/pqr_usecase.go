package crm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// SentimentAnalyzer interfaz opcional para analizar sentimiento al crear ticket (inyectada en PASO 4).
type SentimentAnalyzer interface {
	AnalyzePQRSentiment(ctx context.Context, description string) (string, error)
}

// PQRUseCase gestión de tickets PQR.
type PQRUseCase struct {
	ticketRepo      repository.CRMTicketRepository
	customerRepo    repository.CustomerRepository
	interactionRepo repository.CRMInteractionRepository
	sentiment       SentimentAnalyzer // opcional; si es nil no se analiza sentimiento
}

// NewPQRUseCase construye el caso de uso.
func NewPQRUseCase(
	ticketRepo repository.CRMTicketRepository,
	customerRepo repository.CustomerRepository,
	sentiment SentimentAnalyzer,
	interactionRepo repository.CRMInteractionRepository,
) *PQRUseCase {
	return &PQRUseCase{
		ticketRepo:      ticketRepo,
		customerRepo:    customerRepo,
		sentiment:       sentiment,
		interactionRepo: interactionRepo,
	}
}

// Create radica un ticket PQR. Si SentimentAnalyzer está inyectado, analiza el sentimiento y lo guarda.
func (uc *PQRUseCase) Create(ctx context.Context, companyID, userID string, in dto.CreateTicketRequest) (*dto.TicketResponse, error) {
	if in.CustomerID == "" || in.Subject == "" || in.Description == "" {
		return nil, domain.ErrInvalidInput
	}
	customer, err := uc.customerRepo.GetByID(in.CustomerID)
	if err != nil || customer == nil {
		return nil, domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	now := time.Now()
	ticket := &entity.CRMTicket{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CustomerID:  in.CustomerID,
		Subject:     in.Subject,
		Description: in.Description,
		Status:      "open",
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if uc.sentiment != nil {
		if sent, err := uc.sentiment.AnalyzePQRSentiment(ctx, in.Description); err == nil && sent != "" {
			ticket.Sentiment = sent
		}
	}
	if err := uc.ticketRepo.Create(ticket); err != nil {
		return nil, err
	}
	return toTicketResponse(ticket), nil
}

// CreateFromEmail radica un ticket PQR a partir de un correo y asocia cliente por sender_email.
func (uc *PQRUseCase) CreateFromEmail(ctx context.Context, companyID, userID string, in dto.CreateTicketFromEmailRequest) (*dto.TicketResponse, error) {
	if strings.TrimSpace(in.EmailID) == "" || strings.TrimSpace(in.Subject) == "" || strings.TrimSpace(in.Description) == "" || strings.TrimSpace(in.SenderEmail) == "" || strings.TrimSpace(in.Priority) == "" {
		return nil, domain.ErrInvalidInput
	}

	customer, err := uc.customerRepo.GetByCompanyAndEmail(companyID, strings.TrimSpace(strings.ToLower(in.SenderEmail)))
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, domain.ErrNotFound
	}

	createdBy := strings.TrimSpace(userID)
	if strings.TrimSpace(in.AssignedTo) != "" {
		createdBy = strings.TrimSpace(in.AssignedTo)
	}

	description := strings.TrimSpace(in.Description)
	description = fmt.Sprintf("%s\n\n[Origen correo] email_id=%s\n[Prioridad] %s", description, strings.TrimSpace(in.EmailID), strings.TrimSpace(in.Priority))

	now := time.Now()
	ticket := &entity.CRMTicket{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CustomerID:  customer.ID,
		Subject:     strings.TrimSpace(in.Subject),
		Description: description,
		Status:      entity.TicketStatusOpen,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if uc.sentiment != nil {
		if sent, err := uc.sentiment.AnalyzePQRSentiment(ctx, ticket.Description); err == nil && sent != "" {
			ticket.Sentiment = sent
		}
	}

	if err := uc.ticketRepo.Create(ticket); err != nil {
		return nil, err
	}
	return toTicketResponse(ticket), nil
}

// GetByID obtiene un ticket por ID.
func (uc *PQRUseCase) GetByID(ctx context.Context, companyID, id string) (*dto.TicketResponse, error) {
	ticket, err := uc.ticketRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, domain.ErrNotFound
	}
	if ticket.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	return toTicketResponse(ticket), nil
}

// Update actualiza un ticket. Si cambia el status, registra una interacción automática.
func (uc *PQRUseCase) Update(ctx context.Context, companyID, userID, id string, in dto.UpdateTicketRequest) (*dto.TicketResponse, error) {
	ticket, err := uc.ticketRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, domain.ErrNotFound
	}
	if ticket.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	prevStatus := ticket.Status
	if in.Subject != nil {
		ticket.Subject = *in.Subject
	}
	if in.Description != nil {
		ticket.Description = *in.Description
	}
	if in.Status != nil {
		ticket.Status = *in.Status
	}
	if in.Sentiment != nil {
		ticket.Sentiment = *in.Sentiment
	}
	ticket.UpdatedAt = time.Now()
	if err := uc.ticketRepo.Update(ticket); err != nil {
		return nil, err
	}

	if in.Status != nil && ticket.Status != prevStatus && uc.interactionRepo != nil {
		now := time.Now()
		m := &entity.CRMInteraction{
			ID:         uuid.New().String(),
			CompanyID:  companyID,
			CustomerID: ticket.CustomerID,
			Type:       entity.InteractionTypeOther,
			Subject:    fmt.Sprintf("Actualización de ticket PQR (%s)", ticket.ID),
			Body:       fmt.Sprintf("El ticket cambió de estado: '%s' → '%s'. Asunto: %s", prevStatus, ticket.Status, ticket.Subject),
			CreatedBy:  userID,
			CreatedAt:  now,
		}
		if err := uc.interactionRepo.Create(m); err != nil {
			return nil, err
		}
	}
	return toTicketResponse(ticket), nil
}

// ListByCompany lista tickets de la empresa con filtros opcionales.
func (uc *PQRUseCase) ListByCompany(ctx context.Context, companyID string, search string, status string, sort string, limit, offset int) (*dto.TicketResponseList, error) {
	list, err := uc.ticketRepo.ListByCompany(companyID, search, status, sort, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.TicketResponse, 0, len(list))
	for _, t := range list {
		items = append(items, *toTicketResponse(t))
	}
	return &dto.TicketResponseList{Items: items, Limit: limit, Offset: offset}, nil
}

// EscalateTicket marca un ticket como ESCALATED y registra la razón.
func (uc *PQRUseCase) EscalateTicket(ctx context.Context, companyID, ticketID, reason string) error {
	if ticketID == "" || reason == "" {
		return domain.ErrInvalidInput
	}
	ticket, err := uc.ticketRepo.GetByID(ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return domain.ErrNotFound
	}
	if ticket.CompanyID != companyID {
		return domain.ErrForbidden
	}
	now := time.Now()
	ticket.Status = entity.TicketStatusEscalated
	ticket.EscalationReason = reason
	ticket.UpdatedAt = now
	if err := uc.ticketRepo.Update(ticket); err != nil {
		return err
	}
	if uc.interactionRepo != nil {
		m := &entity.CRMInteraction{
			ID:         fmt.Sprintf("%s-esc", ticket.ID), // uuid collision-safe
			CompanyID:  companyID,
			CustomerID: ticket.CustomerID,
			Type:       entity.InteractionTypeOther,
			Subject:    fmt.Sprintf("Ticket PQR escalado (%s)", ticket.ID),
			Body:       fmt.Sprintf("Razón de escalamiento: %s", reason),
			CreatedBy:  "",
			CreatedAt:  now,
		}
		if idErr := uc.interactionRepo.Create(m); idErr != nil {
			return idErr
		}
	}
	return nil
}

// ListOverdue lista los tickets en estado OVERDUE de una empresa.
func (uc *PQRUseCase) ListOverdue(ctx context.Context, companyID string) ([]*entity.CRMTicket, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	return uc.ticketRepo.ListOverdue(companyID)
}

func toTicketResponse(t *entity.CRMTicket) *dto.TicketResponse {
	return &dto.TicketResponse{
		ID:               t.ID,
		CompanyID:        t.CompanyID,
		CustomerID:       t.CustomerID,
		Subject:          t.Subject,
		Description:      t.Description,
		Status:           t.Status,
		Sentiment:        t.Sentiment,
		EscalationReason: t.EscalationReason,
		CreatedBy:        t.CreatedBy,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

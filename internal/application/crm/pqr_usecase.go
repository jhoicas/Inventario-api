package crm

import (
	"context"
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
	ticketRepo repository.CRMTicketRepository
	customerRepo repository.CustomerRepository
	sentiment    SentimentAnalyzer // opcional; si es nil no se analiza sentimiento
}

// NewPQRUseCase construye el caso de uso.
func NewPQRUseCase(
	ticketRepo repository.CRMTicketRepository,
	customerRepo repository.CustomerRepository,
	sentiment SentimentAnalyzer,
) *PQRUseCase {
	return &PQRUseCase{
		ticketRepo:   ticketRepo,
		customerRepo: customerRepo,
		sentiment:    sentiment,
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

// Update actualiza un ticket.
func (uc *PQRUseCase) Update(ctx context.Context, companyID, id string, in dto.UpdateTicketRequest) (*dto.TicketResponse, error) {
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
	if in.Subject != nil {
		ticket.Subject = *in.Subject
	}
	if in.Description != nil {
		ticket.Description = *in.Description
	}
	if in.Status != nil {
		ticket.Status = *in.Status
	}
	ticket.UpdatedAt = time.Now()
	if err := uc.ticketRepo.Update(ticket); err != nil {
		return nil, err
	}
	return toTicketResponse(ticket), nil
}

// ListByCompany lista tickets de la empresa.
func (uc *PQRUseCase) ListByCompany(ctx context.Context, companyID string, limit, offset int) (*dto.TicketResponseList, error) {
	list, err := uc.ticketRepo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.TicketResponse, 0, len(list))
	for _, t := range list {
		items = append(items, *toTicketResponse(t))
	}
	return &dto.TicketResponseList{Items: items, Limit: limit, Offset: offset}, nil
}

func toTicketResponse(t *entity.CRMTicket) *dto.TicketResponse {
	return &dto.TicketResponse{
		ID:          t.ID,
		CompanyID:   t.CompanyID,
		CustomerID:  t.CustomerID,
		Subject:     t.Subject,
		Description: t.Description,
		Status:      t.Status,
		Sentiment:   t.Sentiment,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

package crm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

// LoyaltyUseCase gestiona categorías de fidelización y perfiles CRM.
type LoyaltyUseCase struct {
	profileRepo     repository.CRMProfileRepository
	customerRepo    repository.CustomerRepository
	categoryRepo    repository.CRMCategoryRepository
	benefitRepo     repository.CRMBenefitRepository
	interactionRepo repository.CRMInteractionRepository
}

// NewLoyaltyUseCase construye el caso de uso.
func NewLoyaltyUseCase(
	profileRepo repository.CRMProfileRepository,
	customerRepo repository.CustomerRepository,
	categoryRepo repository.CRMCategoryRepository,
	benefitRepo repository.CRMBenefitRepository,
	interactionRepo repository.CRMInteractionRepository,
) *LoyaltyUseCase {
	return &LoyaltyUseCase{
		profileRepo:     profileRepo,
		customerRepo:    customerRepo,
		categoryRepo:    categoryRepo,
		benefitRepo:     benefitRepo,
		interactionRepo: interactionRepo,
	}
}

const loyaltyPointsSubjectPrefix = "LOYALTY_POINTS_"

type loyaltyPointEventPayload struct {
	Delta       int    `json:"delta"`
	Reason      string `json:"reason"`
	ReferenceID string `json:"reference_id,omitempty"`
}

// AwardPoints acredita puntos al cliente y registra un evento de historial.
func (uc *LoyaltyUseCase) AwardPoints(ctx context.Context, customerID string, points int, reason, referenceID string) error {
	if customerID == "" || points <= 0 || strings.TrimSpace(reason) == "" {
		return domain.ErrInvalidInput
	}
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return domain.ErrNotFound
	}
	if uc.interactionRepo == nil {
		return domain.ErrConflict
	}

	payload, err := json.Marshal(loyaltyPointEventPayload{Delta: points, Reason: reason, ReferenceID: referenceID})
	if err != nil {
		return err
	}
	now := time.Now()
	return uc.interactionRepo.Create(&entity.CRMInteraction{
		ID:         uuid.New().String(),
		CompanyID:  customer.CompanyID,
		CustomerID: customerID,
		Type:       entity.InteractionTypeOther,
		Subject:    loyaltyPointsSubjectPrefix + "AWARD",
		Body:       string(payload),
		CreatedBy:  "system",
		CreatedAt:  now,
	})
}

// GetBalance obtiene balance, tier actual, umbral al siguiente tier e historial de puntos.
func (uc *LoyaltyUseCase) GetBalance(ctx context.Context, customerID string) (*dto.LoyaltyBalanceDTO, error) {
	if customerID == "" {
		return nil, domain.ErrInvalidInput
	}
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, domain.ErrNotFound
	}
	return uc.getBalanceByCustomer(ctx, customerID, customer.CompanyID)
}

// GetBalanceByCompany obtiene el balance validando pertenencia del cliente a la empresa.
func (uc *LoyaltyUseCase) GetBalanceByCompany(ctx context.Context, companyID, customerID string) (*dto.LoyaltyBalanceDTO, error) {
	if companyID == "" || customerID == "" {
		return nil, domain.ErrInvalidInput
	}
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	return uc.getBalanceByCustomer(ctx, customerID, customer.CompanyID)
}

func (uc *LoyaltyUseCase) getBalanceByCustomer(ctx context.Context, customerID, companyID string) (*dto.LoyaltyBalanceDTO, error) {
	profile, err := uc.profileRepo.GetByCustomerID(customerID)
	if err != nil {
		return nil, err
	}

	balance := 0
	history := make([]dto.PointEventDTO, 0)
	if uc.interactionRepo != nil {
		batchSize := 100
		offset := 0
		for {
			list, _, listErr := uc.interactionRepo.ListInteractions(customerID, repository.InteractionFilters{
				Type:   string(entity.InteractionTypeOther),
				Limit:  batchSize,
				Offset: offset,
			})
			if listErr != nil {
				return nil, listErr
			}
			if len(list) == 0 {
				break
			}
			for _, it := range list {
				if !strings.HasPrefix(it.Subject, loyaltyPointsSubjectPrefix) {
					continue
				}
				var payload loyaltyPointEventPayload
				if err := json.Unmarshal([]byte(it.Body), &payload); err != nil {
					continue
				}
				balance += payload.Delta
				history = append(history, dto.PointEventDTO{
					Points:      payload.Delta,
					Reason:      payload.Reason,
					ReferenceID: payload.ReferenceID,
					OccurredAt:  it.CreatedAt,
				})
			}
			if len(list) < batchSize {
				break
			}
			offset += batchSize
		}
	}

	tier := ""
	if profile != nil && profile.CategoryID != "" {
		cat, _ := uc.categoryRepo.GetByID(profile.CategoryID)
		if cat != nil {
			tier = cat.Name
		}
	}

	nextTierThreshold := 0
	cats, err := uc.categoryRepo.ListByCompany(companyID, 200, 0)
	if err == nil {
		current := decimal.NewFromInt(int64(balance))
		var next *decimal.Decimal
		for _, c := range cats {
			if c.MinLTV.GreaterThan(current) {
				candidate := c.MinLTV
				if next == nil || candidate.LessThan(*next) {
					next = &candidate
				}
			}
		}
		if next != nil {
			nextTierThreshold = int(next.IntPart())
		}
	}

	return &dto.LoyaltyBalanceDTO{
		Balance:           balance,
		Tier:              tier,
		NextTierThreshold: nextTierThreshold,
		History:           history,
	}, nil
}

// RedeemPoints debita puntos del cliente y registra el evento en historial.
func (uc *LoyaltyUseCase) RedeemPoints(ctx context.Context, customerID string, points int, reason string) error {
	if customerID == "" || points <= 0 || strings.TrimSpace(reason) == "" {
		return domain.ErrInvalidInput
	}
	balance, err := uc.GetBalance(ctx, customerID)
	if err != nil {
		return err
	}
	if balance.Balance < points {
		return domain.ErrConflict
	}
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return domain.ErrNotFound
	}
	if uc.interactionRepo == nil {
		return domain.ErrConflict
	}
	payload, err := json.Marshal(loyaltyPointEventPayload{Delta: -points, Reason: reason})
	if err != nil {
		return err
	}
	now := time.Now()
	return uc.interactionRepo.Create(&entity.CRMInteraction{
		ID:         uuid.New().String(),
		CompanyID:  customer.CompanyID,
		CustomerID: customerID,
		Type:       entity.InteractionTypeOther,
		Subject:    loyaltyPointsSubjectPrefix + "REDEEM",
		Body:       string(payload),
		CreatedBy:  "system",
		CreatedAt:  now,
	})
}

// ResolveCampaignRecipients genera una lista de destinatarios para campañas en base
// a estrategias simples (ej. categoría Oro). Por ahora solo implementa category_gold.
func (uc *LoyaltyUseCase) ResolveCampaignRecipients(ctx context.Context, companyID string, req dto.ResolveCampaignRecipientsRequest) (*dto.ResolveCampaignRecipientsResponse, error) {
	if companyID == "" || len(req.Strategies) == 0 {
		return nil, domain.ErrInvalidInput
	}

	recipients := make(map[string]dto.CampaignRecipientDTO)

	for _, s := range req.Strategies {
		switch s.Type {
		case "category_gold":
			cats, err := uc.categoryRepo.ListByCompany(companyID, 200, 0)
			if err != nil {
				return nil, err
			}
			gold := make(map[string]struct{})
			for _, c := range cats {
				name := strings.ToLower(strings.TrimSpace(c.Name))
				if name == "oro" || strings.Contains(name, "gold") {
					gold[c.ID] = struct{}{}
				}
			}
			if len(gold) == 0 {
				continue
			}
			profiles, err := uc.profileRepo.ListByCompany(companyID, 2000, 0)
			if err != nil {
				return nil, err
			}
			for _, p := range profiles {
				if _, ok := gold[p.CategoryID]; !ok {
					continue
				}
				cust, err := uc.customerRepo.GetByID(p.CustomerID)
				if err != nil || cust == nil || cust.CompanyID != companyID {
					continue
				}
				recipients[cust.ID] = dto.CampaignRecipientDTO{
					CustomerID: cust.ID,
					Name:       cust.Name,
					Email:      cust.Email,
					Segment:    "Categoría Oro",
				}
			}
		case "reorder_product":
			// pendiente de implementar cuando exista soporte por producto en InvoiceRepository
			continue
		default:
			continue
		}
	}

	out := &dto.ResolveCampaignRecipientsResponse{
		Recipients: make([]dto.CampaignRecipientDTO, 0, len(recipients)),
	}
	for _, r := range recipients {
		out.Recipients = append(out.Recipients, r)
	}
	return out, nil
}

// GetProfile360 devuelve la vista 360 del cliente (datos base + perfil CRM + categoría y beneficios si aplica).
func (uc *LoyaltyUseCase) GetProfile360(ctx context.Context, companyID, customerID string) (*dto.Profile360Response, error) {
	p360, err := uc.profileRepo.GetProfile360(ctx, companyID, customerID)
	if err != nil {
		return nil, err
	}
	if p360 == nil {
		return nil, domain.ErrNotFound
	}
	if p360.Customer.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	resp := &dto.Profile360Response{
		Customer: dto.CustomerResponse{
			ID:        p360.Customer.ID,
			CompanyID: p360.Customer.CompanyID,
			Name:      p360.Customer.Name,
			TaxID:     p360.Customer.TaxID,
			Email:     p360.Customer.Email,
			Phone:     p360.Customer.Phone,
		},
		ProfileID:  p360.ProfileID,
		CategoryID: p360.CategoryID,
		LTV:        p360.LTV,
		Benefits:   []dto.BenefitResponse{},
	}
	if p360.CategoryID != "" {
		cat, _ := uc.categoryRepo.GetByID(p360.CategoryID)
		if cat != nil {
			resp.CategoryName = cat.Name
			benefits, _ := uc.benefitRepo.ListByCategory(p360.CategoryID, 50, 0)
			for _, b := range benefits {
				resp.Benefits = append(resp.Benefits, dto.BenefitResponse{
					ID:          b.ID,
					CompanyID:   b.CompanyID,
					CategoryID:  b.CategoryID,
					Name:        b.Name,
					Description: b.Description,
					CreatedAt:   b.CreatedAt,
					UpdatedAt:   b.UpdatedAt,
				})
			}
		}
	}
	return resp, nil
}

// AssignCategory asigna o actualiza la categoría y LTV del perfil CRM del cliente.
func (uc *LoyaltyUseCase) AssignCategory(ctx context.Context, companyID, customerID string, in dto.AssignCategoryRequest) error {
	customer, err := uc.customerRepo.GetByID(customerID)
	if err != nil || customer == nil {
		return domain.ErrNotFound
	}
	if customer.CompanyID != companyID {
		return domain.ErrForbidden
	}
	if in.CategoryID != "" {
		cat, _ := uc.categoryRepo.GetByID(in.CategoryID)
		if cat == nil || cat.CompanyID != companyID {
			return domain.ErrNotFound
		}
	}
	profile, _ := uc.profileRepo.GetByCustomerID(customerID)
	now := time.Now()
	if profile == nil {
		profile = &entity.CRMCustomerProfile{
			ID:         uuid.New().String(),
			CustomerID: customerID,
			CompanyID:  companyID,
			CategoryID: in.CategoryID,
			LTV:        in.LTV,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	} else {
		profile.CategoryID = in.CategoryID
		profile.LTV = in.LTV
		profile.UpdatedAt = now
	}
	return uc.profileRepo.Upsert(profile)
}

// ListCategories lista categorías de fidelización de la empresa.
func (uc *LoyaltyUseCase) ListCategories(ctx context.Context, companyID string, limit, offset int) ([]dto.CategoryResponse, error) {
	list, err := uc.categoryRepo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CategoryResponse, 0, len(list))
	for _, c := range list {
		out = append(out, dto.CategoryResponse{
			ID:        c.ID,
			CompanyID: c.CompanyID,
			Name:      c.Name,
			MinLTV:    c.MinLTV,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return out, nil
}

// CreateCategory crea una categoría de fidelización (solo admin).
func (uc *LoyaltyUseCase) CreateCategory(ctx context.Context, companyID string, in dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	if companyID == "" || strings.TrimSpace(in.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	now := time.Now()
	cat := &entity.CRMCategory{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Name:      in.Name,
		MinLTV:    in.MinLTV,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.categoryRepo.Create(cat); err != nil {
		return nil, err
	}
	return &dto.CategoryResponse{
		ID:        cat.ID,
		CompanyID: cat.CompanyID,
		Name:      cat.Name,
		MinLTV:    cat.MinLTV,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

// UpdateCategory actualiza una categoría de fidelización existente (solo admin).
func (uc *LoyaltyUseCase) UpdateCategory(ctx context.Context, companyID, categoryID string, in dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	if companyID == "" || categoryID == "" {
		return nil, domain.ErrInvalidInput
	}
	cat, err := uc.categoryRepo.GetByID(categoryID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, domain.ErrNotFound
	}
	if cat.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return nil, domain.ErrInvalidInput
		}
		cat.Name = *in.Name
	}
	if in.MinLTV != nil {
		cat.MinLTV = *in.MinLTV
	}
	cat.UpdatedAt = time.Now()
	if err := uc.categoryRepo.Update(cat); err != nil {
		return nil, err
	}
	return &dto.CategoryResponse{
		ID:        cat.ID,
		CompanyID: cat.CompanyID,
		Name:      cat.Name,
		MinLTV:    cat.MinLTV,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

// DeactivateCategory desactiva una categoría (soft delete, solo admin).
func (uc *LoyaltyUseCase) DeactivateCategory(ctx context.Context, companyID, categoryID string) error {
	if companyID == "" || categoryID == "" {
		return domain.ErrInvalidInput
	}
	cat, err := uc.categoryRepo.GetByID(categoryID)
	if err != nil {
		return err
	}
	if cat == nil {
		return domain.ErrNotFound
	}
	if cat.CompanyID != companyID {
		return domain.ErrForbidden
	}
	return uc.categoryRepo.SetActive(companyID, categoryID, false, time.Now())
}

// ListBenefitsByCategory lista beneficios de una categoría.
func (uc *LoyaltyUseCase) ListBenefitsByCategory(ctx context.Context, categoryID string, limit, offset int) ([]dto.BenefitResponse, error) {
	list, err := uc.benefitRepo.ListByCategory(categoryID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BenefitResponse, 0, len(list))
	for _, b := range list {
		out = append(out, dto.BenefitResponse{
			ID:          b.ID,
			CompanyID:   b.CompanyID,
			CategoryID:  b.CategoryID,
			Name:        b.Name,
			Description: b.Description,
			CreatedAt:   b.CreatedAt,
			UpdatedAt:   b.UpdatedAt,
		})
	}
	return out, nil
}

// CreateBenefit crea un beneficio dentro de una categoría (admin).
func (uc *LoyaltyUseCase) CreateBenefit(ctx context.Context, companyID, categoryID string, in dto.CreateBenefitRequest) (*dto.BenefitResponse, error) {
	if categoryID == "" || in.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	cat, err := uc.categoryRepo.GetByID(categoryID)
	if err != nil || cat == nil {
		return nil, domain.ErrNotFound
	}
	if cat.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	now := time.Now()
	b := &entity.CRMBenefit{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CategoryID:  categoryID,
		Name:        in.Name,
		Description: in.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.benefitRepo.Create(b); err != nil {
		return nil, err
	}
	return &dto.BenefitResponse{
		ID:          b.ID,
		CompanyID:   b.CompanyID,
		CategoryID:  b.CategoryID,
		Name:        b.Name,
		Description: b.Description,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}, nil
}

// UpdateBenefit actualiza un beneficio existente (admin).
func (uc *LoyaltyUseCase) UpdateBenefit(ctx context.Context, companyID, benefitID string, in dto.UpdateBenefitRequest) (*dto.BenefitResponse, error) {
	if benefitID == "" || in.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	b, err := uc.benefitRepo.GetByID(benefitID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, domain.ErrNotFound
	}
	if b.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	b.Name = in.Name
	b.Description = in.Description
	b.UpdatedAt = time.Now()
	if err := uc.benefitRepo.Update(b); err != nil {
		return nil, err
	}
	return &dto.BenefitResponse{
		ID:          b.ID,
		CompanyID:   b.CompanyID,
		CategoryID:  b.CategoryID,
		Name:        b.Name,
		Description: b.Description,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}, nil
}

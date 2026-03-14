package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

var _ repository.CRMCategoryRepository = (*CRMCategoryRepo)(nil)
var _ repository.CRMBenefitRepository = (*CRMBenefitRepo)(nil)
var _ repository.CRMProfileRepository = (*CRMProfileRepo)(nil)
var _ repository.CRMInteractionRepository = (*CRMInteractionRepo)(nil)
var _ repository.CRMTaskRepository = (*CRMTaskRepo)(nil)
var _ repository.CRMTicketRepository = (*CRMTicketRepo)(nil)

// CRMCategoryRepo implementación de CRMCategoryRepository.
type CRMCategoryRepo struct{ q Querier }

func NewCRMCategoryRepository(q Querier) *CRMCategoryRepo { return &CRMCategoryRepo{q: q} }

func (r *CRMCategoryRepo) Create(c *entity.CRMCategory) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_categories (id, company_id, name, min_ltv, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		c.ID, c.CompanyID, c.Name, c.MinLTV, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *CRMCategoryRepo) GetByID(id string) (*entity.CRMCategory, error) {
	var c entity.CRMCategory
	var minLtv pgtype.Numeric
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, name, min_ltv, created_at, updated_at FROM crm_categories WHERE id = $1`, id,
	).Scan(&c.ID, &c.CompanyID, &c.Name, &minLtv, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if minLtv.Valid && minLtv.Int != nil {
		c.MinLTV = decimal.NewFromBigInt(minLtv.Int, minLtv.Exp)
	}
	return &c, nil
}

func (r *CRMCategoryRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCategory, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, name, min_ltv, created_at, updated_at FROM crm_categories WHERE company_id = $1 ORDER BY name LIMIT $2 OFFSET $3`,
		companyID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMCategory
	for rows.Next() {
		var c entity.CRMCategory
		var minLtv pgtype.Numeric
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.Name, &minLtv, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if minLtv.Valid && minLtv.Int != nil {
			c.MinLTV = decimal.NewFromBigInt(minLtv.Int, minLtv.Exp)
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

func (r *CRMCategoryRepo) Update(c *entity.CRMCategory) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE crm_categories SET name = $2, min_ltv = $3, updated_at = $4 WHERE id = $1`,
		c.ID, c.Name, c.MinLTV, c.UpdatedAt,
	)
	return err
}

func (r *CRMCategoryRepo) Delete(id string) error {
	_, err := r.q.Exec(context.Background(), `DELETE FROM crm_categories WHERE id = $1`, id)
	return err
}

// CRMBenefitRepo implementación de CRMBenefitRepository.
type CRMBenefitRepo struct{ q Querier }

func NewCRMBenefitRepository(q Querier) *CRMBenefitRepo { return &CRMBenefitRepo{q: q} }

func (r *CRMBenefitRepo) Create(b *entity.CRMBenefit) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_benefits (id, company_id, category_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		b.ID, b.CompanyID, b.CategoryID, b.Name, b.Description, b.CreatedAt, b.UpdatedAt,
	)
	return err
}

func (r *CRMBenefitRepo) GetByID(id string) (*entity.CRMBenefit, error) {
	var b entity.CRMBenefit
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, category_id, name, description, created_at, updated_at FROM crm_benefits WHERE id = $1`, id,
	).Scan(&b.ID, &b.CompanyID, &b.CategoryID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *CRMBenefitRepo) ListByCategory(categoryID string, limit, offset int) ([]*entity.CRMBenefit, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, category_id, name, description, created_at, updated_at FROM crm_benefits WHERE category_id = $1 ORDER BY name LIMIT $2 OFFSET $3`,
		categoryID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMBenefit
	for rows.Next() {
		var b entity.CRMBenefit
		if err := rows.Scan(&b.ID, &b.CompanyID, &b.CategoryID, &b.Name, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &b)
	}
	return list, rows.Err()
}

func (r *CRMBenefitRepo) Update(b *entity.CRMBenefit) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE crm_benefits SET name = $2, description = $3, updated_at = $4 WHERE id = $1`,
		b.ID, b.Name, b.Description, b.UpdatedAt,
	)
	return err
}

func (r *CRMBenefitRepo) Delete(id string) error {
	_, err := r.q.Exec(context.Background(), `DELETE FROM crm_benefits WHERE id = $1`, id)
	return err
}

// CRMProfileRepo implementación de CRMProfileRepository.
type CRMProfileRepo struct{ q Querier }

func NewCRMProfileRepository(q Querier) *CRMProfileRepo { return &CRMProfileRepo{q: q} }

func (r *CRMProfileRepo) GetByCustomerID(customerID string) (*entity.CRMCustomerProfile, error) {
	var p entity.CRMCustomerProfile
	var catID *string
	err := r.q.QueryRow(context.Background(), `
		SELECT id, customer_id, company_id, category_id, ltv, created_at, updated_at FROM crm_customer_profiles WHERE customer_id = $1`, customerID,
	).Scan(&p.ID, &p.CustomerID, &p.CompanyID, &catID, &p.LTV, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if catID != nil {
		p.CategoryID = *catID
	}
	return &p, nil
}

func (r *CRMProfileRepo) GetProfile360(ctx context.Context, companyID, customerID string) (*entity.Profile360, error) {
	query := `
		SELECT c.id, c.company_id, c.name, c.tax_id, c.email, c.phone, c.created_at, c.updated_at,
		       p.id AS profile_id, p.category_id, COALESCE(p.ltv, 0) AS ltv
		FROM customers c
		LEFT JOIN crm_customer_profiles p ON p.customer_id = c.id
		WHERE c.id = $1 AND c.company_id = $2`
	var cust entity.Customer
	var profileID, catID *string
	var ltv decimal.Decimal
	err := r.q.QueryRow(ctx, query, customerID, companyID).Scan(
		&cust.ID, &cust.CompanyID, &cust.Name, &cust.TaxID, &cust.Email, &cust.Phone, &cust.CreatedAt, &cust.UpdatedAt,
		&profileID, &catID, &ltv,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get profile360: %w", err)
	}
	out := &entity.Profile360{Customer: cust, LTV: ltv}
	if profileID != nil {
		out.ProfileID = *profileID
	}
	if catID != nil {
		out.CategoryID = *catID
	}
	return out, nil
}

func (r *CRMProfileRepo) Upsert(p *entity.CRMCustomerProfile) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_customer_profiles (id, customer_id, company_id, category_id, ltv, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (customer_id) DO UPDATE SET category_id = EXCLUDED.category_id, ltv = EXCLUDED.ltv, updated_at = EXCLUDED.updated_at`,
		p.ID, p.CustomerID, p.CompanyID, nullIfEmpty(p.CategoryID), p.LTV, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *CRMProfileRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.CRMCustomerProfile, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, customer_id, company_id, category_id, ltv, created_at, updated_at FROM crm_customer_profiles WHERE company_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		companyID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMCustomerProfile
	for rows.Next() {
		var p entity.CRMCustomerProfile
		var catID *string
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.CompanyID, &catID, &p.LTV, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if catID != nil {
			p.CategoryID = *catID
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

// CRMInteractionRepo implementación de CRMInteractionRepository.
type CRMInteractionRepo struct{ q Querier }

func NewCRMInteractionRepository(q Querier) *CRMInteractionRepo { return &CRMInteractionRepo{q: q} }

func (r *CRMInteractionRepo) Create(m *entity.CRMInteraction) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_interactions (id, company_id, customer_id, type, subject, body, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.ID, m.CompanyID, m.CustomerID, string(m.Type), m.Subject, m.Body, nullIfEmpty(m.CreatedBy), m.CreatedAt,
	)
	return err
}

func (r *CRMInteractionRepo) GetByID(id string) (*entity.CRMInteraction, error) {
	var m entity.CRMInteraction
	var typ string
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, customer_id, type, subject, body, created_by, created_at FROM crm_interactions WHERE id = $1`, id,
	).Scan(&m.ID, &m.CompanyID, &m.CustomerID, &typ, &m.Subject, &m.Body, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m.Type = entity.InteractionType(typ)
	return &m, nil
}

func (r *CRMInteractionRepo) ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, customer_id, type, subject, body, created_by, created_at FROM crm_interactions WHERE customer_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		customerID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMInteraction
	for rows.Next() {
		var m entity.CRMInteraction
		var typ string
		if err := rows.Scan(&m.ID, &m.CompanyID, &m.CustomerID, &typ, &m.Subject, &m.Body, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Type = entity.InteractionType(typ)
		list = append(list, &m)
	}
	return list, rows.Err()
}

func (r *CRMInteractionRepo) ListInteractions(customerID string, f repository.InteractionFilters) ([]*entity.CRMInteraction, int64, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, company_id, customer_id, type, subject, body, created_by, created_at,
			COUNT(*) OVER() AS total
		FROM crm_interactions
		WHERE customer_id = $1`
	args := []any{customerID}
	idx := 2

	if f.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, f.Type)
		idx++
	}
	if !f.StartDate.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, f.StartDate)
		idx++
	}
	if !f.EndDate.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", idx)
		args = append(args, f.EndDate)
		idx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*entity.CRMInteraction
	var total int64
	for rows.Next() {
		var m entity.CRMInteraction
		var typ string
		if err := rows.Scan(&m.ID, &m.CompanyID, &m.CustomerID, &typ, &m.Subject, &m.Body, &m.CreatedBy, &m.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		m.Type = entity.InteractionType(typ)
		list = append(list, &m)
	}
	return list, total, rows.Err()
}

// CRMTaskRepo implementación de CRMTaskRepository.
type CRMTaskRepo struct{ q Querier }

func NewCRMTaskRepository(q Querier) *CRMTaskRepo { return &CRMTaskRepo{q: q} }

func (r *CRMTaskRepo) Create(t *entity.CRMTask) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_tasks (id, company_id, customer_id, title, description, due_at, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		t.ID, t.CompanyID, nullIfEmpty(t.CustomerID), t.Title, t.Description, t.DueAt, string(t.Status), nullIfEmpty(t.CreatedBy), t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *CRMTaskRepo) GetByID(id string) (*entity.CRMTask, error) {
	var t entity.CRMTask
	var status string
	var customerID *string
	var createdBy *string
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, customer_id, title, description, due_at, status, created_by, created_at, updated_at FROM crm_tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.CompanyID, &customerID, &t.Title, &t.Description, &t.DueAt, &status, &createdBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t.Status = entity.TaskStatus(status)
	if customerID != nil {
		t.CustomerID = *customerID
	}
	if createdBy != nil {
		t.CreatedBy = *createdBy
	}
	return &t, nil
}

func (r *CRMTaskRepo) Update(t *entity.CRMTask) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE crm_tasks SET title = $2, description = $3, due_at = $4, status = $5, updated_at = $6 WHERE id = $1`,
		t.ID, t.Title, t.Description, t.DueAt, string(t.Status), t.UpdatedAt,
	)
	return err
}

func (r *CRMTaskRepo) ListByCompany(companyID string, status string, limit, offset int) ([]*entity.CRMTask, error) {
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = r.q.Query(context.Background(), `
			SELECT id, company_id, customer_id, title, description, due_at, status, created_by, created_at, updated_at
			FROM crm_tasks WHERE company_id = $1 AND status = $2 ORDER BY due_at ASC NULLS LAST LIMIT $3 OFFSET $4`,
			companyID, status, limit, offset,
		)
	} else {
		rows, err = r.q.Query(context.Background(), `
			SELECT id, company_id, customer_id, title, description, due_at, status, created_by, created_at, updated_at
			FROM crm_tasks WHERE company_id = $1 ORDER BY due_at ASC NULLS LAST LIMIT $2 OFFSET $3`,
			companyID, limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMTask
	for rows.Next() {
		var t entity.CRMTask
		var st string
		var customerID *string
		var createdBy *string
		if err := rows.Scan(&t.ID, &t.CompanyID, &customerID, &t.Title, &t.Description, &t.DueAt, &st, &createdBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Status = entity.TaskStatus(st)
		if customerID != nil {
			t.CustomerID = *customerID
		}
		if createdBy != nil {
			t.CreatedBy = *createdBy
		}
		list = append(list, &t)
	}
	return list, rows.Err()
}

// CRMTicketRepo implementación de CRMTicketRepository.
type CRMTicketRepo struct{ q Querier }

func NewCRMTicketRepository(q Querier) *CRMTicketRepo { return &CRMTicketRepo{q: q} }

func (r *CRMTicketRepo) Create(t *entity.CRMTicket) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO crm_tickets (id, company_id, customer_id, subject, description, status, sentiment, escalation_reason, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID, t.CompanyID, t.CustomerID, t.Subject, t.Description, t.Status,
		nullIfEmpty(t.Sentiment), nullIfEmpty(t.EscalationReason), nullIfEmpty(t.CreatedBy), t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *CRMTicketRepo) GetByID(id string) (*entity.CRMTicket, error) {
	var t entity.CRMTicket
	var sentiment *string
	var createdBy *string
	var escalationReason *string
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, customer_id, subject, description, status, sentiment, escalation_reason, created_by, created_at, updated_at FROM crm_tickets WHERE id = $1`, id,
	).Scan(&t.ID, &t.CompanyID, &t.CustomerID, &t.Subject, &t.Description, &t.Status, &sentiment, &escalationReason, &createdBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if sentiment != nil {
		t.Sentiment = *sentiment
	}
	if escalationReason != nil {
		t.EscalationReason = *escalationReason
	}
	if createdBy != nil {
		t.CreatedBy = *createdBy
	}
	return &t, nil
}

func (r *CRMTicketRepo) Update(t *entity.CRMTicket) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE crm_tickets SET subject = $2, description = $3, status = $4, sentiment = $5, escalation_reason = $6, updated_at = $7 WHERE id = $1`,
		t.ID, t.Subject, t.Description, t.Status, nullIfEmpty(t.Sentiment), nullIfEmpty(t.EscalationReason), t.UpdatedAt,
	)
	return err
}

func (r *CRMTicketRepo) ListByCompany(companyID string, search string, status string, sort string, limit, offset int) ([]*entity.CRMTicket, error) {
	query := `
		SELECT id, company_id, customer_id, subject, description, status, sentiment, escalation_reason, created_by, created_at, updated_at
		FROM crm_tickets
		WHERE company_id = $1`
	args := []any{companyID}
	argIdx := 2

	if search != "" {
		query += fmt.Sprintf(" AND subject ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	orderDir := "DESC"
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "asc", "created_at_asc":
		orderDir = "ASC"
	case "desc", "created_at_desc", "":
		orderDir = "DESC"
	}

	query += " ORDER BY created_at " + orderDir
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.q.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMTicket
	for rows.Next() {
		var t entity.CRMTicket
		var sentiment *string
		var createdBy *string
		var escalationReason *string
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.CustomerID, &t.Subject, &t.Description, &t.Status, &sentiment, &escalationReason, &createdBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if sentiment != nil {
			t.Sentiment = *sentiment
		}
		if createdBy != nil {
			t.CreatedBy = *createdBy
		}
		if escalationReason != nil {
			t.EscalationReason = *escalationReason
		}
		list = append(list, &t)
	}
	return list, rows.Err()
}

// UpdateStatus actualiza únicamente el estado y el timestamp de un ticket.
func (r *CRMTicketRepo) UpdateStatus(id, status string, updatedAt time.Time) error {
	_, err := r.q.Exec(context.Background(),
		`UPDATE crm_tickets SET status = $2, updated_at = $3 WHERE id = $1`,
		id, status, updatedAt,
	)
	return err
}

// ListOverdue devuelve todos los tickets con estado OVERDUE de una empresa.
func (r *CRMTicketRepo) ListOverdue(companyID string) ([]*entity.CRMTicket, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, customer_id, subject, description, status, sentiment, escalation_reason, created_by, created_at, updated_at
		FROM crm_tickets
		WHERE company_id = $1 AND status = 'OVERDUE'
		ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.CRMTicket
	for rows.Next() {
		var t entity.CRMTicket
		var sentiment, escalationReason, createdBy *string
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.CustomerID, &t.Subject, &t.Description, &t.Status, &sentiment, &escalationReason, &createdBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if sentiment != nil {
			t.Sentiment = *sentiment
		}
		if escalationReason != nil {
			t.EscalationReason = *escalationReason
		}
		if createdBy != nil {
			t.CreatedBy = *createdBy
		}
		list = append(list, &t)
	}
	return list, rows.Err()
}

// MarkOverdueTickets marca como OVERDUE todos los tickets cuyo SLA ha expirado.
func (r *CRMTicketRepo) MarkOverdueTickets(ctx context.Context) (int64, error) {
	tag, err := r.q.Exec(ctx, `
		UPDATE crm_tickets t
		SET status = 'OVERDUE', updated_at = now()
		FROM sla_config s
		WHERE t.company_id = s.company_id
		  AND t.status NOT IN ('closed', 'resolved', 'ESCALATED', 'OVERDUE')
		  AND now() > t.created_at + (s.max_hours * interval '1 hour')`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---------------------------------------------------------------------------
// SLAConfigRepo
// ---------------------------------------------------------------------------

// SLAConfigRepo implementación de SLAConfigRepository.
type SLAConfigRepo struct{ q Querier }

func NewSLAConfigRepository(q Querier) *SLAConfigRepo { return &SLAConfigRepo{q: q} }

func (r *SLAConfigRepo) Upsert(ctx context.Context, cfg *entity.SLAConfig) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO sla_config (company_id, ticket_type, max_hours, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (company_id, ticket_type)
		DO UPDATE SET max_hours = EXCLUDED.max_hours, updated_at = now()`,
		cfg.CompanyID, cfg.TicketType, cfg.MaxHours,
	)
	return err
}

func (r *SLAConfigRepo) GetByCompanyAndType(ctx context.Context, companyID, ticketType string) (*entity.SLAConfig, error) {
	var cfg entity.SLAConfig
	err := r.q.QueryRow(ctx,
		`SELECT company_id, ticket_type, max_hours FROM sla_config WHERE company_id = $1 AND ticket_type = $2`,
		companyID, ticketType,
	).Scan(&cfg.CompanyID, &cfg.TicketType, &cfg.MaxHours)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *SLAConfigRepo) ListByCompany(ctx context.Context, companyID string) ([]*entity.SLAConfig, error) {
	rows, err := r.q.Query(ctx,
		`SELECT company_id, ticket_type, max_hours FROM sla_config WHERE company_id = $1 ORDER BY ticket_type`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*entity.SLAConfig
	for rows.Next() {
		var cfg entity.SLAConfig
		if err := rows.Scan(&cfg.CompanyID, &cfg.TicketType, &cfg.MaxHours); err != nil {
			return nil, err
		}
		list = append(list, &cfg)
	}
	return list, rows.Err()
}

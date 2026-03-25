package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.EmailAccountRepository = (*EmailAccountRepo)(nil)
var _ repository.EmailRepository = (*EmailRepo)(nil)

type EmailAccountRepo struct{ q Querier }

type EmailRepo struct{ q Querier }

func NewEmailAccountRepository(q Querier) *EmailAccountRepo { return &EmailAccountRepo{q: q} }
func NewEmailRepository(q Querier) *EmailRepo               { return &EmailRepo{q: q} }

func (r *EmailAccountRepo) Create(account *entity.EmailAccount) error {
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO email_accounts (id, company_id, email_address, imap_server, imap_port, password, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		account.ID,
		account.CompanyID,
		account.EmailAddress,
		account.IMAPServer,
		account.IMAPPort,
		account.Password,
		account.IsActive,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert email account: %w", err)
	}
	return nil
}

func (r *EmailAccountRepo) Update(account *entity.EmailAccount) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE email_accounts
		SET email_address = $3,
		    imap_server = $4,
		    imap_port = $5,
		    password = $6,
		    is_active = $7,
		    updated_at = $8
		WHERE id = $1 AND company_id = $2`,
		account.ID,
		account.CompanyID,
		account.EmailAddress,
		account.IMAPServer,
		account.IMAPPort,
		account.Password,
		account.IsActive,
		account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update email account: %w", err)
	}
	return nil
}

func (r *EmailAccountRepo) Delete(companyID, id string) error {
	_, err := r.q.Exec(context.Background(), `DELETE FROM email_accounts WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		return fmt.Errorf("delete email account: %w", err)
	}
	return nil
}

func (r *EmailAccountRepo) GetByID(companyID, id string) (*entity.EmailAccount, error) {
	var acc entity.EmailAccount
	err := r.q.QueryRow(context.Background(), `
		SELECT id, company_id, email_address, imap_server, imap_port, password, is_active, created_at, updated_at
		FROM email_accounts
		WHERE company_id = $1 AND id = $2`, companyID, id,
	).Scan(
		&acc.ID,
		&acc.CompanyID,
		&acc.EmailAddress,
		&acc.IMAPServer,
		&acc.IMAPPort,
		&acc.Password,
		&acc.IsActive,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get email account: %w", err)
	}
	return &acc, nil
}

func (r *EmailAccountRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.EmailAccount, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, email_address, imap_server, imap_port, password, is_active, created_at, updated_at
		FROM email_accounts
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list email accounts: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.EmailAccount, 0)
	for rows.Next() {
		var acc entity.EmailAccount
		if err := rows.Scan(
			&acc.ID,
			&acc.CompanyID,
			&acc.EmailAddress,
			&acc.IMAPServer,
			&acc.IMAPPort,
			&acc.Password,
			&acc.IsActive,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan email account: %w", err)
		}
		list = append(list, &acc)
	}
	return list, rows.Err()
}

func (r *EmailAccountRepo) ListActive() ([]*entity.EmailAccount, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, company_id, email_address, imap_server, imap_port, password, is_active, created_at, updated_at
		FROM email_accounts
		WHERE is_active = true
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active email accounts: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.EmailAccount, 0)
	for rows.Next() {
		var acc entity.EmailAccount
		if err := rows.Scan(
			&acc.ID,
			&acc.CompanyID,
			&acc.EmailAddress,
			&acc.IMAPServer,
			&acc.IMAPPort,
			&acc.Password,
			&acc.IsActive,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan active email account: %w", err)
		}
		list = append(list, &acc)
	}
	return list, rows.Err()
}

func (r *EmailRepo) Create(email *entity.Email, attachments []entity.EmailAttachment) error {
	if email.ID == "" {
		email.ID = uuid.New().String()
	}
	_, err := r.q.Exec(context.Background(), `
		INSERT INTO emails (id, account_id, message_id, customer_id, from_address, to_address, subject, body_html, body_text, received_at, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		email.ID,
		email.AccountID,
		email.MessageID,
		nullIfEmpty(email.CustomerID),
		email.FromAddress,
		email.ToAddress,
		email.Subject,
		nullIfEmpty(email.BodyHTML),
		nullIfEmpty(email.BodyText),
		email.ReceivedAt,
		email.IsRead,
		email.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert email: %w", err)
	}

	for i := range attachments {
		if attachments[i].ID == "" {
			attachments[i].ID = uuid.New().String()
		}
		attachments[i].EmailID = email.ID
		_, err := r.q.Exec(context.Background(), `
			INSERT INTO email_attachments (id, email_id, file_name, file_url, mime_type, size)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			attachments[i].ID,
			attachments[i].EmailID,
			attachments[i].FileName,
			attachments[i].FileURL,
			attachments[i].MIMEType,
			attachments[i].Size,
		)
		if err != nil {
			return fmt.Errorf("insert email attachment: %w", err)
		}
	}

	return nil
}

func (r *EmailRepo) GetByAccountAndMessageID(accountID, messageID string) (*entity.Email, error) {
	var e entity.Email
	var customerID *string
	err := r.q.QueryRow(context.Background(), `
		SELECT id, account_id, message_id, customer_id, from_address, to_address, subject, COALESCE(body_html, ''), COALESCE(body_text, ''), received_at, is_read, created_at
		FROM emails
		WHERE account_id = $1 AND message_id = $2`, accountID, messageID,
	).Scan(
		&e.ID,
		&e.AccountID,
		&e.MessageID,
		&customerID,
		&e.FromAddress,
		&e.ToAddress,
		&e.Subject,
		&e.BodyHTML,
		&e.BodyText,
		&e.ReceivedAt,
		&e.IsRead,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get email by message_id: %w", err)
	}
	if customerID != nil {
		e.CustomerID = *customerID
	}
	return &e, nil
}

func (r *EmailRepo) ListByCompany(filter repository.EmailListFilter) ([]*entity.Email, int64, error) {
	args := []any{filter.CompanyID}
	countArgs := []any{filter.CompanyID}
	conds := []string{"ea.company_id = $1"}
	idx := 2

	if filter.CustomerID != "" {
		conds = append(conds, fmt.Sprintf("e.customer_id = $%d", idx))
		args = append(args, filter.CustomerID)
		countArgs = append(countArgs, filter.CustomerID)
		idx++
	}
	if filter.IsRead != nil {
		conds = append(conds, fmt.Sprintf("e.is_read = $%d", idx))
		args = append(args, *filter.IsRead)
		countArgs = append(countArgs, *filter.IsRead)
		idx++
	}

	where := strings.Join(conds, " AND ")

	countSQL := `
		SELECT COUNT(1)
		FROM emails e
		INNER JOIN email_accounts ea ON ea.id = e.account_id
		WHERE ` + where
	var total int64
	if err := r.q.QueryRow(context.Background(), countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count emails: %w", err)
	}

	listSQL := `
		SELECT e.id, e.account_id, ea.company_id, e.message_id, e.customer_id, e.from_address, e.to_address, e.subject,
		       COALESCE(e.body_html, ''), COALESCE(e.body_text, ''), e.received_at, e.is_read, e.created_at
		FROM emails e
		INNER JOIN email_accounts ea ON ea.id = e.account_id
		WHERE ` + where + fmt.Sprintf(" ORDER BY e.received_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.q.Query(context.Background(), listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list emails: %w", err)
	}
	defer rows.Close()

	list := make([]*entity.Email, 0)
	for rows.Next() {
		var e entity.Email
		var customerID *string
		if err := rows.Scan(
			&e.ID,
			&e.AccountID,
			&e.CompanyID,
			&e.MessageID,
			&customerID,
			&e.FromAddress,
			&e.ToAddress,
			&e.Subject,
			&e.BodyHTML,
			&e.BodyText,
			&e.ReceivedAt,
			&e.IsRead,
			&e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan email: %w", err)
		}
		if customerID != nil {
			e.CustomerID = *customerID
		}
		attachments, err := r.listAttachmentsByEmailID(e.ID)
		if err != nil {
			return nil, 0, err
		}
		e.Attachments = attachments
		list = append(list, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *EmailRepo) GetByID(companyID, id string) (*entity.Email, error) {
	var e entity.Email
	var customerID *string
	err := r.q.QueryRow(context.Background(), `
		SELECT e.id, e.account_id, ea.company_id, e.message_id, e.customer_id, e.from_address, e.to_address, e.subject,
		       COALESCE(e.body_html, ''), COALESCE(e.body_text, ''), e.received_at, e.is_read, e.created_at
		FROM emails e
		INNER JOIN email_accounts ea ON ea.id = e.account_id
		WHERE ea.company_id = $1 AND e.id = $2`, companyID, id,
	).Scan(
		&e.ID,
		&e.AccountID,
		&e.CompanyID,
		&e.MessageID,
		&customerID,
		&e.FromAddress,
		&e.ToAddress,
		&e.Subject,
		&e.BodyHTML,
		&e.BodyText,
		&e.ReceivedAt,
		&e.IsRead,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get email by id: %w", err)
	}
	if customerID != nil {
		e.CustomerID = *customerID
	}
	attachments, err := r.listAttachmentsByEmailID(e.ID)
	if err != nil {
		return nil, err
	}
	e.Attachments = attachments
	return &e, nil
}

func (r *EmailRepo) MarkAsRead(companyID, id string) error {
	_, err := r.q.Exec(context.Background(), `
		UPDATE emails e
		SET is_read = true
		FROM email_accounts ea
		WHERE e.account_id = ea.id
		  AND ea.company_id = $1
		  AND e.id = $2`, companyID, id)
	if err != nil {
		return fmt.Errorf("mark email as read: %w", err)
	}
	return nil
}

func (r *EmailRepo) listAttachmentsByEmailID(emailID string) ([]entity.EmailAttachment, error) {
	rows, err := r.q.Query(context.Background(), `
		SELECT id, email_id, file_name, file_url, mime_type, size
		FROM email_attachments
		WHERE email_id = $1
		ORDER BY file_name`, emailID)
	if err != nil {
		return nil, fmt.Errorf("list email attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]entity.EmailAttachment, 0)
	for rows.Next() {
		var a entity.EmailAttachment
		if err := rows.Scan(&a.ID, &a.EmailID, &a.FileName, &a.FileURL, &a.MIMEType, &a.Size); err != nil {
			return nil, fmt.Errorf("scan email attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

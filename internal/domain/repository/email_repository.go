package repository

import (
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

type EmailListFilter struct {
	CompanyID  string
	CustomerID string
	IsRead     *bool
	Limit      int
	Offset     int
}

type EmailAccountRepository interface {
	Create(account *entity.EmailAccount) error
	Update(account *entity.EmailAccount) error
	Delete(companyID, id string) error
	GetByID(companyID, id string) (*entity.EmailAccount, error)
	ListByCompany(companyID string, limit, offset int) ([]*entity.EmailAccount, error)
	ListActive() ([]*entity.EmailAccount, error)
}

type EmailRepository interface {
	Create(email *entity.Email, attachments []entity.EmailAttachment) error
	GetByAccountAndMessageID(accountID, messageID string) (*entity.Email, error)
	ListByCompany(filter EmailListFilter) ([]*entity.Email, int64, error)
	GetByID(companyID, id string) (*entity.Email, error)
	MarkAsRead(companyID, id string) error
}

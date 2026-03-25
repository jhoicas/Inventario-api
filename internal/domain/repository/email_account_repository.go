package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// HybridEmailAccountRepository persiste configuraciones de cuentas de correo
// para proveedores OAuth y cuentas custom IMAP/SMTP.
type HybridEmailAccountRepository interface {
	Save(ctx context.Context, account *entity.EmailAccountConfig) error
	GetByID(ctx context.Context, companyID, id string) (*entity.EmailAccountConfig, error)
	GetByCompanyAndEmail(ctx context.Context, companyID, emailAddress string) (*entity.EmailAccountConfig, error)
}

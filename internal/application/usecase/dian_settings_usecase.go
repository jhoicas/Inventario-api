package usecase

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

type DIANCertificateStore interface {
	Save(companyID, environment, originalFileName string, content []byte) (storedPath string, storedName string, err error)
}

type SecretEncryptor interface {
	Encrypt(plaintext string) (string, error)
}

type DIANSettingsUseCase struct {
	companyRepo  repository.CompanyRepository
	settingsRepo repository.DIANSettingsRepository
	certStore    DIANCertificateStore
	encryptor    SecretEncryptor
}

func NewDIANSettingsUseCase(
	companyRepo repository.CompanyRepository,
	settingsRepo repository.DIANSettingsRepository,
	certStore DIANCertificateStore,
	encryptor SecretEncryptor,
) *DIANSettingsUseCase {
	return &DIANSettingsUseCase{
		companyRepo:  companyRepo,
		settingsRepo: settingsRepo,
		certStore:    certStore,
		encryptor:    encryptor,
	}
}

func (uc *DIANSettingsUseCase) Save(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}

	env := strings.ToLower(strings.TrimSpace(in.Environment))
	if env != "test" && env != "prod" {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(in.CertificatePassword) == "" {
		return nil, domain.ErrInvalidInput
	}
	if len(in.CertificateData) == 0 {
		return nil, domain.ErrInvalidInput
	}

	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	storedPath, storedName, err := uc.certStore.Save(companyID, env, in.CertificateFileName, in.CertificateData)
	if err != nil {
		return nil, err
	}

	passwordEncrypted, err := uc.encryptor.Encrypt(in.CertificatePassword)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	settings := &entity.DIANSettings{
		CompanyID:                    companyID,
		Environment:                  env,
		CertificatePath:              filepath.ToSlash(storedPath),
		CertificateFileName:          storedName,
		CertificateFileSize:          int64(len(in.CertificateData)),
		CertificatePasswordEncrypted: passwordEncrypted,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	if err := uc.settingsRepo.Upsert(context.Background(), settings); err != nil {
		return nil, err
	}

	if env == "prod" {
		company.CertProd = settings.CertificatePath
	} else {
		company.CertHab = settings.CertificatePath
	}
	company.UpdatedAt = now
	if err := uc.companyRepo.Update(company); err != nil {
		return nil, err
	}

	return &dto.DIANSettingsResponse{
		CompanyID:           companyID,
		Environment:         env,
		CertificateFileName: settings.CertificateFileName,
		CertificateFileSize: settings.CertificateFileSize,
		UpdatedAt:           settings.UpdatedAt,
	}, nil
}

func (uc *DIANSettingsUseCase) Get(companyID string) (*dto.DIANSettingsResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}

	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	settings, err := uc.settingsRepo.GetByCompanyID(context.Background(), companyID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, domain.ErrNotFound
	}

	return &dto.DIANSettingsResponse{
		CompanyID:           settings.CompanyID,
		Environment:         settings.Environment,
		CertificateFileName: settings.CertificateFileName,
		CertificateFileSize: settings.CertificateFileSize,
		UpdatedAt:           settings.UpdatedAt,
	}, nil
}

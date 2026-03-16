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

	env, ok := normalizeDIANEnvironment(in.Environment)
	if !ok {
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
		if !isLegacyCompanyUpdateColumnError(err) {
			return nil, err
		}
	}

	return &dto.DIANSettingsResponse{
		CompanyID:           companyID,
		Environment:         env,
		CertificateFileName: settings.CertificateFileName,
		CertificateFileSize: settings.CertificateFileSize,
		UpdatedAt:           settings.UpdatedAt,
	}, nil
}

// Get devuelve la configuración DIAN de la empresa.
// Si environment es vacío, devuelve la configuración más reciente (cualquier ambiente).
// Si se indica environment ("test"|"prod" o sus alias), devuelve la del ambiente específico.
func (uc *DIANSettingsUseCase) Get(companyID, environment string) (*dto.DIANSettingsResponse, error) {
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

	var settings *entity.DIANSettings
	if env := strings.TrimSpace(environment); env != "" {
		normEnv, ok := normalizeDIANEnvironment(env)
		if !ok {
			return nil, domain.ErrInvalidInput
		}
		settings, err = uc.settingsRepo.GetByCompanyIDAndEnvironment(context.Background(), companyID, normEnv)
	} else {
		settings, err = uc.settingsRepo.GetByCompanyID(context.Background(), companyID)
	}
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

func normalizeDIANEnvironment(environment string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "test", "testing", "habilitacion", "hab":
		return "test", true
	case "prod", "production", "produccion", "productional":
		return "prod", true
	default:
		return "", false
	}
}

func isLegacyCompanyUpdateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "sqlstate 42703") {
		return false
	}
	return strings.Contains(msg, "cert_hab") || strings.Contains(msg, "cert_prod") || strings.Contains(msg, "environment")
}

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCompanyRepoForDIANSettings struct {
	company *entity.Company

	updated   *entity.Company
	updateErr error
}

func (f *fakeCompanyRepoForDIANSettings) Create(company *entity.Company) error { return nil }
func (f *fakeCompanyRepoForDIANSettings) GetByID(id string) (*entity.Company, error) {
	if f.company == nil {
		return nil, nil
	}
	clone := *f.company
	return &clone, nil
}
func (f *fakeCompanyRepoForDIANSettings) GetByNIT(nit string) (*entity.Company, error) {
	return nil, nil
}
func (f *fakeCompanyRepoForDIANSettings) Update(company *entity.Company) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	clone := *company
	f.updated = &clone
	return nil
}
func (f *fakeCompanyRepoForDIANSettings) List(limit, offset int) ([]*entity.Company, error) {
	return nil, nil
}
func (f *fakeCompanyRepoForDIANSettings) Delete(id string) error { return nil }
func (f *fakeCompanyRepoForDIANSettings) ListModules(ctx context.Context, companyID string) ([]*entity.CompanyModule, error) {
	return []*entity.CompanyModule{}, nil
}
func (f *fakeCompanyRepoForDIANSettings) GetModule(ctx context.Context, companyID, moduleName string) (*entity.CompanyModule, error) {
	return nil, nil
}
func (f *fakeCompanyRepoForDIANSettings) UpsertModule(ctx context.Context, module *entity.CompanyModule) error {
	return nil
}
func (f *fakeCompanyRepoForDIANSettings) DeleteModule(ctx context.Context, companyID, moduleName string) error {
	return nil
}
func (f *fakeCompanyRepoForDIANSettings) HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error) {
	return true, nil
}

type fakeDIANSettingsRepoForUseCase struct {
	upserted *entity.DIANSettings
	stored   *entity.DIANSettings
}

func (f *fakeDIANSettingsRepoForUseCase) Upsert(ctx context.Context, settings *entity.DIANSettings) error {
	clone := *settings
	f.upserted = &clone
	return nil
}

func (f *fakeDIANSettingsRepoForUseCase) GetByCompanyID(ctx context.Context, companyID string) (*entity.DIANSettings, error) {
	if f.stored == nil {
		return nil, nil
	}
	clone := *f.stored
	return &clone, nil
}

func (f *fakeDIANSettingsRepoForUseCase) GetByCompanyIDAndEnvironment(ctx context.Context, companyID, environment string) (*entity.DIANSettings, error) {
	if f.stored == nil || f.stored.Environment != environment {
		return nil, nil
	}
	clone := *f.stored
	return &clone, nil
}

type fakeDIANCertificateStore struct{}

func (f *fakeDIANCertificateStore) Save(companyID, environment, originalFileName string, content []byte) (string, string, error) {
	if len(content) == 0 {
		return "", "", errors.New("missing content")
	}
	return "/private/" + companyID + "/" + environment + "/cert.p12", "cert.p12", nil
}

type fakeSecretEncryptor struct{}

func (f *fakeSecretEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func TestDIANSettingsUseCase_Save_NormalizesTestingEnvironmentAndStoresByEnv(t *testing.T) {
	companyRepo := &fakeCompanyRepoForDIANSettings{company: &entity.Company{ID: "company-1"}}
	settingsRepo := &fakeDIANSettingsRepoForUseCase{}
	uc := NewDIANSettingsUseCase(
		companyRepo,
		settingsRepo,
		&fakeDIANCertificateStore{},
		&fakeSecretEncryptor{},
	)

	out, err := uc.Save("company-1", dto.UpsertDIANSettingsRequest{
		Environment:         "testing",
		CertificateFileName: "certificado_prueba.p12",
		CertificateData:     []byte("dummy-p12"),
		CertificatePassword: "123456",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, settingsRepo.upserted)
	require.NotNil(t, companyRepo.updated)

	assert.Equal(t, "test", out.Environment)
	assert.Equal(t, "test", settingsRepo.upserted.Environment)
	assert.Contains(t, settingsRepo.upserted.CertificatePath, "/test/")
	assert.Equal(t, settingsRepo.upserted.CertificatePath, companyRepo.updated.CertHab)
	assert.Empty(t, companyRepo.updated.CertProd)
}

func TestNormalizeDIANEnvironment(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		ok       bool
	}{
		{in: "test", expected: "test", ok: true},
		{in: "testing", expected: "test", ok: true},
		{in: "hab", expected: "test", ok: true},
		{in: "prod", expected: "prod", ok: true},
		{in: "production", expected: "prod", ok: true},
		{in: "produccion", expected: "prod", ok: true},
		{in: "", expected: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := normalizeDIANEnvironment(tt.in)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestDIANSettingsUseCase_Get_ReturnsStoredLatestShape(t *testing.T) {
	now := time.Now().UTC()
	companyRepo := &fakeCompanyRepoForDIANSettings{company: &entity.Company{ID: "company-1"}}
	settingsRepo := &fakeDIANSettingsRepoForUseCase{stored: &entity.DIANSettings{
		CompanyID:           "company-1",
		Environment:         "prod",
		CertificateFileName: "cert_prod.p12",
		CertificateFileSize: 2048,
		UpdatedAt:           now,
	}}
	uc := NewDIANSettingsUseCase(
		companyRepo,
		settingsRepo,
		&fakeDIANCertificateStore{},
		&fakeSecretEncryptor{},
	)

	out, err := uc.Get("company-1", "")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "prod", out.Environment)
	assert.Equal(t, "cert_prod.p12", out.CertificateFileName)
	assert.Equal(t, int64(2048), out.CertificateFileSize)
	assert.Equal(t, now, out.UpdatedAt)
}

func TestDIANSettingsUseCase_Get_ByExplicitEnvironment(t *testing.T) {
	now := time.Now().UTC()
	companyRepo := &fakeCompanyRepoForDIANSettings{company: &entity.Company{ID: "company-1"}}
	settingsRepo := &fakeDIANSettingsRepoForUseCase{stored: &entity.DIANSettings{
		CompanyID:           "company-1",
		Environment:         "test",
		CertificateFileName: "cert_test.p12",
		CertificateFileSize: 1024,
		UpdatedAt:           now,
	}}
	uc := NewDIANSettingsUseCase(
		companyRepo,
		settingsRepo,
		&fakeDIANCertificateStore{},
		&fakeSecretEncryptor{},
	)

	// alias "testing" debe normalizar a "test" y encontrar la config
	out, err := uc.Get("company-1", "testing")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "test", out.Environment)
	assert.Equal(t, "cert_test.p12", out.CertificateFileName)

	// ambiente prod no tiene config → errNotFound
	_, err = uc.Get("company-1", "production")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// ambiente invalido → errInvalidInput
	_, err = uc.Get("company-1", "unknown")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestDIANSettingsUseCase_Save_IgnoresLegacyCompanyUpdateColumnError(t *testing.T) {
	companyRepo := &fakeCompanyRepoForDIANSettings{
		company:   &entity.Company{ID: "company-1"},
		updateErr: errors.New("update company: ERROR: column cert_hab does not exist (SQLSTATE 42703)"),
	}
	settingsRepo := &fakeDIANSettingsRepoForUseCase{}
	uc := NewDIANSettingsUseCase(
		companyRepo,
		settingsRepo,
		&fakeDIANCertificateStore{},
		&fakeSecretEncryptor{},
	)

	out, err := uc.Save("company-1", dto.UpsertDIANSettingsRequest{
		Environment:         "testing",
		CertificateFileName: "certificado_prueba.p12",
		CertificateData:     []byte("dummy-p12"),
		CertificatePassword: "123456",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, settingsRepo.upserted)
	assert.Equal(t, "test", out.Environment)
}

func TestDIANSettingsUseCase_Save_FailsOnNonLegacyCompanyUpdateError(t *testing.T) {
	companyRepo := &fakeCompanyRepoForDIANSettings{
		company:   &entity.Company{ID: "company-1"},
		updateErr: errors.New("update company: deadlock detected"),
	}
	settingsRepo := &fakeDIANSettingsRepoForUseCase{}
	uc := NewDIANSettingsUseCase(
		companyRepo,
		settingsRepo,
		&fakeDIANCertificateStore{},
		&fakeSecretEncryptor{},
	)

	_, err := uc.Save("company-1", dto.UpsertDIANSettingsRequest{
		Environment:         "testing",
		CertificateFileName: "certificado_prueba.p12",
		CertificateData:     []byte("dummy-p12"),
		CertificatePassword: "123456",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deadlock")
}

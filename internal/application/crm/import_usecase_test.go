package crm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// countingCategoryRepo cuenta llamadas a GetOrCreateCategoryByName (tests de concurrencia).
type countingCategoryRepo struct {
	getOrCreateCalls atomic.Int32
}

func (c *countingCategoryRepo) Create(_ *entity.CRMCategory) error { return nil }

func (c *countingCategoryRepo) GetByID(_ string) (*entity.CRMCategory, error) { return nil, nil }

func (c *countingCategoryRepo) GetOrCreateCategoryByName(_, _ string) (string, error) {
	c.getOrCreateCalls.Add(1)
	return "00000000-0000-0000-0000-0000000000aa", nil
}

func (c *countingCategoryRepo) ListByCompany(_ string, _, _ int) ([]*entity.CRMCategory, int64, error) {
	return nil, 0, nil
}

func (c *countingCategoryRepo) Update(_ *entity.CRMCategory) error { return nil }

func (c *countingCategoryRepo) Delete(_ string) error { return nil }

func (c *countingCategoryRepo) SetActive(_, _ string, _ bool, _ time.Time) error { return nil }

var _ repository.CRMCategoryRepository = (*countingCategoryRepo)(nil)

// importConcurrentCustomerRepo devuelve siempre "sin cliente" para forzar creación con email único por test.
type importConcurrentCustomerRepo struct{}

func (importConcurrentCustomerRepo) GetByCompanyAndEmail(_, _ string) (*entity.Customer, error) {
	return nil, nil
}

func (importConcurrentCustomerRepo) Create(_ *entity.Customer) error { return nil }

func (importConcurrentCustomerRepo) GetByID(_ string) (*entity.Customer, error) { return nil, nil }

func (importConcurrentCustomerRepo) GetByCompanyAndTaxID(_, _ string) (*entity.Customer, error) {
	return nil, nil
}

func (importConcurrentCustomerRepo) ListByCompany(_ string, _ string, _, _ int) ([]*entity.Customer, int64, error) {
	return nil, 0, nil
}

func (importConcurrentCustomerRepo) Update(_ *entity.Customer) error { return nil }

func (importConcurrentCustomerRepo) Delete(_ string) error { return nil }

func (importConcurrentCustomerRepo) SetActive(_, _ string, _ bool) error { return nil }

var _ repository.CustomerRepository = (*importConcurrentCustomerRepo)(nil)

func TestNormalizeImportEmail(t *testing.T) {
	assert.Equal(t, "cliente@acme.com", normalizeImportEmail("  Cliente @ Acme.COM  "))
	assert.Equal(t, "", normalizeImportEmail("   "))
}

func TestValidateImportRows_FlagsDuplicatesMissingEmailAndLastPurchaseWarnings(t *testing.T) {
	uc := &ImportUseCase{}
	rows := [][]string{
		{"nombre", "email", "telefono", "documento", "fecha_nacimiento", "categoria", "ultima compra"},
		{"Ana", "Ana@Example.com", "3001234567", "ID-1", "", "VIP", ""},
		{"Ana 2", " ana@example.com ", "3001234568", "ID-2", "", "VIP", "01/2025"},
		{"Sin Email", "", "3001234569", "ID-3", "", "VIP", "02/2025"},
		{"Con Espacios", "  cliente @ demo.com ", "3001234570", "ID-4", "", "VIP", ""},
	}

	validRows, preview := uc.validateImportRows(rows)
	require.NotNil(t, preview)

	assert.Len(t, preview.Rows, 4)
	assert.Equal(t, 4, preview.Summary.TotalRows)
	assert.Equal(t, 1, preview.Summary.ValidRows)
	assert.Equal(t, 3, preview.Summary.InvalidRows)
	assert.Equal(t, 2, preview.Summary.DuplicateRows)
	assert.Equal(t, 1, preview.Summary.MissingEmailRows)
	assert.Equal(t, 4, preview.Summary.WarningRows)
	assert.Len(t, validRows, 1)

	first := preview.Rows[0]
	assert.Equal(t, 2, first.Row)
	assert.Equal(t, "ana@example.com", first.NormalizedEmail)
	assert.False(t, first.Valid)
	assert.Contains(t, first.Errors, "email duplicado dentro del archivo")
	assert.Contains(t, first.Warnings, "última compra vacía")

	third := preview.Rows[2]
	assert.Equal(t, 4, third.Row)
	assert.False(t, third.Valid)
	assert.Contains(t, third.Errors, "email es obligatorio")

	fourth := preview.Rows[3]
	assert.Equal(t, "cliente@demo.com", fourth.NormalizedEmail)
	assert.True(t, fourth.Valid)
}

func TestValidateImportRows_ParsesBirthDateAndKeepsItOptional(t *testing.T) {
	uc := &ImportUseCase{}
	rows := [][]string{
		{"nombre", "email", "telefono", "documento", "fecha_nacimiento", "categoria", "ultima compra"},
		{"Ana", "ana@example.com", "3001234567", "ID-1", "31-12-1990", "VIP", "01/2025"},
		{"Sin Fecha", "sinfecha@example.com", "3001234568", "ID-2", "", "VIP", "02/2025"},
	}

	validRows, preview := uc.validateImportRows(rows)
	require.NotNil(t, preview)

	assert.Len(t, validRows, 2)
	assert.Equal(t, 2, preview.Summary.TotalRows)
	assert.Equal(t, 2, preview.Summary.ValidRows)
	assert.Equal(t, 0, preview.Summary.InvalidRows)

	first := preview.Rows[0]
	assert.True(t, first.Valid)
	assert.Equal(t, "1990-12-31", first.FechaNacimiento)
	assert.Empty(t, first.Errors)
	assert.Empty(t, first.Warnings)

	second := preview.Rows[1]
	assert.True(t, second.Valid)
	assert.Empty(t, second.FechaNacimiento)
	assert.Contains(t, second.Warnings, "fecha_nacimiento vacía")
}

func TestValidateImportRows_RejectsInvalidBirthDateFormatAndImpossibleDate(t *testing.T) {
	uc := &ImportUseCase{}
	rows := [][]string{
		{"nombre", "email", "telefono", "documento", "fecha_nacimiento", "categoria"},
		{"Formato Malo", "bad-format@example.com", "3001234567", "ID-1", "1990-12-31", "VIP"},
		{"Fecha Inexistente", "bad-date@example.com", "3001234568", "ID-2", "31-02-2024", "VIP"},
	}

	_, preview := uc.validateImportRows(rows)
	require.NotNil(t, preview)

	assert.Len(t, preview.Rows, 2)
	assert.False(t, preview.Rows[0].Valid)
	assert.Contains(t, preview.Rows[0].Errors, "fecha_nacimiento: formato inválido, use DD-MM-YYYY")
	assert.False(t, preview.Rows[1].Valid)
	assert.Contains(t, preview.Rows[1].Errors, "fecha_nacimiento: fecha inválida o inexistente")
	assert.Equal(t, 2, preview.Summary.InvalidRows)
	assert.Equal(t, 0, preview.Summary.ValidRows)
}

func TestParseImportBirthDate_ReturnsNormalizedISODate(t *testing.T) {
	parsed, iso, err := parseImportBirthDate("07-03-2001")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "2001-03-07", iso)
	assert.Equal(t, time.Date(2001, 3, 7, 0, 0, 0, 0, time.UTC), parsed.UTC())

	parsed, iso, err = parseImportBirthDate("")
	require.NoError(t, err)
	assert.Nil(t, parsed)
	assert.Empty(t, iso)
}

func TestParseImportPhone_NormalizesLocalAndInternationalNumbers(t *testing.T) {
	phone, err := parseImportPhone("300 123 4567")
	require.NoError(t, err)
	assert.Equal(t, "+573001234567", phone)

	phone, err = parseImportPhone("+57 300-123-4567")
	require.NoError(t, err)
	assert.Equal(t, "+573001234567", phone)

	phone, err = parseImportPhone("")
	require.NoError(t, err)
	assert.Empty(t, phone)
}

func TestParseImportPhone_RejectsInvalidNumbers(t *testing.T) {
	phone, err := parseImportPhone("ABC-123")
	require.Error(t, err)
	assert.Empty(t, phone)
	assert.Contains(t, err.Error(), "telefono: formato inválido")
}

func TestParseRowReadsCategoryAndNormalizesPhone(t *testing.T) {
	uc := &ImportUseCase{}
	headers := []string{"Nombre", "Email", "Telefono", "Documento", "Fecha_Nacimiento", "Categoria"}
	headersMap := uc.mapHeaders(headers)
	profile, errs := uc.parseRow(headersMap, []string{"Ana", "ana@example.com", "300 123 4567", "900123123", "01-01-1990", "VIP"})

	require.Empty(t, errs)
	assert.Equal(t, "+573001234567", profile.Phone)
	assert.Equal(t, "900123123", profile.TaxID)
	assert.Equal(t, "VIP", profile.CategoryName)
}

func TestImportJobProgress_TracksInsertedUpdatedSkippedAndFailed(t *testing.T) {
	jobID := "job-1"
	defer importJobs.Delete(jobID)

	preview := &dto.CRMImportPreviewResponse{
		Summary: dto.ImportPreviewSummary{
			TotalRows:        3,
			ValidRows:        1,
			InvalidRows:      2,
			DuplicateRows:    1,
			MissingEmailRows: 1,
			WarningRows:      0,
		},
		Rows: []dto.ImportPreviewRow{
			{Row: 2, Email: "ok@example.com", NormalizedEmail: "ok@example.com", Valid: true},
			{Row: 3, Email: "", Valid: false, Errors: []string{"email es obligatorio"}},
			{Row: 4, Email: "dup@example.com", NormalizedEmail: "dup@example.com", Valid: false, Errors: []string{"email duplicado dentro del archivo"}},
		},
	}

	importJobs.Store(jobID, newImportJobState(jobID, preview))
	uc := &ImportUseCase{}

	uc.markJobRowAction(jobID, 2, "inserted", nil)
	uc.incrementJobCounter(jobID, func(progress *JobProgress) { progress.InsertedRows++ })
	uc.updateJobProcessed(jobID, 1, "processing")
	uc.markJobRowFailed(jobID, 2, "create error")
	uc.incrementJobCounter(jobID, func(progress *JobProgress) { progress.FailedRows++ })

	progress, ok := uc.GetJobProgress(jobID)
	require.True(t, ok)
	assert.Equal(t, 3, progress.TotalRows)
	assert.Equal(t, 1, progress.ValidRows)
	assert.Equal(t, 2, progress.InvalidRows)
	assert.Equal(t, 1, progress.DuplicateRows)
	assert.Equal(t, 2, progress.SkippedRows)
	assert.Equal(t, 1, progress.ProcessedRows)
	assert.Equal(t, 1, progress.InsertedRows)
	assert.Equal(t, 1, progress.FailedRows)
	assert.Len(t, progress.Rows, 3)
	assert.Equal(t, "failed", progress.Rows[0].Action)
	assert.Contains(t, progress.Rows[0].Errors, "create error")
	assert.Equal(t, "skipped", progress.Rows[1].Action)
	assert.Equal(t, "skipped", progress.Rows[2].Action)
}

func TestUpsertProfile_CategoryCacheConcurrent_CallsGetOrCreateOnce(t *testing.T) {
	catRepo := &countingCategoryRepo{}
	uc := NewImportUseCase(
		nil,
		&loyaltyProfileRepoFake{},
		importConcurrentCustomerRepo{},
		catRepo,
		nil,
		nil,
	)

	cache := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	cats := []string{"vip", "VIP", " Vip "}
	ctx := context.Background()

	const n = 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			prof := dto.ImportCRMProfileRequest{
				Name:         fmt.Sprintf("Usuario %d", i),
				Email:        fmt.Sprintf("conc-%d@example.com", i),
				Phone:        "+573001234567",
				TaxID:        fmt.Sprintf("DOC%06d", i),
				CategoryName: cats[i%3],
			}
			_, err := uc.upsertProfile(ctx, "company-1", "user-1", prof, cache, &mu)
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int32(1), catRepo.getOrCreateCalls.Load(), "GetOrCreateCategoryByName debe ejecutarse una sola vez con caché+mutex")
}

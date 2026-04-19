package crm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

func TestNormalizeImportEmail(t *testing.T) {
	assert.Equal(t, "cliente@acme.com", normalizeImportEmail("  Cliente @ Acme.COM  "))
	assert.Equal(t, "", normalizeImportEmail("   "))
}

func TestValidateImportRows_FlagsDuplicatesMissingEmailAndLastPurchaseWarnings(t *testing.T) {
	uc := &ImportUseCase{}
	rows := [][]string{
		{"Nombre", "Email", "IDCliente", "Última Compra"},
		{"Ana", "Ana@Example.com", "ID-1", ""},
		{"Ana 2", " ana@example.com ", "ID-2", "01/2025"},
		{"Sin Email", "", "ID-3", "02/2025"},
		{"Con Espacios", "  cliente @ demo.com ", "ID-4", ""},
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
		{"Nombre", "Email", "IDCliente", "Última Compra", "Fecha Nacimiento"},
		{"Ana", "ana@example.com", "ID-1", "01/2025", "31-12-1990"},
		{"Sin Fecha", "sinfecha@example.com", "ID-2", "02/2025", ""},
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
		{"Nombre", "Email", "Fecha Nacimiento"},
		{"Formato Malo", "bad-format@example.com", "1990-12-31"},
		{"Fecha Inexistente", "bad-date@example.com", "31-02-2024"},
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

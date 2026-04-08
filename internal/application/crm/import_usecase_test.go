package crm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, 2, preview.Summary.ValidRows)
	assert.Equal(t, 2, preview.Summary.InvalidRows)
	assert.Equal(t, 2, preview.Summary.DuplicateRows)
	assert.Equal(t, 1, preview.Summary.MissingEmailRows)
	assert.Equal(t, 2, preview.Summary.WarningRows)
	assert.Len(t, validRows, 2)

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

package http

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/stretchr/testify/mock"
)

type mockAIAnalystService struct {
	mock.Mock
}

func (m *mockAIAnalystService) Ask(ctx context.Context, companyID, question string) (*dto.CRMTextToSQLResponse, error) {
	args := m.Called(ctx, companyID, question)
	if out, ok := args.Get(0).(*dto.CRMTextToSQLResponse); ok {
		return out, args.Error(1)
	}
	return nil, args.Error(1)
}

type mockBulkImporterService struct {
	mock.Mock
}

func (m *mockBulkImporterService) ImportFromCSV(ctx context.Context, companyID, filePath string, tableName string) (int, error) {
	args := m.Called(ctx, companyID, filePath, tableName)
	return args.Int(0), args.Error(1)
}

func (m *mockBulkImporterService) ImportFromExcel(ctx context.Context, companyID, filePath string, sheetName string, tableName string) (int, error) {
	args := m.Called(ctx, companyID, filePath, sheetName, tableName)
	return args.Int(0), args.Error(1)
}

func (m *mockBulkImporterService) ValidateImportData(records []map[string]interface{}, tableName string) []error {
	args := m.Called(records, tableName)
	if errs, ok := args.Get(0).([]error); ok {
		return errs
	}
	return nil
}

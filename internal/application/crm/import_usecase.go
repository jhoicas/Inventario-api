package crm

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// ImportUseCase gestiona la importación masiva de perfiles CRM.
type ImportUseCase struct {
	profileRepo  repository.CRMProfileRepository
	customerRepo repository.CustomerRepository
}

// NewImportUseCase construye el caso de uso de importación.
func NewImportUseCase(
	profileRepo repository.CRMProfileRepository,
	customerRepo repository.CustomerRepository,
) *ImportUseCase {
	return &ImportUseCase{
		profileRepo:  profileRepo,
		customerRepo: customerRepo,
	}
}

// ImportProfilesFromFile lee un archivo Excel/CSV y hace upsert de perfiles CRM.
// Soporta archivos .xlsx (Excel) y .csv.
func (uc *ImportUseCase) ImportProfilesFromFile(
	ctx context.Context,
	companyID string,
	file *multipart.FileHeader,
) (*dto.CRMImportResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	if file == nil {
		return nil, domain.ErrInvalidInput
	}

	// Abre el archivo subido
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("abrir archivo: %w", err)
	}
	defer src.Close()

	var rows [][]string
	filename := strings.ToLower(file.Filename)

	if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls") {
		rows, err = uc.readExcel(src)
	} else if strings.HasSuffix(filename, ".csv") {
		rows, err = uc.readCSV(src)
	} else {
		return nil, domain.ErrInvalidInput
	}

	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	return uc.processRows(ctx, companyID, rows)
}

// readExcel parsea un archivo Excel.
func (uc *ImportUseCase) readExcel(r io.Reader) ([][]string, error) {
	// Lee el archivo completo en memoria
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("leer contenido: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("abrir Excel: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no hay hojas disponibles")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("obtener filas: %w", err)
	}
	return rows, nil
}

// readCSV parsea un archivo CSV.
func (uc *ImportUseCase) readCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // Permite registros de diferentes longitudes

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("leer CSV: %w", err)
		}
		rows = append(rows, record)
	}
	return rows, nil
}

// processRows procesa las filas del archivo y ejecuta upserts.
func (uc *ImportUseCase) processRows(ctx context.Context, companyID string, rows [][]string) (*dto.CRMImportResponse, error) {
	result := &dto.CRMImportResponse{
		TotalRows:    len(rows),
		CreatedCount: 0,
		UpdatedCount: 0,
		SkippedCount: 0,
		Errors:       make([]dto.ImportError, 0),
		ProcessedAt:  time.Now(),
		CompanyID:    companyID,
	}

	if len(rows) == 0 {
		return result, nil
	}

	// Primera fila como encabezados
	headers := rows[0]
	headerMap := uc.mapHeaders(headers)

	// Procesa datos desde la segunda fila
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Si la fila está vacía, salta
		if len(row) == 0 || uc.isEmptyRow(row) {
			result.SkippedCount++
			continue
		}

		profile := uc.parseRow(headerMap, row)

		// Validación básica
		if strings.TrimSpace(profile.Email) == "" {
			result.SkippedCount++
			result.Errors = append(result.Errors, dto.ImportError{
				Row:     i + 1,
				Message: "email requerido",
			})
			continue
		}

		// Normaliza email
		profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))

		// Intenta upsert
		created, err := uc.upsertProfile(ctx, companyID, profile)
		if err != nil {
			result.Errors = append(result.Errors, dto.ImportError{
				Row:     i + 1,
				Email:   profile.Email,
				Message: fmt.Sprintf("error: %v", err),
			})
			result.SkippedCount++
			continue
		}

		if created {
			result.CreatedCount++
		} else {
			result.UpdatedCount++
		}
	}

	return result, nil
}

// mapHeaders crea un mapa de índices de columnas basado en los encabezados.
func (uc *ImportUseCase) mapHeaders(headers []string) map[string]int {
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return headerMap
}

// parseRow extrae datos de una fila según los encabezados.
func (uc *ImportUseCase) parseRow(headerMap map[string]int, row []string) dto.ImportCRMProfileRequest {
	profile := dto.ImportCRMProfileRequest{}

	// Busca y asigna campos por nombre de columna (flexible)
	if idx, ok := headerMap["nombre"]; ok && idx < len(row) {
		profile.Nombre = strings.TrimSpace(row[idx])
	}
	if idx, ok := headerMap["email"]; ok && idx < len(row) {
		profile.Email = strings.TrimSpace(row[idx])
	}
	if idx, ok := headerMap["segmento"]; ok && idx < len(row) {
		profile.Segmento = strings.TrimSpace(row[idx])
	}
	if idx, ok := headerMap["total_comprado"]; ok && idx < len(row) {
		if val := strings.TrimSpace(row[idx]); val != "" {
			// Convierte a float si es posible
			var f float64
			fmt.Sscanf(val, "%f", &f)
			profile.TotalComprado = f
		}
	}
	if idx, ok := headerMap["categoria_principal"]; ok && idx < len(row) {
		profile.CategoriaPrincipal = strings.TrimSpace(row[idx])
	}
	if idx, ok := headerMap["productos_comprados"]; ok && idx < len(row) {
		profile.ProductosComprados = strings.TrimSpace(row[idx])
	}
	if idx, ok := headerMap["accion_remarketing"]; ok && idx < len(row) {
		profile.AccionRemarketingType = strings.TrimSpace(row[idx])
	}

	return profile
}

// isEmptyRow verifica si una fila está completamente vacía.
func (uc *ImportUseCase) isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// upsertProfile hace upsert de un perfil CRM.
// Retorna true si fue creado, false si fue actualizado.
func (uc *ImportUseCase) upsertProfile(
	ctx context.Context,
	companyID string,
	profile dto.ImportCRMProfileRequest,
) (bool, error) {
	// Busca cliente por email; si no existe, lo crea automáticamente.
	customer, err := uc.customerRepo.GetByCompanyAndEmail(companyID, profile.Email)
	if err != nil {
		return false, fmt.Errorf("buscar cliente por email: %w", err)
	}
	if customer == nil {
		now := time.Now()
		name := strings.TrimSpace(profile.Nombre)
		if name == "" {
			name = strings.Split(profile.Email, "@")[0]
		}

		customer = &entity.Customer{
			ID:        uuid.NewString(),
			CompanyID: companyID,
			Name:      name,
			TaxID:     uc.buildTempTaxID(),
			Email:     profile.Email,
			Phone:     "",
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.customerRepo.Create(customer); err != nil {
			return false, fmt.Errorf("crear cliente automático: %w", err)
		}
	}

	existingProfile, err := uc.profileRepo.GetByCustomerID(customer.ID)
	if err != nil {
		return false, fmt.Errorf("buscar perfil por customer_id: %w", err)
	}

	now := time.Now()
	ltv := decimal.Zero
	if existingProfile != nil {
		ltv = existingProfile.LTV
	}
	if profile.TotalComprado > 0 {
		ltv = decimal.NewFromFloat(profile.TotalComprado)
	}

	upsertProfile := &entity.CRMCustomerProfile{
		CustomerID: customer.ID,
		CompanyID:  companyID,
		LTV:        ltv,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if existingProfile != nil {
		upsertProfile.ID = existingProfile.ID
		upsertProfile.CategoryID = existingProfile.CategoryID
		upsertProfile.CreatedAt = existingProfile.CreatedAt
	}

	if err := uc.profileRepo.Upsert(upsertProfile); err != nil {
		return false, fmt.Errorf("upsert perfil: %w", err)
	}

	_ = ctx
	return existingProfile == nil, nil
}

func (uc *ImportUseCase) buildTempTaxID() string {
	// Prefijo CF (Cliente Fiscal) + sufijo UUID corto para minimizar colisiones.
	return "CF-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:12]
}

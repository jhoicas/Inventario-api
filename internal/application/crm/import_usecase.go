package crm

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

var importBirthDatePattern = regexp.MustCompile(`^\d{2}-\d{2}-\d{4}$`)
var requiredCRMImportHeaders = []string{"nombre", "email", "telefono", "documento", "fecha_nacimiento", "categoria"}

var importJobs = sync.Map{}

// JobProgress representa el estado de un job de importación en background.
// ProcessedRows representa filas válidas que entraron al pipeline, no necesariamente filas insertadas.
type JobProgress struct {
	JobID            string                   `json:"job_id"`
	Status           string                   `json:"status"`
	TotalRows        int                      `json:"total_rows"`
	ValidRows        int                      `json:"valid_rows"`
	InvalidRows      int                      `json:"invalid_rows"`
	DuplicateRows    int                      `json:"duplicate_rows"`
	MissingEmailRows int                      `json:"missing_email_rows"`
	WarningRows      int                      `json:"warning_rows"`
	InsertedRows     int                      `json:"inserted_rows"`
	UpdatedRows      int                      `json:"updated_rows"`
	SkippedRows      int                      `json:"skipped_rows"`
	FailedRows       int                      `json:"failed_rows"`
	ProcessedRows    int                      `json:"processed_rows"`
	UpdatedAt        time.Time                `json:"updated_at"`
	Rows             []dto.ImportJobRowStatus `json:"rows,omitempty"`
}

type importJobState struct {
	mu       sync.Mutex
	progress JobProgress
}

type importValidatedRow struct {
	RowNumber int
	Profile   dto.ImportCRMProfileRequest
	Preview   dto.ImportPreviewRow
}

// ImportUseCase gestiona la importación masiva de perfiles CRM.
type ImportUseCase struct {
	db              txStarter
	profileRepo     repository.CRMProfileRepository
	customerRepo    repository.CustomerRepository
	categoryRepo    repository.CRMCategoryRepository
	taskRepo        repository.CRMTaskRepository
	opportunityRepo repository.CRMOpportunityRepository
}

type txStarter interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// NewImportUseCase construye el caso de uso de importación.
func NewImportUseCase(
	db txStarter,
	profileRepo repository.CRMProfileRepository,
	customerRepo repository.CustomerRepository,
	categoryRepo repository.CRMCategoryRepository,
	taskRepo repository.CRMTaskRepository,
	opportunityRepo repository.CRMOpportunityRepository,
) *ImportUseCase {
	return &ImportUseCase{
		db:              db,
		profileRepo:     profileRepo,
		customerRepo:    customerRepo,
		categoryRepo:    categoryRepo,
		taskRepo:        taskRepo,
		opportunityRepo: opportunityRepo,
	}
}

// ImportProfilesFromFile lee un archivo Excel/CSV y hace upsert de perfiles CRM.
// Soporta archivos .xlsx (Excel) y .csv.
func (uc *ImportUseCase) ImportProfilesFromFile(
	ctx context.Context,
	companyID string,
	userID string,
	file *multipart.FileHeader,
) (string, error) {
	rows, err := uc.readImportRows(file)
	if err != nil {
		return "", err
	}
	if err := validateCRMImportHeaders(rows); err != nil {
		return "", err
	}
	validatedRows, preview := uc.validateImportRows(rows)

	jobID := uuid.NewString()
	importJobs.Store(jobID, newImportJobState(jobID, preview))

	// Desacoplar del request actual para que el job continúe aunque el cliente cierre conexión.
	go uc.processRowsAsync(context.Background(), companyID, userID, validatedRows, preview.Summary.TotalRows, jobID)

	_ = ctx
	return jobID, nil
}

// PreviewProfilesFromFile analiza el archivo y devuelve validaciones por fila sin persistir nada.
func (uc *ImportUseCase) PreviewProfilesFromFile(file *multipart.FileHeader) (*dto.CRMImportPreviewResponse, error) {
	rows, err := uc.readImportRows(file)
	if err != nil {
		return nil, err
	}
	if err := validateCRMImportHeaders(rows); err != nil {
		return nil, err
	}
	_, preview := uc.validateImportRows(rows)
	return preview, nil
}

func (uc *ImportUseCase) readImportRows(file *multipart.FileHeader) ([][]string, error) {
	if file == nil {
		return nil, domain.ErrInvalidInput
	}
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("abrir archivo: %w", err)
	}
	defer src.Close()

	filename := strings.ToLower(file.Filename)
	if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls") {
		return uc.readExcel(src)
	}
	if strings.HasSuffix(filename, ".csv") {
		return uc.readCSV(src)
	}
	return nil, domain.ErrInvalidInput
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

// GetJobProgress retorna el estado actual de un job de importación.
func (uc *ImportUseCase) GetJobProgress(jobID string) (JobProgress, bool) {
	v, ok := importJobs.Load(jobID)
	if !ok {
		return JobProgress{}, false
	}
	state, castOK := v.(*importJobState)
	if !castOK {
		return JobProgress{}, false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return cloneJobProgress(state.progress), true
}

// processRowsAsync procesa filas en background y actualiza el progreso del job.
func (uc *ImportUseCase) processRowsAsync(ctx context.Context, companyID string, userID string, rows []importValidatedRow, totalRows int, jobID string) {
	defer func() {
		if r := recover(); r != nil {
			uc.updateJobProcessed(jobID, len(rows), "error")
		}
	}()

	if len(rows) == 0 {
		uc.updateJobProcessed(jobID, 0, "completed")
		return
	}

	var wg sync.WaitGroup
	jobs := make(chan importValidatedRow, len(rows))
	var processedCount atomic.Int32
	var emailLocks sync.Map
	categoryCache := make(map[string]string)
	var categoryCacheMu sync.Mutex

	workerCount := 20
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				func(item importValidatedRow) {
					defer func() {
						current := processedCount.Add(1)
						if current%50 == 0 {
							uc.updateJobProcessed(jobID, int(current), "processing")
						}
					}()

					if item.Preview.Valid == false {
						return
					}

					email := item.Profile.Email
					muVal, _ := emailLocks.LoadOrStore(email, &sync.Mutex{})
					mu := muVal.(*sync.Mutex)
					mu.Lock()
					defer mu.Unlock()

					inserted, err := uc.upsertProfile(ctx, companyID, userID, item.Profile, categoryCache, &categoryCacheMu)
					if err != nil {
						uc.markJobRowFailed(jobID, item.RowNumber, err.Error())
						uc.incrementJobCounter(jobID, func(p *JobProgress) { p.FailedRows++ })
						return
					}
					if inserted {
						uc.markJobRowAction(jobID, item.RowNumber, "inserted", nil)
						uc.incrementJobCounter(jobID, func(p *JobProgress) { p.InsertedRows++ })
						return
					}
					uc.markJobRowAction(jobID, item.RowNumber, "updated", nil)
					uc.incrementJobCounter(jobID, func(p *JobProgress) { p.UpdatedRows++ })
				}(item)
			}
		}()
	}

	for _, row := range rows {
		normalizedCategory := strings.ToUpper(strings.TrimSpace(row.Profile.CategoryName))
		row.Profile.CategoryName = normalizedCategory
		jobs <- row
	}
	close(jobs)

	wg.Wait()
	uc.updateJobProcessed(jobID, int(processedCount.Load()), "completed")
	uc.finalizeJob(jobID)
}

func (uc *ImportUseCase) updateProcessed(jobID string, totalRows int, processedRows int) {
	_ = totalRows
	uc.updateJobProcessed(jobID, processedRows, "processing")
}

func (uc *ImportUseCase) setJobProgress(jobID string, progress JobProgress) {
	importJobs.Store(jobID, &importJobState{progress: progress})
}

// updateJobProcessed conserva los contadores de validación al actualizar el progreso.
func (uc *ImportUseCase) updateJobProcessed(jobID string, processedRows int, status string) {
	v, ok := importJobs.Load(jobID)
	if !ok {
		importJobs.Store(jobID, &importJobState{progress: JobProgress{ProcessedRows: processedRows, Status: status, UpdatedAt: time.Now()}})
		return
	}
	state, castOK := v.(*importJobState)
	if !castOK {
		importJobs.Store(jobID, &importJobState{progress: JobProgress{ProcessedRows: processedRows, Status: status, UpdatedAt: time.Now()}})
		return
	}
	state.mu.Lock()
	state.progress.ProcessedRows = processedRows
	state.progress.Status = status
	state.progress.UpdatedAt = time.Now()
	state.mu.Unlock()
}

func (uc *ImportUseCase) incrementJobCounter(jobID string, fn func(*JobProgress)) {
	uc.withJobState(jobID, func(progress *JobProgress) {
		fn(progress)
		progress.UpdatedAt = time.Now()
	})
}

func (uc *ImportUseCase) markJobRowAction(jobID string, rowNumber int, action string, errMsg []string) {
	uc.withJobState(jobID, func(progress *JobProgress) {
		if idx, ok := findRowIndex(progress.Rows, rowNumber); ok {
			progress.Rows[idx].Action = action
			if len(errMsg) > 0 {
				progress.Rows[idx].Errors = append([]string(nil), errMsg...)
			}
			progress.Rows[idx].Valid = action != "failed" && action != "skipped"
		}
		progress.UpdatedAt = time.Now()
	})
}

func (uc *ImportUseCase) markJobRowFailed(jobID string, rowNumber int, msg string) {
	uc.withJobState(jobID, func(progress *JobProgress) {
		if idx, ok := findRowIndex(progress.Rows, rowNumber); ok {
			progress.Rows[idx].Action = "failed"
			progress.Rows[idx].Errors = append(progress.Rows[idx].Errors, msg)
			progress.Rows[idx].Valid = false
		}
		progress.UpdatedAt = time.Now()
	})
}

func (uc *ImportUseCase) finalizeJob(jobID string) {
	uc.withJobState(jobID, func(progress *JobProgress) {
		progress.SkippedRows = progress.InvalidRows
		if progress.FailedRows > 0 {
			progress.Status = "completed_with_errors"
		} else {
			progress.Status = "completed"
		}
		progress.UpdatedAt = time.Now()
	})
}

func (uc *ImportUseCase) withJobState(jobID string, fn func(*JobProgress)) {
	v, ok := importJobs.Load(jobID)
	if !ok {
		return
	}
	state, castOK := v.(*importJobState)
	if !castOK {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	fn(&state.progress)
}

func newImportJobState(jobID string, preview *dto.CRMImportPreviewResponse) *importJobState {
	progress := JobProgress{
		JobID:            jobID,
		Status:           "processing",
		TotalRows:        preview.Summary.TotalRows,
		ValidRows:        preview.Summary.ValidRows,
		InvalidRows:      preview.Summary.InvalidRows,
		DuplicateRows:    preview.Summary.DuplicateRows,
		MissingEmailRows: preview.Summary.MissingEmailRows,
		WarningRows:      preview.Summary.WarningRows,
		SkippedRows:      preview.Summary.InvalidRows,
		Rows:             make([]dto.ImportJobRowStatus, 0, len(preview.Rows)),
		UpdatedAt:        time.Now(),
	}
	for _, row := range preview.Rows {
		status := dto.ImportJobRowStatus{
			Row:             row.Row,
			Email:           row.Email,
			NormalizedEmail: row.NormalizedEmail,
			FechaNacimiento: row.FechaNacimiento,
			Valid:           row.Valid,
			Warnings:        append([]string(nil), row.Warnings...),
			Errors:          append([]string(nil), row.Errors...),
			IDCliente:       row.IDCliente,
			LastPurchase:    row.LastPurchase,
		}
		if row.Valid {
			status.Action = "pending"
		} else {
			status.Action = "skipped"
		}
		progress.Rows = append(progress.Rows, status)
	}
	return &importJobState{progress: progress}
}

func cloneJobProgress(progress JobProgress) JobProgress {
	out := progress
	if len(progress.Rows) > 0 {
		out.Rows = make([]dto.ImportJobRowStatus, len(progress.Rows))
		copy(out.Rows, progress.Rows)
	}
	return out
}

func findRowIndex(rows []dto.ImportJobRowStatus, rowNumber int) (int, bool) {
	for i, row := range rows {
		if row.Row == rowNumber {
			return i, true
		}
	}
	return 0, false
}

// mapHeaders crea un mapa de índices de columnas basado en los encabezados.
func (uc *ImportUseCase) mapHeaders(headers []string) map[string]int {
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[normalizeHeaderKey(h)] = i
	}
	return headerMap
}

// parseRow extrae datos de una fila según los encabezados.
func (uc *ImportUseCase) parseRow(headerMap map[string]int, row []string) (dto.ImportCRMProfileRequest, []string) {
	profile := dto.ImportCRMProfileRequest{}
	var rowErrors []string

	// Mapeo canónico para importación básica de clientes.
	if idx, ok := findHeaderIndex(headerMap, "nombre"); ok && idx < len(row) {
		profile.Name = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "email"); ok && idx < len(row) {
		profile.Email = normalizeImportEmail(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "telefono"); ok && idx < len(row) {
		normalizedPhone, err := parseImportPhone(row[idx])
		if err != nil {
			rowErrors = append(rowErrors, err.Error())
		} else {
			profile.Phone = normalizedPhone
		}
	}
	if idx, ok := findHeaderIndex(headerMap, "documento"); ok && idx < len(row) {
		profile.TaxID = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "fecha_nacimiento"); ok && idx < len(row) {
		raw := strings.TrimSpace(row[idx])
		if raw != "" {
			profile.FechaNacimiento = raw
			parsed, iso, err := parseImportBirthDate(raw)
			if err != nil {
				rowErrors = append(rowErrors, err.Error())
			} else {
				profile.BirthDate = parsed
				profile.FechaNacimiento = iso
			}
		}
	}
	if idx, ok := findHeaderIndex(headerMap, "segmento"); ok && idx < len(row) {
		profile.Segmento = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "categoria"); ok && idx < len(row) {
		profile.CategoryName = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "ventas totales", "ventas_totales", "ventastotales"); ok && idx < len(row) {
		if val := strings.TrimSpace(row[idx]); val != "" {
			if f, err := strconv.ParseFloat(strings.ReplaceAll(val, ",", ""), 64); err == nil {
				profile.VentasTotales = f
			}
		}
	}
	if idx, ok := findHeaderIndex(headerMap, "pedidos"); ok && idx < len(row) {
		if val := strings.TrimSpace(row[idx]); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				profile.Pedidos = n
			}
		}
	}
	if idx, ok := findHeaderIndex(headerMap, "productos"); ok && idx < len(row) {
		if val := strings.TrimSpace(row[idx]); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				profile.Productos = n
			}
		}
	}
	if idx, ok := findHeaderIndex(headerMap, "ultima compra", "ultima_compra", "ultimacompra", "última compra", "última_compra", "últimacompra"); ok && idx < len(row) {
		profile.UltimaCompra = normalizeMonthYear(strings.TrimSpace(row[idx]))
	}
	if idx, ok := findHeaderIndex(headerMap, "categoria producto", "categoria_producto", "categoriaproducto", "categoría producto", "categoría_producto", "categoríaproducto"); ok && idx < len(row) {
		profile.CategoriaProducto = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "descripcion de productos", "descripcion_de_productos", "descripcionproductos", "descripción de productos", "descripción_de_productos", "descripciónproductos"); ok && idx < len(row) {
		profile.DescripcionProductos = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(
		headerMap,
		"estrategia seguimiento",
		"estrategia de seguimiento",
		"estrategia_seguimiento",
		"estrategia_de_seguimiento",
		"estrategiaseguimiento",
		"estrategiadeseguimiento",
		"accion remarketing",
		"accion_remarketing",
		"acción remarketing",
		"acción_remarketing",
	); ok && idx < len(row) {
		profile.EstrategiaSeguimiento = strings.TrimSpace(row[idx])
	}

	return profile, rowErrors
}

func (uc *ImportUseCase) validateImportRows(rows [][]string) ([]importValidatedRow, *dto.CRMImportPreviewResponse) {
	if len(rows) == 0 {
		return nil, &dto.CRMImportPreviewResponse{Summary: dto.ImportPreviewSummary{}, Rows: []dto.ImportPreviewRow{}}
	}

	headers := rows[0]
	headerMap := uc.mapHeaders(headers)
	dataRows := rows[1:]
	items := make([]importValidatedRow, 0, len(dataRows))
	emailCounts := make(map[string]int)

	for idx, row := range dataRows {
		rowNumber := idx + 2
		if len(row) == 0 || uc.isEmptyRow(row) {
			continue
		}
		profile, rowErrors := uc.parseRow(headerMap, row)
		previewRow := dto.ImportPreviewRow{
			Row:             rowNumber,
			Email:           profile.Email,
			NormalizedEmail: profile.Email,
			FechaNacimiento: profile.FechaNacimiento,
			IDCliente:       strings.TrimSpace(profile.TaxID),
			LastPurchase:    strings.TrimSpace(profile.UltimaCompra),
			Valid:           true,
		}
		if len(rowErrors) > 0 {
			previewRow.Valid = false
			previewRow.Errors = append(previewRow.Errors, rowErrors...)
		}
		if previewRow.NormalizedEmail != "" {
			emailCounts[previewRow.NormalizedEmail]++
		}
		if previewRow.NormalizedEmail == "" {
			previewRow.Valid = false
			previewRow.Errors = append(previewRow.Errors, "email es obligatorio")
		}
		if strings.TrimSpace(profile.Name) == "" {
			previewRow.Valid = false
			previewRow.Errors = append(previewRow.Errors, "nombre es obligatorio")
		}
		if strings.TrimSpace(profile.TaxID) == "" {
			previewRow.Valid = false
			previewRow.Errors = append(previewRow.Errors, "documento es obligatorio")
		}
		if strings.TrimSpace(profile.UltimaCompra) == "" {
			previewRow.Warnings = append(previewRow.Warnings, "última compra vacía")
		}
		if strings.TrimSpace(profile.FechaNacimiento) == "" {
			previewRow.Warnings = append(previewRow.Warnings, "fecha_nacimiento vacía")
		}
		items = append(items, importValidatedRow{RowNumber: rowNumber, Profile: profile, Preview: previewRow})
	}

	for i := range items {
		if items[i].Preview.NormalizedEmail != "" && emailCounts[items[i].Preview.NormalizedEmail] > 1 {
			items[i].Preview.Valid = false
			items[i].Preview.Errors = append(items[i].Preview.Errors, "email duplicado dentro del archivo")
		}
	}

	preview := &dto.CRMImportPreviewResponse{Rows: make([]dto.ImportPreviewRow, 0, len(items))}
	for _, item := range items {
		preview.Rows = append(preview.Rows, item.Preview)
		preview.Summary.TotalRows++
		if item.Preview.Valid {
			preview.Summary.ValidRows++
		} else {
			preview.Summary.InvalidRows++
		}
		if containsString(item.Preview.Errors, "email duplicado dentro del archivo") {
			preview.Summary.DuplicateRows++
		}
		if containsString(item.Preview.Errors, "email es obligatorio") {
			preview.Summary.MissingEmailRows++
		}
		if len(item.Preview.Warnings) > 0 {
			preview.Summary.WarningRows++
		}
	}

	return filterValidImportRows(items), preview
}

func filterValidImportRows(items []importValidatedRow) []importValidatedRow {
	valid := make([]importValidatedRow, 0, len(items))
	for _, item := range items {
		if item.Preview.Valid {
			valid = append(valid, item)
		}
	}
	return valid
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func normalizeImportEmail(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}

func parseImportBirthDate(raw string) (*time.Time, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", nil
	}
	if !importBirthDatePattern.MatchString(raw) {
		return nil, "", fmt.Errorf("fecha_nacimiento: formato inválido, use DD-MM-YYYY")
	}
	parsed, err := time.Parse("02-01-2006", raw)
	if err != nil {
		return nil, "", fmt.Errorf("fecha_nacimiento: fecha inválida o inexistente")
	}
	parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	iso := parsed.Format("2006-01-02")
	return &parsed, iso, nil
}

func parseImportPhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	var builder strings.Builder
	for i, r := range raw {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
			continue
		}
		if i == 0 && r == '+' {
			continue
		}
	}

	digits := builder.String()
	if digits == "" {
		return "", fmt.Errorf("telefono: formato inválido")
	}
	if strings.HasPrefix(raw, "+57") && len(digits) == 12 && strings.HasPrefix(digits, "57") {
		return "+57" + digits[2:], nil
	}
	if len(digits) == 10 {
		return "+57" + digits, nil
	}
	if len(digits) == 12 && strings.HasPrefix(digits, "57") {
		return "+57" + digits[2:], nil
	}
	return "", fmt.Errorf("telefono: formato inválido, use 10 dígitos locales o +57XXXXXXXXXX")
}

func birthDatesEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func findHeaderIndex(headerMap map[string]int, keys ...string) (int, bool) {
	for _, key := range keys {
		if idx, ok := headerMap[normalizeHeaderKey(key)]; ok {
			return idx, true
		}
	}
	return 0, false
}

func normalizeHeaderKey(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
	return strings.ToLower(value)
}

func validateCRMImportHeaders(rows [][]string) error {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return domain.ErrInvalidInput
	}

	headerMap := make(map[string]struct{}, len(rows[0]))
	for _, header := range rows[0] {
		headerMap[normalizeHeaderKey(header)] = struct{}{}
	}

	for _, required := range requiredCRMImportHeaders {
		if _, ok := headerMap[required]; !ok {
			return fmt.Errorf("cabecera requerida faltante: %s", required)
		}
	}

	if len(headerMap) != len(requiredCRMImportHeaders) {
		return fmt.Errorf("cabeceras inválidas: se esperaban exactamente %s", strings.Join(requiredCRMImportHeaders, ", "))
	}

	return nil
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
	userID string,
	profile dto.ImportCRMProfileRequest,
	categoryCache map[string]string,
	categoryCacheMu *sync.Mutex,
) (bool, error) {
	// Busca cliente únicamente por email.
	var (
		customer *entity.Customer
		err      error
		created  bool
	)
	customer, err = uc.customerRepo.GetByCompanyAndEmail(companyID, profile.Email)
	if err != nil {
		return false, fmt.Errorf("buscar cliente por email: %w", err)
	}
	if customer == nil {
		created = true
		now := time.Now()
		name := strings.TrimSpace(profile.Name)
		taxID := strings.TrimSpace(profile.TaxID)
		if name == "" || taxID == "" {
			return false, fmt.Errorf("nombre y documento son obligatorios para crear cliente")
		}

		customer = &entity.Customer{
			ID:        uuid.NewString(),
			CompanyID: companyID,
			Name:      name,
			TaxID:     taxID,
			Email:     profile.Email,
			Phone:     profile.Phone,
			BirthDate: profile.BirthDate,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.customerRepo.Create(customer); err != nil {
			return false, fmt.Errorf("crear cliente automático: %w", err)
		}
	} else {
		incomingPhone := profile.Phone
		incomingBirthDate := profile.BirthDate
		incomingName := strings.TrimSpace(profile.Name)
		incomingTaxID := strings.TrimSpace(profile.TaxID)
		needsUpdate := false
		if incomingName != "" && strings.TrimSpace(customer.Name) != incomingName {
			customer.Name = incomingName
			needsUpdate = true
		}
		if incomingTaxID != "" && strings.TrimSpace(customer.TaxID) != incomingTaxID {
			customer.TaxID = incomingTaxID
			needsUpdate = true
		}
		if incomingPhone != "" && strings.TrimSpace(customer.Phone) != incomingPhone {
			customer.Phone = incomingPhone
			needsUpdate = true
		}
		if incomingBirthDate != nil && (customer.BirthDate == nil || !customer.BirthDate.Equal(*incomingBirthDate)) {
			customer.BirthDate = incomingBirthDate
			needsUpdate = true
		}
		if needsUpdate {
			customer.UpdatedAt = time.Now()
			if err := uc.customerRepo.Update(customer); err != nil {
				return false, fmt.Errorf("actualizar cliente: %w", err)
			}
		}
	}

	if uc.categoryRepo == nil {
		return false, fmt.Errorf("repositorio de categorías no configurado")
	}
	normalizedCategory := strings.ToUpper(strings.TrimSpace(profile.CategoryName))
	profile.CategoryName = normalizedCategory
	if normalizedCategory == "" {
		log.Printf("crm import: categoría vacía company_id=%s customer_id=%s", companyID, customer.ID)
		return false, fmt.Errorf("categoría vacía para customer_id=%s", customer.ID)
	}

	var categoryID string
	categoryCacheMu.Lock()
	if cachedID, exists := categoryCache[normalizedCategory]; exists {
		categoryID = cachedID
		categoryCacheMu.Unlock()
	} else {
		var resolveErr error
		categoryID, resolveErr = uc.categoryRepo.GetOrCreateCategoryByName(companyID, normalizedCategory)
		if resolveErr != nil {
			categoryCacheMu.Unlock()
			log.Printf("crm import: error resolviendo categoría company_id=%s customer_id=%s category=%q: %v", companyID, customer.ID, normalizedCategory, resolveErr)
			return false, fmt.Errorf("resolver categoría: %w", resolveErr)
		}
		categoryCache[normalizedCategory] = categoryID
		categoryCacheMu.Unlock()
	}
	if strings.TrimSpace(categoryID) == "" {
		log.Printf("crm import: category_id vacío company_id=%s customer_id=%s category=%q", companyID, customer.ID, normalizedCategory)
		return false, fmt.Errorf("category_id vacío para customer_id=%s", customer.ID)
	}
	if err := uc.profileRepo.UpsertCustomerProfile(ctx, customer.ID, companyID, categoryID); err != nil {
		log.Printf("crm import: error upsert crm_customer_profiles company_id=%s customer_id=%s category_id=%s: %v", companyID, customer.ID, categoryID, err)
		return false, fmt.Errorf("upsert customer profile: %w", err)
	}

	if err := uc.createAutomationArtifacts(ctx, companyID, userID, customer.ID, profile); err != nil {
		return false, err
	}

	return created, nil
}

func (uc *ImportUseCase) createAutomationArtifacts(ctx context.Context, companyID, userID, customerID string, profile dto.ImportCRMProfileRequest) error {
	segment := strings.ToUpper(strings.TrimSpace(profile.Segmento))
	if segment != "VIP" && segment != "PREMIUM" {
		return nil
	}

	if uc.taskRepo != nil {
		desc := strings.TrimSpace(profile.EstrategiaSeguimiento)
		if desc == "" {
			desc = "Seguimiento recomendado tras importación masiva."
		}
		task := &entity.CRMTask{
			ID:          uuid.NewString(),
			CompanyID:   companyID,
			CustomerID:  customerID,
			Title:       "Seguimiento post-importación",
			Description: desc,
			Status:      entity.TaskStatusPending,
			CreatedBy:   userID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := uc.taskRepo.Create(task); err != nil {
			return fmt.Errorf("crear tarea automática: %w", err)
		}
	}

	if uc.opportunityRepo != nil {
		titleCategory := strings.TrimSpace(profile.CategoriaProducto)
		if titleCategory == "" {
			titleCategory = "categoría principal"
		}
		desc := profile.DescripcionProductos
		if strings.TrimSpace(desc) == "" {
			desc = fmt.Sprintf("Oportunidad de Up-Sell basada en %s.", titleCategory)
		}
		opp := &entity.Opportunity{
			ID:          uuid.NewString(),
			CompanyID:   companyID,
			CustomerID:  customerID,
			Title:       fmt.Sprintf("Up-Sell: %s", titleCategory),
			Amount:      decimal.NewFromFloat(profile.VentasTotales),
			Probability: 25,
			Stage:       entity.OpportunityStageProspecto,
			CreatedBy:   userID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_ = desc
		if err := uc.opportunityRepo.Create(ctx, opp); err != nil {
			return fmt.Errorf("crear oportunidad automática: %w", err)
		}
	}

	return nil
}

func normalizeMonthYear(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		month := strings.TrimSpace(parts[0])
		year := strings.TrimSpace(parts[1])
		if len(month) == 1 {
			month = "0" + month
		}
		if len(month) == 2 && len(year) == 4 {
			return month + "/" + year
		}
	}
	return value
}

// ImportSalesFromFile procesa un archivo Excel/CSV y persiste ventas con snapshot JSONB por orden.
func (uc *ImportUseCase) ImportSalesFromFile(ctx context.Context, companyID, userID string, file *multipart.FileHeader) (*dto.ImportSalesResponse, error) {
	if uc == nil || uc.db == nil || strings.TrimSpace(companyID) == "" || file == nil {
		return nil, domain.ErrInvalidInput
	}

	rows, err := uc.readImportRows(file)
	if err != nil {
		return nil, err
	}

	orders, err := uc.groupSalesImportRows(rows)
	if err != nil {
		return nil, err
	}

	categoryCache := make(map[string]string)
	var categoryCacheMu sync.Mutex

	resp := &dto.ImportSalesResponse{TotalOrders: len(orders)}
	for _, order := range orders {
		res, err := uc.importSingleSalesOrder(ctx, companyID, userID, order, categoryCache, &categoryCacheMu)
		if err != nil {
			resp.FailedOrders++
			resp.Errors = append(resp.Errors, dto.ImportSalesError{OrderNumber: order.OrderNumber, Message: err.Error()})
			continue
		}
		resp.SuccessfulOrders++
		resp.TotalItems += res.TotalItems
		resp.CreatedCustomers += boolToInt(res.CreatedCustomer)
		resp.CreatedCategories += res.CreatedCategories
		resp.CreatedProducts += res.CreatedProducts
	}

	return resp, nil
}

type salesImportItem struct {
	ProductCode   string
	ProductName   string
	CategoryName  string
	Quantity      int
	UnitPrice     float64
	LineTotal     float64
	RawRowNumber  int
	CustomerEmail string
	CustomerPhone string
	CustomerName  string
	OrderNumber   string
	SaleDate      time.Time
	ProductID     string
	CategoryID    string
	CustomerID    string
	ItemsSnapshot map[string]any
}

type salesImportOrder struct {
	OrderNumber   string
	SaleDate      time.Time
	CustomerEmail string
	CustomerPhone string
	CustomerName  string
	Items         []salesImportItem
	Errors        []string
	TotalAmount   float64
	CreatedAt     time.Time
}

type salesImportTxResult struct {
	CreatedCustomer   bool
	CreatedCategories int
	CreatedProducts   int
	TotalItems        int
}

func (uc *ImportUseCase) groupSalesImportRows(rows [][]string) ([]salesImportOrder, error) {
	if len(rows) == 0 {
		return nil, domain.ErrInvalidInput
	}

	headers := rows[0]
	headerMap := uc.mapHeaders(headers)
	dataRows := rows[1:]
	ordersByNumber := make(map[string]*salesImportOrder)
	orderedKeys := make([]string, 0)

	for idx, row := range dataRows {
		rowNumber := idx + 2
		if len(row) == 0 || uc.isEmptyRow(row) {
			continue
		}

		orderNumber := trimCell(salesCell(headerMap, row, "numero_orden", "numero orden", "numeroorden", "orden", "order_number", "ordernumber"))
		saleDateRaw := trimCell(salesCell(headerMap, row, "fecha_venta", "fecha venta", "fechaventa", "sale_date", "saledate"))
		email := normalizeImportEmail(salesCell(headerMap, row, "email_cliente", "email cliente", "email", "customer_email", "customeremail"))
		phone := normalizeImportPhone(salesCell(headerMap, row, "telefono", "teléfono", "phone", "customer_phone"))
		customerName := trimCell(salesCell(headerMap, row, "nombre_cliente", "nombre cliente", "nombrecliente", "customer_name"))
		productCode := trimCell(salesCell(headerMap, row, "codigo_producto", "código_producto", "codigo producto", "product_code", "productcode"))
		productName := trimCell(salesCell(headerMap, row, "nombre_producto", "nombre producto", "nombreproducto", "product_name", "productname"))
		categoryName := trimCell(salesCell(headerMap, row, "categoria_producto", "categoría_producto", "categoria producto", "category", "product_category"))
		qtyRaw := trimCell(salesCell(headerMap, row, "cantidad", "quantity", "qty"))
		unitPriceRaw := trimCell(salesCell(headerMap, row, "precio_unitario", "precio unitario", "unit_price", "price"))

		if orderNumber == "" || saleDateRaw == "" || email == "" || productCode == "" || productName == "" || qtyRaw == "" || unitPriceRaw == "" {
			continue
		}

		saleDate, err := parseSalesDate(saleDateRaw)
		if err != nil {
			continue
		}
		qty, err := strconv.Atoi(qtyRaw)
		if err != nil || qty <= 0 {
			continue
		}
		unitPrice, err := strconv.ParseFloat(strings.ReplaceAll(unitPriceRaw, ",", ""), 64)
		if err != nil {
			continue
		}
		lineTotal := float64(qty) * unitPrice

		order, ok := ordersByNumber[orderNumber]
		if !ok {
			order = &salesImportOrder{OrderNumber: orderNumber, SaleDate: saleDate, CustomerEmail: email, CustomerPhone: phone, CustomerName: customerName, CreatedAt: time.Now()}
			ordersByNumber[orderNumber] = order
			orderedKeys = append(orderedKeys, orderNumber)
		}
		if order.CustomerEmail == "" {
			order.CustomerEmail = email
		}
		if order.CustomerPhone == "" {
			order.CustomerPhone = phone
		}
		if order.CustomerName == "" {
			order.CustomerName = customerName
		}
		if order.SaleDate.IsZero() {
			order.SaleDate = saleDate
		}
		order.Items = append(order.Items, salesImportItem{
			ProductCode:   productCode,
			ProductName:   productName,
			CategoryName:  categoryName,
			Quantity:      qty,
			UnitPrice:     unitPrice,
			LineTotal:     lineTotal,
			RawRowNumber:  rowNumber,
			CustomerEmail: email,
			CustomerPhone: phone,
			CustomerName:  customerName,
			OrderNumber:   orderNumber,
			SaleDate:      saleDate,
		})
		order.TotalAmount += lineTotal
	}

	orders := make([]salesImportOrder, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		if order := ordersByNumber[key]; order != nil && len(order.Items) > 0 {
			orders = append(orders, *order)
		}
	}

	if len(orders) == 0 {
		return nil, domain.ErrInvalidInput
	}
	return orders, nil
}

func (uc *ImportUseCase) importSingleSalesOrder(
	ctx context.Context,
	companyID, userID string,
	order salesImportOrder,
	categoryCache map[string]string,
	categoryMu *sync.Mutex,
) (*salesImportTxResult, error) {
	tx, err := uc.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin sales import tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := &salesImportTxResult{}

	customerID, createdCustomer, err := upsertCustomerFromImportTx(ctx, tx, companyID, order.CustomerEmail, order.CustomerName, order.CustomerPhone)
	if err != nil {
		return nil, err
	}
	result.CreatedCustomer = createdCustomer

	snapshotItems := make([]salesSnapshotItem, 0, len(order.Items))
	for _, item := range order.Items {
		normCat := strings.ToUpper(strings.TrimSpace(item.CategoryName))
		var categoryID string
		var createdCategory bool
		if normCat == "" {
			categoryID = ""
			createdCategory = false
		} else {
			categoryMu.Lock()
			if cachedID, ok := categoryCache[normCat]; ok {
				categoryID = cachedID
				categoryMu.Unlock()
				createdCategory = false
			} else {
				var err error
				categoryID, createdCategory, err = upsertCategoryHubTx(ctx, tx, companyID, normCat)
				if err != nil {
					categoryMu.Unlock()
					return nil, err
				}
				categoryCache[normCat] = categoryID
				categoryMu.Unlock()
			}
		}
		if createdCategory {
			result.CreatedCategories++
		}

		productID, createdProduct, err := upsertProductHubTx(ctx, tx, companyID, item.ProductCode, item.ProductName, normCat)
		if err != nil {
			return nil, err
		}
		if createdProduct {
			result.CreatedProducts++
		}

		snapshotItems = append(snapshotItems, salesSnapshotItem{
			OrderNumber:  order.OrderNumber,
			ProductCode:  item.ProductCode,
			ProductName:  item.ProductName,
			CategoryName: normCat,
			CategoryID:   categoryID,
			ProductID:    productID,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			LineTotal:    item.LineTotal,
		})
		result.TotalItems++
	}

	snapshotJSON, err := json.Marshal(snapshotItems)
	if err != nil {
		return nil, fmt.Errorf("marshal sale snapshot: %w", err)
	}

	saleID, err := insertSaleHubTx(ctx, tx, saleHubInput{
		CompanyID:     companyID,
		CustomerID:    customerID,
		CustomerEmail: order.CustomerEmail,
		CustomerName:  emptyStringPtr(order.CustomerName),
		SaleDate:      order.SaleDate,
		TotalAmount:   order.TotalAmount,
		ItemsSnapshot: snapshotJSON,
	})
	if err != nil {
		return nil, err
	}

	if err := recalculateCustomerProfileTx(ctx, tx, companyID, customerID, order.SaleDate, order.TotalAmount); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sales import tx: %w", err)
	}

	_ = saleID
	_ = userID
	return result, nil
}

type salesSnapshotItem struct {
	OrderNumber  string  `json:"order_number"`
	ProductCode  string  `json:"product_code"`
	ProductName  string  `json:"product_name"`
	CategoryName string  `json:"category_name"`
	CategoryID   string  `json:"category_id,omitempty"`
	ProductID    string  `json:"product_id,omitempty"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	LineTotal    float64 `json:"line_total"`
}

type saleHubInput struct {
	CompanyID     string
	CustomerID    string
	CustomerEmail string
	CustomerName  *string
	SaleDate      time.Time
	TotalAmount   float64
	ItemsSnapshot []byte
}

func upsertCustomerFromImportTx(ctx context.Context, tx pgx.Tx, companyID, email, name, phone string) (string, bool, error) {
	email = normalizeImportEmail(email)
	if email == "" {
		return "", false, domain.ErrInvalidInput
	}
	cleanPhone := normalizeImportPhone(phone)
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = strings.Split(email, "@")[0]
	}
	customerID := uuid.NewString()
	var existingID string
	if err := tx.QueryRow(ctx, `
		SELECT id FROM customers WHERE company_id = $1 AND LOWER(email) = LOWER($2)`, companyID, email).Scan(&existingID); err == nil && existingID != "" {
		_, err := tx.Exec(ctx, `
			UPDATE customers
			SET name = $2, phone = $3, updated_at = now()
			WHERE id = $1`, existingID, cleanName, cleanPhone)
		if err != nil {
			return "", false, fmt.Errorf("update imported customer: %w", err)
		}
		return existingID, false, nil
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO customers (id, company_id, name, tax_id, email, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, now(), now())`, customerID, companyID, cleanName, customerID, email, cleanPhone)
	if err != nil {
		return "", false, fmt.Errorf("insert imported customer: %w", err)
	}
	return customerID, true, nil
}

func upsertCategoryHubTx(ctx context.Context, tx pgx.Tx, companyID, categoryName string) (string, bool, error) {
	categoryName = strings.ToUpper(strings.TrimSpace(categoryName))
	if categoryName == "" {
		return "", false, nil
	}
	var existingID string
	err := tx.QueryRow(ctx, `
		SELECT id FROM crm_categories
		WHERE company_id = $1 AND TRIM(name) ILIKE $2
		LIMIT 1`, companyID, categoryName).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err := tx.Exec(ctx, `UPDATE crm_categories SET updated_at = now() WHERE id = $1`, existingID)
		if err != nil {
			return "", false, fmt.Errorf("touch category hub: %w", err)
		}
		return existingID, false, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO crm_categories (id, company_id, name, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, now(), now())
		ON CONFLICT (company_id, name) DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING id`, companyID, categoryName).Scan(&id)
	if err != nil {
		return "", false, fmt.Errorf("upsert category hub: %w", err)
	}
	return id, true, nil
}

func upsertProductHubTx(ctx context.Context, tx pgx.Tx, companyID, productCode, productName, categoryName string) (string, bool, error) {
	productCode = strings.TrimSpace(productCode)
	productName = strings.TrimSpace(productName)
	categoryName = strings.ToUpper(strings.TrimSpace(categoryName))
	if productCode == "" || productName == "" {
		return "", false, domain.ErrInvalidInput
	}
	var existingID string
	if err := tx.QueryRow(ctx, `SELECT id FROM crm_products_hub WHERE company_id = $1 AND product_code = $2`, companyID, productCode).Scan(&existingID); err == nil && existingID != "" {
		_, err := tx.Exec(ctx, `
			UPDATE crm_products_hub
			SET product_name = $2, category = NULLIF($3, ''), updated_at = now()
			WHERE id = $1`, existingID, productName, categoryName)
		if err != nil {
			return "", false, fmt.Errorf("touch product hub: %w", err)
		}
		return existingID, false, nil
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO crm_products_hub (id, company_id, product_code, product_name, category, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NULLIF($4, ''), now(), now())
		ON CONFLICT (company_id, product_code) DO UPDATE SET
			product_name = EXCLUDED.product_name,
			category = EXCLUDED.category,
			updated_at = EXCLUDED.updated_at
		RETURNING id`, companyID, productCode, productName, categoryName).Scan(&id)
	if err != nil {
		return "", false, fmt.Errorf("upsert product hub: %w", err)
	}
	return id, true, nil
}

func insertSaleHubTx(ctx context.Context, tx pgx.Tx, in saleHubInput) (string, error) {
	saleID := uuid.NewString()
	_, err := tx.Exec(ctx, `
		INSERT INTO crm_sales_hub (
			id, company_id, customer_id, customer_email, customer_name, sale_date, total_amount, cost_total, profit, items_snapshot, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, $8::jsonb, now())`,
		saleID, in.CompanyID, in.CustomerID, in.CustomerEmail, in.CustomerName, in.SaleDate, in.TotalAmount, in.ItemsSnapshot)
	if err != nil {
		return "", fmt.Errorf("insert sale hub: %w", err)
	}
	return saleID, nil
}

func recalculateCustomerProfileTx(ctx context.Context, tx pgx.Tx, companyID, customerID string, saleDate time.Time, totalAmount float64) error {
	metadata := map[string]any{
		"ordersCount":      1,
		"distinctProducts": 0,
		"lastPurchaseDate": saleDate.Format("01/2006"),
		"mainCategory":     "",
		"productsList":     "",
		"followUpStrategy": "",
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal profile metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO crm_customer_profiles (id, customer_id, company_id, ltv, metadata, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4::jsonb, now(), now())
		ON CONFLICT (customer_id) DO UPDATE SET
			ltv = crm_customer_profiles.ltv + EXCLUDED.ltv,
			metadata = jsonb_set(
				jsonb_set(COALESCE(crm_customer_profiles.metadata, '{}'::jsonb), '{ordersCount}', to_jsonb(COALESCE((crm_customer_profiles.metadata->>'ordersCount')::int, 0) + 1), true),
				'{lastPurchaseDate}', to_jsonb($5::text), true
			),
			updated_at = now()`, customerID, companyID, totalAmount, metadataJSON, saleDate.Format("01/2006"))
	if err != nil {
		return fmt.Errorf("recalculate customer profile: %w", err)
	}
	return nil
}

func salesCell(headerMap map[string]int, row []string, keys ...string) string {
	idx, ok := findHeaderIndex(headerMap, keys...)
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func trimCell(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
}

func parseSalesDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("fecha vacía")
	}
	formats := []string{"2006-01-02", "02/01/2006", "02-01-2006", time.RFC3339, "2006-01-02 15:04:05"}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("formato de fecha no reconocido: %s", value)
}

func normalizeImportPhone(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "+") {
		return value
	}
	digits := make([]rune, 0, len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) == 10 {
		return "+57" + string(digits)
	}
	if len(digits) > 0 {
		return string(digits)
	}
	return value
}

func emptyStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

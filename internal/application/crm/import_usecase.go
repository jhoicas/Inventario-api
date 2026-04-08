package crm

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

var importJobs = sync.Map{}

// JobProgress representa el estado de un job de importación en background.
type JobProgress struct {
	TotalRows        int
	ValidRows        int
	InvalidRows      int
	DuplicateRows    int
	MissingEmailRows int
	WarningRows      int
	ProcessedRows    int
	Status           string
}

type importValidatedRow struct {
	RowNumber int
	Profile   dto.ImportCRMProfileRequest
	Preview   dto.ImportPreviewRow
}

// ImportUseCase gestiona la importación masiva de perfiles CRM.
type ImportUseCase struct {
	profileRepo     repository.CRMProfileRepository
	customerRepo    repository.CustomerRepository
	categoryRepo    repository.CRMCategoryRepository
	taskRepo        repository.CRMTaskRepository
	opportunityRepo repository.CRMOpportunityRepository
}

// NewImportUseCase construye el caso de uso de importación.
func NewImportUseCase(
	profileRepo repository.CRMProfileRepository,
	customerRepo repository.CustomerRepository,
	categoryRepo repository.CRMCategoryRepository,
	taskRepo repository.CRMTaskRepository,
	opportunityRepo repository.CRMOpportunityRepository,
) *ImportUseCase {
	return &ImportUseCase{
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
	validatedRows, preview := uc.validateImportRows(rows)

	jobID := uuid.NewString()
	importJobs.Store(jobID, JobProgress{
		TotalRows:        preview.Summary.TotalRows,
		ValidRows:        preview.Summary.ValidRows,
		InvalidRows:      preview.Summary.InvalidRows,
		DuplicateRows:    preview.Summary.DuplicateRows,
		MissingEmailRows: preview.Summary.MissingEmailRows,
		WarningRows:      preview.Summary.WarningRows,
		ProcessedRows:    0,
		Status:           "processing",
	})

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
	p, castOK := v.(JobProgress)
	if !castOK {
		return JobProgress{}, false
	}
	return p, true
}

// processRowsAsync procesa filas en background y actualiza el progreso del job.
func (uc *ImportUseCase) processRowsAsync(ctx context.Context, companyID string, userID string, rows []importValidatedRow, totalRows int, jobID string) {
	defer func() {
		if r := recover(); r != nil {
			uc.setJobProgress(jobID, JobProgress{TotalRows: totalRows, ProcessedRows: len(rows), Status: "error"})
		}
	}()

	if len(rows) == 0 {
		uc.setJobProgress(jobID, JobProgress{TotalRows: totalRows, ProcessedRows: 0, Status: "completed"})
		return
	}

	var wg sync.WaitGroup
	jobs := make(chan importValidatedRow, len(rows))
	var processedCount atomic.Int32
	var emailLocks sync.Map

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
							uc.setJobProgress(jobID, JobProgress{TotalRows: totalRows, ProcessedRows: int(current), Status: "processing"})
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

					_, _ = uc.upsertProfile(ctx, companyID, userID, item.Profile)
				}(item)
			}
		}()
	}

	for _, row := range rows {
		jobs <- row
	}
	close(jobs)

	wg.Wait()
	uc.setJobProgress(jobID, JobProgress{TotalRows: totalRows, ProcessedRows: int(processedCount.Load()), Status: "completed"})
}

func (uc *ImportUseCase) updateProcessed(jobID string, totalRows int, processedRows int) {
	// Actualización en cada iteración para reflejar progreso en tiempo real.
	uc.setJobProgress(jobID, JobProgress{TotalRows: totalRows, ProcessedRows: processedRows, Status: "processing"})
}

func (uc *ImportUseCase) setJobProgress(jobID string, progress JobProgress) {
	importJobs.Store(jobID, progress)
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

	// Busca y asigna campos por nombre de columna (admite espacios, underscore y acentos).
	if idx, ok := findHeaderIndex(headerMap, "nombre"); ok && idx < len(row) {
		profile.Nombre = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "idcliente", "id cliente", "id_cliente"); ok && idx < len(row) {
		profile.IDCliente = strings.TrimSpace(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "email"); ok && idx < len(row) {
		profile.Email = normalizeImportEmail(row[idx])
	}
	if idx, ok := findHeaderIndex(headerMap, "segmento"); ok && idx < len(row) {
		profile.Segmento = strings.TrimSpace(row[idx])
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

	return profile
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
		profile := uc.parseRow(headerMap, row)
		previewRow := dto.ImportPreviewRow{
			Row:             rowNumber,
			Email:           profile.Email,
			NormalizedEmail: profile.Email,
			IDCliente:       strings.TrimSpace(profile.IDCliente),
			LastPurchase:    strings.TrimSpace(profile.UltimaCompra),
			Valid:           true,
		}
		if previewRow.NormalizedEmail != "" {
			emailCounts[previewRow.NormalizedEmail]++
		}
		if previewRow.NormalizedEmail == "" {
			previewRow.Valid = false
			previewRow.Errors = append(previewRow.Errors, "email es obligatorio")
		}
		if strings.TrimSpace(profile.UltimaCompra) == "" {
			previewRow.Warnings = append(previewRow.Warnings, "última compra vacía")
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

func findHeaderIndex(headerMap map[string]int, keys ...string) (int, bool) {
	for _, key := range keys {
		if idx, ok := headerMap[strings.ToLower(strings.TrimSpace(key))]; ok {
			return idx, true
		}
	}
	return 0, false
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
) (bool, error) {
	// Busca cliente únicamente por email. IDCliente se conserva como dato, pero no como clave de coincidencia.
	var (
		customer *entity.Customer
		err      error
	)
	customer, err = uc.customerRepo.GetByCompanyAndEmail(companyID, profile.Email)
	if err != nil {
		return false, fmt.Errorf("buscar cliente por email: %w", err)
	}
	if customer == nil {
		now := time.Now()
		name := strings.TrimSpace(profile.Nombre)
		if name == "" {
			name = strings.Split(profile.Email, "@")[0]
		}
		taxID := strings.TrimSpace(profile.IDCliente)
		if taxID == "" {
			taxID = uc.buildTempTaxID()
		}

		customer = &entity.Customer{
			ID:        uuid.NewString(),
			CompanyID: companyID,
			Name:      name,
			TaxID:     taxID,
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
	// VentasTotales del Excel se persiste como LTV del perfil CRM.
	if profile.VentasTotales > 0 {
		ltv = decimal.NewFromFloat(profile.VentasTotales)
	}

	categoryID, err := uc.resolveCategoryID(companyID, profile.Segmento)
	if err != nil {
		return false, fmt.Errorf("resolver categoría por segmento: %w", err)
	}

	metadata := entity.ProfileMetadata{
		OrdersCount:      profile.Pedidos,
		DistinctProducts: profile.Productos,
		LastPurchaseDate: profile.UltimaCompra,
		// CategoriaProducto del Excel se persiste en metadata.mainCategory.
		MainCategory:     profile.CategoriaProducto,
		ProductsList:     profile.DescripcionProductos,
		FollowUpStrategy: profile.EstrategiaSeguimiento,
	}

	upsertProfile := &entity.CRMCustomerProfile{
		CustomerID: customer.ID,
		CompanyID:  companyID,
		CategoryID: categoryID,
		LTV:        ltv,
		Metadata:   metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if existingProfile != nil {
		upsertProfile.ID = existingProfile.ID
		upsertProfile.CreatedAt = existingProfile.CreatedAt
	}

	if err := uc.profileRepo.Upsert(upsertProfile); err != nil {
		return false, fmt.Errorf("upsert perfil: %w", err)
	}

	if err := uc.createAutomationArtifacts(ctx, companyID, userID, customer.ID, profile); err != nil {
		return false, err
	}

	return existingProfile == nil, nil
}

func (uc *ImportUseCase) resolveCategoryID(companyID, segment string) (string, error) {
	segment = strings.TrimSpace(segment)
	if segment == "" || uc.categoryRepo == nil {
		return "", nil
	}
	categories, _, err := uc.categoryRepo.ListByCompany(companyID, 200, 0)
	if err != nil {
		return "", err
	}
	for _, cat := range categories {
		if strings.EqualFold(strings.TrimSpace(cat.Name), segment) {
			return cat.ID, nil
		}
	}
	return "", nil
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

func (uc *ImportUseCase) buildTempTaxID() string {
	// Prefijo CF (Cliente Fiscal) + sufijo UUID corto para minimizar colisiones.
	return "CF-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:12]
}

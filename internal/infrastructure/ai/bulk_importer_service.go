package ai

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/jhoicas/Inventario-api/pkg/logger"
	"github.com/xuri/excelize/v2"
)

// BulkImporterService implementa importacion de datos CSV/Excel hacia tablas hub.
// Soporta lotes de 1000 registros con auto-creacion de productos y manejo de duplicados.
type BulkImporterService struct {
	productRepo repository.CRMProductHubRepository
	salesRepo   repository.CRMSalesHubRepository
	itemsRepo   repository.CRMSaleItemHubRepository
	log         *logger.Logger
}

// NewBulkImporterService constructor.
func NewBulkImporterService(
	productRepo repository.CRMProductHubRepository,
	salesRepo repository.CRMSalesHubRepository,
	itemsRepo repository.CRMSaleItemHubRepository,
	log *logger.Logger,
) *BulkImporterService {
	return &BulkImporterService{
		productRepo: productRepo,
		salesRepo:   salesRepo,
		itemsRepo:   itemsRepo,
		log:         log,
	}
}

const batchSize = 1000

// ImportFromCSV lee un archivo CSV y carga datos en lotes.
// tableName: "products", "sales", o "sale_items"
func (s *BulkImporterService) ImportFromCSV(
	ctx context.Context,
	companyID, filePath, tableName string,
) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("abrir archivo CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("leer header CSV: %w", err)
	}

	var totalImported int
	var currentBatch []map[string]string
	lineNum := 1 // para logging

	for {
		record, err := reader.Read()
		if err == io.EOF {
			// Procesar ultimo lote si queda pendiente
			if len(currentBatch) > 0 {
				imported, batchErr := s.processBatch(ctx, companyID, tableName, currentBatch, header)
				if batchErr != nil {
					s.log.Error().Err(batchErr).Int("line", lineNum).Msg("error procesando lote final de CSV")
					return totalImported, fmt.Errorf("procesar lote final: %w", batchErr)
				}
				totalImported += imported
			}
			break
		}
		if err != nil {
			return totalImported, fmt.Errorf("leer linea CSV %d: %w", lineNum, err)
		}

		// Construir mapa de valores con headers
		recordMap := make(map[string]string)
		for i, val := range record {
			if i < len(header) {
				recordMap[header[i]] = val
			}
		}
		currentBatch = append(currentBatch, recordMap)

		// Procesar lote cuando alcanza 1000 registros
		if len(currentBatch) >= batchSize {
			imported, batchErr := s.processBatch(ctx, companyID, tableName, currentBatch, header)
			if batchErr != nil {
				s.log.Error().Err(batchErr).Int("batch_start_line", lineNum-len(currentBatch)).Msg("error procesando lote CSV")
			}
			totalImported += imported
			currentBatch = nil
		}

		lineNum++
	}

	s.log.Info().Int("total_imported", totalImported).Str("table", tableName).Msg("CSV import completado")
	return totalImported, nil
}

// ImportFromExcel lee un archivo Excel y carga datos en lotes.
// tableName: "products", "sales", o "sale_items"
func (s *BulkImporterService) ImportFromExcel(
	ctx context.Context,
	companyID, filePath, sheetName, tableName string,
) (int, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("abrir archivo Excel: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("leer sheet %s: %w", sheetName, err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("archivo Excel vacio o sin datos (minimo header + 1 fila requerido)")
	}

	header := rows[0]
	var totalImported int
	var currentBatch []map[string]string

	for i := 1; i < len(rows); i++ {
		record := rows[i]

		// Construir mapa de valores
		recordMap := make(map[string]string)
		for j, val := range record {
			if j < len(header) {
				recordMap[header[j]] = val
			}
		}
		currentBatch = append(currentBatch, recordMap)

		// Procesar lote cuando alcanza 1000 registros
		if len(currentBatch) >= batchSize {
			imported, batchErr := s.processBatch(ctx, companyID, tableName, currentBatch, header)
			if batchErr != nil {
				s.log.Error().Err(batchErr).Int("row", i).Msg("error procesando lote Excel")
			}
			totalImported += imported
			currentBatch = nil
		}
	}

	// Procesar ultimo lote
	if len(currentBatch) > 0 {
		imported, batchErr := s.processBatch(ctx, companyID, tableName, currentBatch, header)
		if batchErr != nil {
			s.log.Error().Err(batchErr).Msg("error procesando lote final Excel")
		}
		totalImported += imported
	}

	s.log.Info().Int("total_imported", totalImported).Str("table", tableName).Str("sheet", sheetName).Msg("Excel import completado")
	return totalImported, nil
}

// processBatch convierte el lote generico en las entidades correspondientes e inserta.
func (s *BulkImporterService) processBatch(
	ctx context.Context,
	companyID, tableName string,
	batch []map[string]string,
	headers []string,
) (int, error) {
	if len(batch) == 0 {
		return 0, nil
	}

	switch strings.ToLower(tableName) {
	case "products", "crm_products_hub":
		return s.processProductsBatch(ctx, companyID, batch)
	case "sales", "crm_sales_hub":
		return s.proceseSalesBatch(ctx, companyID, batch)
	case "sale_items", "crm_sale_items_hub":
		return s.proceseSaleItemsBatch(ctx, companyID, batch)
	default:
		return 0, fmt.Errorf("tabla no soportada: %s", tableName)
	}
}

// processProductsBatch importa productos y crea los que falten.
// Columnas esperadas: product_code, product_name, category, unit_cost
func (s *BulkImporterService) processProductsBatch(
	ctx context.Context,
	companyID string,
	batch []map[string]string,
) (int, error) {
	products := make([]*entity.ProductHub, 0, len(batch))

	for _, record := range batch {
		code := strings.TrimSpace(record["product_code"])
		name := strings.TrimSpace(record["product_name"])

		if code == "" || name == "" {
			s.log.Warn().Msg("saltando producto con codigo o nombre vacio")
			continue
		}

		// Auto-crear producto si no existe
		existing, err := s.productRepo.GetByCompanyAndCode(ctx, companyID, code)
		if err != nil {
			s.log.Warn().Err(err).Str("code", code).Msg("verificar existencia de producto")
		}

		if existing != nil {
			s.log.Debug().Str("code", code).Msg("producto ya existe, actualizando")
			// Upsert si ya existe
			cost, _ := parseFloat(record["unit_cost"])
			existing.ProductName = name
			existing.Category = stringPtr(record["category"])
			existing.UnitCost = cost
			existing.UpdatedAt = time.Now()
			s.productRepo.Upsert(ctx, existing)
			continue
		}

		cost, _ := parseFloat(record["unit_cost"])
		product := &entity.ProductHub{
			ID:          uuid.New().String(),
			CompanyID:   companyID,
			ProductCode: code,
			ProductName: name,
			Category:    stringPtr(record["category"]),
			UnitCost:    cost,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		products = append(products, product)
	}

	if len(products) > 0 {
		if err := s.productRepo.CreateBatch(ctx, products); err != nil {
			return 0, fmt.Errorf("crear lote de productos: %w", err)
		}
	}

	return len(batch), nil
}

// proceseSalesBatch importa ventas.
// Columnas esperadas: customer_email, customer_name, customer_city, sale_date, total_amount, cost_total, profit
func (s *BulkImporterService) proceseSalesBatch(
	ctx context.Context,
	companyID string,
	batch []map[string]string,
) (int, error) {
	sales := make([]*entity.SaleHub, 0, len(batch))

	for _, record := range batch {
		email := strings.TrimSpace(record["customer_email"])
		if email == "" {
			s.log.Warn().Msg("saltando venta sin customer_email")
			continue
		}

		saleDate, _ := parseTime(record["sale_date"])
		total := 0.0
		if t, err := parseFloat(record["total_amount"]); err == nil && t != nil {
			total = *t
		}

		sale := &entity.SaleHub{
			ID:            uuid.New().String(),
			CompanyID:     companyID,
			CustomerEmail: email,
			CustomerName:  stringPtr(record["customer_name"]),
			CustomerCity:  stringPtr(record["customer_city"]),
			SaleDate:      saleDate,
			TotalAmount:   total,
			CostTotal:     parseFloatPtr(record["cost_total"]),
			Profit:        parseFloatPtr(record["profit"]),
			CreatedAt:     time.Now(),
		}
		sales = append(sales, sale)
	}

	if len(sales) > 0 {
		if err := s.salesRepo.CreateBatch(ctx, sales); err != nil {
			return 0, fmt.Errorf("crear lote de ventas: %w", err)
		}
	}

	return len(batch), nil
}

// proceseSaleItemsBatch importa items de venta.
// Columnas esperadas: sales_id, product_id, quantity, unit_price, line_total
func (s *BulkImporterService) proceseSaleItemsBatch(
	ctx context.Context,
	companyID string,
	batch []map[string]string,
) (int, error) {
	items := make([]*entity.SaleItemHub, 0, len(batch))

	for _, record := range batch {
		salesID := strings.TrimSpace(record["sales_id"])
		productID := strings.TrimSpace(record["product_id"])

		if salesID == "" || productID == "" {
			s.log.Warn().Msg("saltando item de venta sin sales_id o product_id")
			continue
		}

		qty, _ := strconv.Atoi(record["quantity"])
		unitPrice := 0.0
		if up, err := parseFloat(record["unit_price"]); err == nil && up != nil {
			unitPrice = *up
		}
		lineTotal := 0.0
		if lt, err := parseFloat(record["line_total"]); err == nil && lt != nil {
			lineTotal = *lt
		}

		item := &entity.SaleItemHub{
			ID:        uuid.New().String(),
			SalesID:   salesID,
			ProductID: productID,
			Quantity:  qty,
			UnitPrice: unitPrice,
			LineTotal: lineTotal,
			CreatedAt: time.Now(),
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.itemsRepo.CreateBatch(ctx, items); err != nil {
			return 0, fmt.Errorf("crear lote de items: %w", err)
		}
	}

	return len(batch), nil
}

// ValidateImportData valida que los registros cumplan con esquema esperado.
func (s *BulkImporterService) ValidateImportData(
	records []map[string]interface{},
	tableName string,
) []error {
	var errors []error

	for i, record := range records {
		var rowErrors []string

		switch strings.ToLower(tableName) {
		case "products", "crm_products_hub":
			if _, ok := record["product_code"]; !ok {
				rowErrors = append(rowErrors, "product_code requerido")
			}
			if _, ok := record["product_name"]; !ok {
				rowErrors = append(rowErrors, "product_name requerido")
			}

		case "sales", "crm_sales_hub":
			if _, ok := record["customer_email"]; !ok {
				rowErrors = append(rowErrors, "customer_email requerido")
			}
			if _, ok := record["sale_date"]; !ok {
				rowErrors = append(rowErrors, "sale_date requerido")
			}
			if _, ok := record["total_amount"]; !ok {
				rowErrors = append(rowErrors, "total_amount requerido")
			}

		case "sale_items", "crm_sale_items_hub":
			if _, ok := record["sales_id"]; !ok {
				rowErrors = append(rowErrors, "sales_id requerido")
			}
			if _, ok := record["product_id"]; !ok {
				rowErrors = append(rowErrors, "product_id requerido")
			}
		}

		for _, errMsg := range rowErrors {
			errors = append(errors, fmt.Errorf("fila %d: %s", i+1, errMsg))
		}
	}

	return errors
}

// Helper functions
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseFloat(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return &f, err
}

func parseFloatPtr(s string) *float64 {
	f, _ := parseFloat(s)
	return f
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	s = strings.TrimSpace(s)
	// Intentar varios formatos comunes
	for _, format := range []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Now(), fmt.Errorf("formato de fecha no reconocido: %s", s)
}

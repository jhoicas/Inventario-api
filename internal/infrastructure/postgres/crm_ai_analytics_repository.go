package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CRMProductHubRepository implementacion PostgreSQL.
type CRMProductHubRepository struct {
	pool *pgxpool.Pool
}

// NewCRMProductHubRepository constructor.
func NewCRMProductHubRepository(pool *pgxpool.Pool) *CRMProductHubRepository {
	return &CRMProductHubRepository{pool: pool}
}

// CreateBatch inserta multiples productos en lote (1000 en chunks por defecto).
func (r *CRMProductHubRepository) CreateBatch(ctx context.Context, products []*entity.ProductHub) error {
	if len(products) == 0 {
		return nil
	}

	const batchSize = 1000
	for start := 0; start < len(products); start += batchSize {
		end := start + batchSize
		if end > len(products) {
			end = len(products)
		}

		chunk := products[start:end]
		values := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*8)
		argPos := 1

		for _, p := range chunk {
			values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6, argPos+7))
			args = append(args, p.ID, p.CompanyID, p.ProductCode, p.ProductName, p.Category, p.UnitCost, p.CreatedAt, p.UpdatedAt)
			argPos += 8
		}

		query := `INSERT INTO crm_products_hub (id, company_id, product_code, product_name, category, unit_cost, created_at, updated_at)
		 VALUES ` + strings.Join(values, ",") + `
		 ON CONFLICT (company_id, product_code) DO NOTHING`

		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insertar productos hub batch: %w", err)
		}
	}

	return nil
}

// GetByCompanyAndCode busca producto por codigo.
func (r *CRMProductHubRepository) GetByCompanyAndCode(ctx context.Context, companyID, productCode string) (*entity.ProductHub, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, product_code, product_name, category, unit_cost, created_at, updated_at
		 FROM crm_products_hub
		 WHERE company_id = $1 AND product_code = $2`,
		companyID, productCode,
	)

	var p entity.ProductHub
	err := row.Scan(&p.ID, &p.CompanyID, &p.ProductCode, &p.ProductName, &p.Category, &p.UnitCost, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("buscar producto hub: %w", err)
	}
	return &p, nil
}

// ListByCompany lista todos los productos de una empresa.
func (r *CRMProductHubRepository) ListByCompany(ctx context.Context, companyID string) ([]*entity.ProductHub, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, product_code, product_name, category, unit_cost, created_at, updated_at
		 FROM crm_products_hub
		 WHERE company_id = $1
		 ORDER BY product_code`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar productos hub: %w", err)
	}
	defer rows.Close()

	var products []*entity.ProductHub
	for rows.Next() {
		var p entity.ProductHub
		err := rows.Scan(&p.ID, &p.CompanyID, &p.ProductCode, &p.ProductName, &p.Category, &p.UnitCost, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan producto hub: %w", err)
		}
		products = append(products, &p)
	}
	return products, rows.Err()
}

// Upsert inserta o actualiza un producto.
func (r *CRMProductHubRepository) Upsert(ctx context.Context, p *entity.ProductHub) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO crm_products_hub (id, company_id, product_code, product_name, category, unit_cost, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (company_id, product_code) 
		 DO UPDATE SET product_name = $4, category = $5, unit_cost = $6, updated_at = $8`,
		p.ID, p.CompanyID, p.ProductCode, p.ProductName, p.Category, p.UnitCost, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert producto hub: %w", err)
	}
	return nil
}

// CRMSalesHubRepository implementacion PostgreSQL.
type CRMSalesHubRepository struct {
	pool *pgxpool.Pool
}

// NewCRMSalesHubRepository constructor.
func NewCRMSalesHubRepository(pool *pgxpool.Pool) *CRMSalesHubRepository {
	return &CRMSalesHubRepository{pool: pool}
}

// CreateBatch inserta multiples ventas en lote.
func (r *CRMSalesHubRepository) CreateBatch(ctx context.Context, sales []*entity.SaleHub) error {
	if len(sales) == 0 {
		return nil
	}

	const batchSize = 1000
	for start := 0; start < len(sales); start += batchSize {
		end := start + batchSize
		if end > len(sales) {
			end = len(sales)
		}

		chunk := sales[start:end]
		values := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*10)
		argPos := 1

		for _, s := range chunk {
			values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6, argPos+7, argPos+8, argPos+9))
			args = append(args, s.ID, s.CompanyID, s.CustomerEmail, s.CustomerName, s.CustomerCity, s.SaleDate, s.TotalAmount, s.CostTotal, s.Profit, s.CreatedAt)
			argPos += 10
		}

		query := `INSERT INTO crm_sales_hub (id, company_id, customer_email, customer_name, customer_city, sale_date, total_amount, cost_total, profit, created_at)
		 VALUES ` + strings.Join(values, ",")

		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insertar ventas hub batch: %w", err)
		}
	}

	return nil
}

// GetByID obtiene una venta por ID.
func (r *CRMSalesHubRepository) GetByID(ctx context.Context, saleID string) (*entity.SaleHub, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, company_id, customer_email, customer_name, customer_city, sale_date, total_amount, cost_total, profit, created_at
		 FROM crm_sales_hub
		 WHERE id = $1`,
		saleID,
	)

	var s entity.SaleHub
	err := row.Scan(&s.ID, &s.CompanyID, &s.CustomerEmail, &s.CustomerName, &s.CustomerCity, &s.SaleDate, &s.TotalAmount, &s.CostTotal, &s.Profit, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("buscar venta hub: %w", err)
	}
	return &s, nil
}

// ListByCompanyAndDateRange lista ventas por rango de fechas.
func (r *CRMSalesHubRepository) ListByCompanyAndDateRange(ctx context.Context, companyID string, startDate, endDate time.Time) ([]*entity.SaleHub, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, customer_email, customer_name, customer_city, sale_date, total_amount, cost_total, profit, created_at
		 FROM crm_sales_hub
		 WHERE company_id = $1 AND sale_date >= $2 AND sale_date < $3
		 ORDER BY sale_date DESC`,
		companyID, startDate, endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("listar ventas hub: %w", err)
	}
	defer rows.Close()

	var sales []*entity.SaleHub
	for rows.Next() {
		var s entity.SaleHub
		err := rows.Scan(&s.ID, &s.CompanyID, &s.CustomerEmail, &s.CustomerName, &s.CustomerCity, &s.SaleDate, &s.TotalAmount, &s.CostTotal, &s.Profit, &s.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan venta hub: %w", err)
		}
		sales = append(sales, &s)
	}
	return sales, rows.Err()
}

// CRMSaleItemHubRepository implementacion PostgreSQL.
type CRMSaleItemHubRepository struct {
	pool *pgxpool.Pool
}

// NewCRMSaleItemHubRepository constructor.
func NewCRMSaleItemHubRepository(pool *pgxpool.Pool) *CRMSaleItemHubRepository {
	return &CRMSaleItemHubRepository{pool: pool}
}

// CreateBatch inserta multiples items de venta en lote.
func (r *CRMSaleItemHubRepository) CreateBatch(ctx context.Context, items []*entity.SaleItemHub) error {
	if len(items) == 0 {
		return nil
	}

	const batchSize = 1000
	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}

		chunk := items[start:end]
		values := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*7)
		argPos := 1

		for _, item := range chunk {
			values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)", argPos, argPos+1, argPos+2, argPos+3, argPos+4, argPos+5, argPos+6))
			args = append(args, item.ID, item.SalesID, item.ProductID, item.Quantity, item.UnitPrice, item.LineTotal, item.CreatedAt)
			argPos += 7
		}

		query := `INSERT INTO crm_sale_items_hub (id, sales_id, product_id, quantity, unit_price, line_total, created_at)
		 VALUES ` + strings.Join(values, ",")

		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insertar items hub batch: %w", err)
		}
	}

	return nil
}

// GetBySaleID obtiene todos los items de una venta.
func (r *CRMSaleItemHubRepository) GetBySaleID(ctx context.Context, saleID string) ([]*entity.SaleItemHub, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, sales_id, product_id, quantity, unit_price, line_total, created_at
		 FROM crm_sale_items_hub
		 WHERE sales_id = $1
		 ORDER BY created_at`,
		saleID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar items venta hub: %w", err)
	}
	defer rows.Close()

	var items []*entity.SaleItemHub
	for rows.Next() {
		var item entity.SaleItemHub
		err := rows.Scan(&item.ID, &item.SalesID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.LineTotal, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan item hub: %w", err)
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

// AIAnalyticsRepository implementacion PostgreSQL para consultas sobre la vista semantica.
type AIAnalyticsRepository struct {
	pool *pgxpool.Pool
}

// NewAIAnalyticsRepository constructor.
func NewAIAnalyticsRepository(pool *pgxpool.Pool) *AIAnalyticsRepository {
	return &AIAnalyticsRepository{pool: pool}
}

// QueryView ejecuta una consulta SQL sobre la vista v_crm_ai_analytics de forma segura.
// El SQL debe estar ya sanitizado con company_id inyectado.
func (r *AIAnalyticsRepository) QueryView(ctx context.Context, companyID, sqlQuery string) ([]*entity.AIAnalyticsRow, error) {
	// Inyeccion adicional de company_id como extra layer de seguridad (deja que SQLGuard valide el SQL antes)
	safeSQLQuery := fmt.Sprintf(`%s AND company_id = $1`, sqlQuery)

	rows, err := r.pool.Query(ctx, safeSQLQuery, companyID)
	if err != nil {
		return nil, fmt.Errorf("queryview: %w", err)
	}
	defer rows.Close()

	var analytics []*entity.AIAnalyticsRow
	for rows.Next() {
		var row entity.AIAnalyticsRow
		err := rows.Scan(
			&row.CompanyID, &row.Fecha, &row.ClienteNombre, &row.Ciudad, &row.Producto,
			&row.Categoria, &row.Cantidad, &row.PrecioUnitario, &row.IngresoNeto,
			&row.CostoTotal, &row.Utilidad, &row.CustomerEmail, &row.SaleID, &row.ItemID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan analytics row: %w", err)
		}
		analytics = append(analytics, &row)
	}
	return analytics, rows.Err()
}

// RunAggregateQuery ejecuta consultas de agregacion como COUNT, SUM, AVG sobre la vista.
func (r *AIAnalyticsRepository) RunAggregateQuery(ctx context.Context, companyID, sqlQuery string) (interface{}, error) {
	safeSQLQuery := fmt.Sprintf(`%s AND company_id = $1`, sqlQuery)

	rows, err := r.pool.Query(ctx, safeSQLQuery, companyID)
	if err != nil {
		return nil, fmt.Errorf("aggregate query: %w", err)
	}
	defer rows.Close()

	// Retorna lista de mapas para flexibilidad con diferentes tipos de agregacion
	var results []map[string]interface{}
	cols := rows.FieldDescriptions()

	colNames := make([]string, len(cols))
	for i, col := range cols {
		colNames[i] = col.Name
	}

	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("row values: %w", err)
		}
		row := make(map[string]interface{})
		for i, val := range vals {
			row[colNames[i]] = val
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

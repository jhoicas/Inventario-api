package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.ProductRepository = (*ProductRepo)(nil)

// ProductRepo implementación del puerto ProductRepository sobre PostgreSQL (usable con pool o tx).
type ProductRepo struct {
	q Querier
}

// NewProductRepository construye el adaptador de persistencia para productos. Pasar pool o tx (Querier).
func NewProductRepository(q Querier) *ProductRepo {
	return &ProductRepo{q: q}
}

// Create persiste un nuevo producto. Cost inicia en 0.
func (r *ProductRepo) Create(product *entity.Product) error {
	query := `
		INSERT INTO products (id, company_id, sku, name, description, price, cost, tax_rate, unspsc_code, unit_measure, attributes, cogs, reorder_point, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := r.q.Exec(context.Background(), query,
		product.ID, product.CompanyID, product.SKU, product.Name, product.Description,
		product.Price, product.Cost, product.TaxRate, product.UNSPSC_Code, product.UnitMeasure,
		product.Attributes, product.COGS, product.ReorderPoint, product.CreatedAt, product.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

// GetByID obtiene un producto por ID.
func (r *ProductRepo) GetByID(id string) (*entity.Product, error) {
	query := `
		SELECT id, company_id, sku, name, description, price, cost, tax_rate, unspsc_code, unit_measure, attributes, cogs, reorder_point, created_at, updated_at
		FROM products WHERE id = $1`
	var p entity.Product
	err := r.q.QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.CompanyID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Cost, &p.TaxRate,
		&p.UNSPSC_Code, &p.UnitMeasure, &p.Attributes, &p.COGS, &p.ReorderPoint, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

// GetByCompanyAndSKU obtiene un producto por empresa y SKU.
func (r *ProductRepo) GetByCompanyAndSKU(companyID, sku string) (*entity.Product, error) {
	query := `
		SELECT id, company_id, sku, name, description, price, cost, tax_rate, unspsc_code, unit_measure, attributes, cogs, reorder_point, created_at, updated_at
		FROM products WHERE company_id = $1 AND sku = $2`
	var p entity.Product
	err := r.q.QueryRow(context.Background(), query, companyID, sku).Scan(
		&p.ID, &p.CompanyID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Cost, &p.TaxRate,
		&p.UNSPSC_Code, &p.UnitMeasure, &p.Attributes, &p.COGS, &p.ReorderPoint, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get product by sku: %w", err)
	}
	return &p, nil
}

// Update actualiza un producto existente. No permite modificar Cost ni Stock (se manejan vía movimientos).
func (r *ProductRepo) Update(product *entity.Product) error {
	query := `
		UPDATE products SET name = $2, description = $3, price = $4, tax_rate = $5, unspsc_code = $6, unit_measure = $7, attributes = $8, updated_at = $9
		WHERE id = $1`
	cmd, err := r.q.Exec(context.Background(), query,
		product.ID, product.Name, product.Description, product.Price, product.TaxRate,
		product.UNSPSC_Code, product.UnitMeasure, product.Attributes, product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}
	return nil
}

// UpdateCost actualiza solo el costo del producto (usado por el motor de inventario).
func (r *ProductRepo) UpdateCost(productID string, cost decimal.Decimal) error {
	_, err := r.q.Exec(context.Background(),
		`UPDATE products SET cost = $2, updated_at = now() WHERE id = $1`,
		productID, cost,
	)
	if err != nil {
		return fmt.Errorf("update product cost: %w", err)
	}
	return nil
}

// ListByCompany lista productos por empresa con paginación.
func (r *ProductRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.Product, error) {
	query := `
		SELECT id, company_id, sku, name, description, price, cost, tax_rate, unspsc_code, unit_measure, attributes, cogs, reorder_point, created_at, updated_at
		FROM products WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.q.Query(context.Background(), query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var list []*entity.Product
	for rows.Next() {
		var p entity.Product
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.SKU, &p.Name, &p.Description, &p.Price, &p.Cost, &p.TaxRate,
			&p.UNSPSC_Code, &p.UnitMeasure, &p.Attributes, &p.COGS, &p.ReorderPoint, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

// Delete elimina un producto por ID.
func (r *ProductRepo) Delete(id string) error {
	_, err := r.q.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

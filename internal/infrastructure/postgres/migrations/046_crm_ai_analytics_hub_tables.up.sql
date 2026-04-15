-- CRM AI Analytics Hub Tables and Semantic Layer
-- Purpose: Data warehouse tables for AI-driven analytics with company isolation

-- 1. Products Hub: Dimension table for products available in company repositories
CREATE TABLE IF NOT EXISTS crm_products_hub (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    product_code VARCHAR(50) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    unit_cost NUMERIC(12, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Prevent duplicates per company
    UNIQUE(company_id, product_code)
);

CREATE INDEX idx_crm_products_hub_company_id ON crm_products_hub(company_id);
CREATE INDEX idx_crm_products_hub_product_code ON crm_products_hub(company_id, product_code);

-- 2. Sales Hub: Fact table for sales transactions aggregated with customer email linkage
CREATE TABLE IF NOT EXISTS crm_sales_hub (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_email VARCHAR(255) NOT NULL,
    customer_name VARCHAR(255),
    customer_city VARCHAR(100),
    sale_date TIMESTAMPTZ NOT NULL,
    total_amount NUMERIC(12, 2) NOT NULL,
    cost_total NUMERIC(12, 2),
    profit NUMERIC(12, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key to crm_customers via email for multi-company isolation
    CONSTRAINT fk_crm_sales_hub_customer FOREIGN KEY (company_id, customer_email) 
        REFERENCES crm_customers(company_id, email) ON DELETE CASCADE MATCH FULL
);

CREATE INDEX idx_crm_sales_hub_company_id ON crm_sales_hub(company_id);
CREATE INDEX idx_crm_sales_hub_customer_email ON crm_sales_hub(company_id, customer_email);
CREATE INDEX idx_crm_sales_hub_sale_date ON crm_sales_hub(company_id, sale_date);

-- 3. Sale Items Hub: Line items of sales with product cost and quantity
CREATE TABLE IF NOT EXISTS crm_sale_items_hub (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_id UUID NOT NULL REFERENCES crm_sales_hub(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES crm_products_hub(id) ON DELETE CASCADE,
    quantity INT NOT NULL,
    unit_price NUMERIC(12, 2) NOT NULL,
    line_total NUMERIC(12, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_crm_sale_items_hub_sales_id ON crm_sale_items_hub(sales_id);
CREATE INDEX idx_crm_sale_items_hub_product_id ON crm_sale_items_hub(product_id);

-- 4. Semantic View: Flatten hub tables for AI analytics queries
-- This view enforces company isolation via WHERE clause in queries
CREATE OR REPLACE VIEW v_crm_ai_analytics AS
SELECT 
    sh.company_id,
    s.sale_date::DATE as fecha,
    sh.customer_name as cliente_nombre,
    sh.customer_city as ciudad,
    ph.product_name as producto,
    ph.category as categoria,
    sih.quantity as cantidad,
    sih.unit_price as precio_unitario,
    sih.line_total as ingreso_neto,
    (sih.quantity * ph.unit_cost) as costo_total,
    (sih.line_total - (sih.quantity * COALESCE(ph.unit_cost, 0))) as utilidad,
    sh.customer_email,
    sh.id as sale_id,
    sih.id as item_id
FROM crm_sales_hub sh
INNER JOIN crm_sale_items_hub sih ON sh.id = sih.sales_id
INNER JOIN crm_products_hub ph ON sih.product_id = ph.id
INNER JOIN crm_sales s ON sh.id = s.id
WHERE sh.created_at IS NOT NULL; -- Placeholder for company_id filtering in queries

-- 5. Function to ensure company isolation on analytics view
-- This function can be used in application-level query builders
CREATE OR REPLACE FUNCTION crm_ai_analytics_with_company(p_company_id UUID)
RETURNS TABLE (
    company_id UUID,
    fecha DATE,
    cliente_nombre VARCHAR,
    ciudad VARCHAR,
    producto VARCHAR,
    categoria VARCHAR,
    cantidad INT,
    precio_unitario NUMERIC,
    ingreso_neto NUMERIC,
    costo_total NUMERIC,
    utilidad NUMERIC,
    customer_email VARCHAR,
    sale_id UUID,
    item_id UUID
) AS $$
SELECT 
    v.company_id,
    v.fecha,
    v.cliente_nombre,
    v.ciudad,
    v.producto,
    v.categoria,
    v.cantidad,
    v.precio_unitario,
    v.ingreso_neto,
    v.costo_total,
    v.utilidad,
    v.customer_email,
    v.sale_id,
    v.item_id
FROM v_crm_ai_analytics v
WHERE v.company_id = p_company_id;
$$ LANGUAGE SQL STABLE;

-- Indexes for analytics performance
CREATE INDEX idx_crm_analytics_company_date ON crm_sales_hub(company_id, sale_date DESC);
CREATE INDEX idx_crm_analytics_email_date ON crm_sales_hub(company_id, customer_email, sale_date DESC);

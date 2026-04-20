-- Soft-delete / filtro de catálogo: flag is_active en productos de inventario (tabla products).
-- Las categorías de producto para CRM/hub viven en crm_category_product_hub (creada en migración 056).

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_products_company_active ON products (company_id, is_active);

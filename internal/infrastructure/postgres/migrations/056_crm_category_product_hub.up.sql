-- Categorías de producto para el hub de analytics / importación de ventas.
-- Separadas de crm_categories (fidelización / perfiles cliente).

CREATE TABLE IF NOT EXISTS crm_category_product_hub (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_category_product_hub_company_name
    ON crm_category_product_hub (company_id, name);

CREATE INDEX IF NOT EXISTS idx_crm_category_product_hub_company_id
    ON crm_category_product_hub (company_id);

-- Si la tabla ya existía sin alguna columna (IF NOT EXISTS no añade columnas nuevas), completar esquema.
ALTER TABLE crm_category_product_hub
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE crm_category_product_hub
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Productos con category_id huérfano respecto a crm_categories (no copiables al nuevo hub)
UPDATE crm_products_hub ph
SET category_id = NULL
WHERE ph.category_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM crm_categories c WHERE c.id = ph.category_id);

-- Si varias filas en crm_categories comparten empresa+nombre, unificar referencias en productos al id canónico
WITH cat_refs AS (
    SELECT
        c.id,
        first_value(c.id) OVER (
            PARTITION BY c.company_id, upper(btrim(c.name))
            ORDER BY c.created_at ASC NULLS LAST, c.id ASC
        ) AS keeper_id
    FROM crm_categories c
    WHERE c.id IN (SELECT DISTINCT ph.category_id FROM crm_products_hub ph WHERE ph.category_id IS NOT NULL)
)
UPDATE crm_products_hub ph
SET category_id = cr.keeper_id
FROM cat_refs cr
WHERE ph.category_id = cr.id
  AND cr.id <> cr.keeper_id;

-- Copiar al hub de producto conservando el mismo UUID (crm_products_hub.category_id no cambia de valor)
INSERT INTO crm_category_product_hub (id, company_id, name, created_at, updated_at)
SELECT c.id, c.company_id, c.name, COALESCE(c.created_at, now()), COALESCE(c.updated_at, now())
FROM crm_categories c
WHERE c.id IN (SELECT DISTINCT ph.category_id FROM crm_products_hub ph WHERE ph.category_id IS NOT NULL)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE crm_products_hub DROP CONSTRAINT IF EXISTS crm_products_hub_category_id_fkey;

ALTER TABLE crm_products_hub
    ADD CONSTRAINT crm_products_hub_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES crm_category_product_hub (id) ON DELETE SET NULL;

DROP VIEW IF EXISTS v_crm_ai_analytics CASCADE;

CREATE VIEW v_crm_ai_analytics AS
SELECT
    sh.company_id,
    sh.sale_date::date AS fecha,
    sh.customer_name AS cliente_nombre,
    lc.name::varchar AS ciudad,
    ph.product_name AS producto,
    pch.name::varchar AS categoria,
    sih.quantity AS cantidad,
    sih.unit_price AS precio_unitario,
    sih.line_total AS ingreso_neto,
    (sih.quantity * ph.unit_cost) AS costo_total,
    (sih.line_total - (sih.quantity * coalesce(ph.unit_cost, 0))) AS utilidad,
    sh.customer_email,
    sh.id AS sale_id,
    sih.id AS item_id
FROM crm_sales_hub sh
INNER JOIN crm_sale_items_hub sih ON sh.id = sih.sales_id
INNER JOIN crm_products_hub ph ON sih.product_id = ph.id
LEFT JOIN crm_category_product_hub pch ON pch.id = ph.category_id
LEFT JOIN customers cu ON cu.id = sh.customer_id AND cu.company_id = sh.company_id
LEFT JOIN locations_cities lc ON lc.id = cu.city_id
WHERE sh.created_at IS NOT NULL;

CREATE OR REPLACE FUNCTION crm_ai_analytics_with_company (p_company_id UUID)
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
)
LANGUAGE sql
STABLE
AS $$
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
$$;

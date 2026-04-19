-- Relacionar productos hub con crm_categories por UUID (category_id) y alinear la vista analítica.

ALTER TABLE crm_products_hub
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES crm_categories (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_crm_products_hub_category_id ON crm_products_hub (category_id);

-- Si aún existe la columna legada de texto "category", intentar enlazar por nombre (misma empresa).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'crm_products_hub'
          AND column_name = 'category'
    ) THEN
        UPDATE crm_products_hub ph
        SET category_id = c.id
        FROM crm_categories c
        WHERE c.company_id = ph.company_id
          AND ph.category IS NOT NULL
          AND btrim(ph.category::text) <> ''
          AND upper(btrim(c.name)) = upper(btrim(ph.category::text))
          AND ph.category_id IS NULL;
    END IF;
END $$;

DROP VIEW IF EXISTS v_crm_ai_analytics CASCADE;

CREATE VIEW v_crm_ai_analytics AS
SELECT
    sh.company_id,
    sh.sale_date::date AS fecha,
    sh.customer_name AS cliente_nombre,
    lc.name::varchar AS ciudad,
    ph.product_name AS producto,
    cat.name::varchar AS categoria,
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
LEFT JOIN crm_categories cat ON cat.id = ph.category_id
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

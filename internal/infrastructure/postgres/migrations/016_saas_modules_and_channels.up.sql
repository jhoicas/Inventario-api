-- ==============================================================================
-- 016 - SaaS Modules, Sales Channels, Product Analytics & Weighted Average Cost
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 1. Módulos SaaS por empresa (activación/desactivación instantánea)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS company_modules (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    module_name  VARCHAR(50)  NOT NULL
                    CHECK (module_name IN ('inventory', 'billing', 'crm', 'analytics', 'purchasing')),
    is_active    BOOLEAN      NOT NULL DEFAULT true,
    activated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ,                          -- NULL = sin vencimiento
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, module_name)                   -- 1 registro por módulo por empresa
);

CREATE INDEX IF NOT EXISTS idx_company_modules_company    ON company_modules (company_id);
CREATE INDEX IF NOT EXISTS idx_company_modules_active     ON company_modules (company_id, is_active);
CREATE INDEX IF NOT EXISTS idx_company_modules_expires    ON company_modules (expires_at)
    WHERE expires_at IS NOT NULL;

-- ------------------------------------------------------------------------------
-- 2. Canales de venta por empresa
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sales_channels (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name            VARCHAR(100)  NOT NULL,             -- Ej: 'Shopify', 'POS', 'B2B', 'WhatsApp'
    channel_type    VARCHAR(30)   DEFAULT 'other'
                        CHECK (channel_type IN ('ecommerce', 'pos', 'b2b', 'marketplace', 'other')),
    commission_rate DECIMAL(5,2)  NOT NULL DEFAULT 0    -- Porcentaje de comisión (ej: 3.50 = 3.5%)
                        CHECK (commission_rate >= 0 AND commission_rate <= 100),
    is_active       BOOLEAN       NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_sales_channels_company ON sales_channels (company_id);
CREATE INDEX IF NOT EXISTS idx_sales_channels_active  ON sales_channels (company_id, is_active);

-- ------------------------------------------------------------------------------
-- 3. Facturas: añadir channel_id (nullable; no toda factura tiene canal)
-- ------------------------------------------------------------------------------
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES sales_channels(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_invoices_channel_id ON invoices (channel_id)
    WHERE channel_id IS NOT NULL;

-- ------------------------------------------------------------------------------
-- 4. Productos: añadir cogs y reorder_point
--    cogs          = Costo de Bienes Vendidos acumulado (se actualiza con movimientos OUT)
--    reorder_point = Punto de reorden (umbral mínimo para disparar compras)
-- ------------------------------------------------------------------------------
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS cogs          DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reorder_point DECIMAL(15,4) NOT NULL DEFAULT 0;

-- ------------------------------------------------------------------------------
-- 5. Función y trigger: Costo Promedio Ponderado en inventory_movements
--    Se dispara AFTER INSERT para tipo 'IN'.
--    Fórmula: costo_nuevo = SUM(total_cost_IN) / SUM(quantity_IN)
--    Calcula el promedio sobre todos los movimientos IN históricos del producto,
--    lo que hace el recálculo idempotente y resistente a correcciones.
-- ------------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION actualizar_costo_promedio()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_total_qty  DECIMAL(15,4);
    v_total_cost DECIMAL(15,4);
    v_new_cost   DECIMAL(15,4);
BEGIN
    -- Solo aplica a entradas (IN) con cantidad positiva
    IF NEW.type <> 'IN' OR NEW.quantity <= 0 THEN
        RETURN NEW;
    END IF;

    -- Sumar toda la historia de movimientos IN para este producto (incluye el nuevo row)
    SELECT
        COALESCE(SUM(quantity),   0),
        COALESCE(SUM(total_cost), 0)
      INTO v_total_qty, v_total_cost
      FROM inventory_movements
     WHERE product_id = NEW.product_id
       AND type = 'IN';

    IF v_total_qty > 0 THEN
        v_new_cost := v_total_cost / v_total_qty;

        UPDATE products
           SET cost       = ROUND(v_new_cost, 4),
               updated_at = now()
         WHERE id = NEW.product_id;
    END IF;

    RETURN NEW;
END;
$$;

-- Trigger: se ejecuta después de cada INSERT en inventory_movements
DROP TRIGGER IF EXISTS trg_actualizar_costo_promedio ON inventory_movements;
CREATE TRIGGER trg_actualizar_costo_promedio
    AFTER INSERT ON inventory_movements
    FOR EACH ROW
    EXECUTE FUNCTION actualizar_costo_promedio();

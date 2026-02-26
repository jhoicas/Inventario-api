-- ==============================================================================
-- 016 - Rollback: SaaS Modules, Sales Channels, Analytics Fields & Trigger
-- ==============================================================================

-- 5. Eliminar trigger y función de costo promedio
DROP TRIGGER IF EXISTS trg_actualizar_costo_promedio ON inventory_movements;
DROP FUNCTION IF EXISTS actualizar_costo_promedio();

-- 4. Quitar columnas analíticas de productos
ALTER TABLE products
    DROP COLUMN IF EXISTS cogs,
    DROP COLUMN IF EXISTS reorder_point;

-- 3. Quitar channel_id de facturas
DROP INDEX  IF EXISTS idx_invoices_channel_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS channel_id;

-- 2. Eliminar canales de venta
DROP INDEX  IF EXISTS idx_sales_channels_active;
DROP INDEX  IF EXISTS idx_sales_channels_company;
DROP TABLE  IF EXISTS sales_channels;

-- 1. Eliminar módulos SaaS
DROP INDEX  IF EXISTS idx_company_modules_expires;
DROP INDEX  IF EXISTS idx_company_modules_active;
DROP INDEX  IF EXISTS idx_company_modules_company;
DROP TABLE  IF EXISTS company_modules;

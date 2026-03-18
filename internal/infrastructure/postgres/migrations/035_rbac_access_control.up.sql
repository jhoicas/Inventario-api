-- 035 - RBAC dinámico para menús y rutas

CREATE TABLE IF NOT EXISTS roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    description TEXT,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS modules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    icon       VARCHAR(100) NOT NULL,
    "order"    INTEGER      NOT NULL DEFAULT 0,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS screens (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id      UUID         NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    key            VARCHAR(80)  NOT NULL UNIQUE,
    name           VARCHAR(100) NOT NULL,
    frontend_route VARCHAR(255) NOT NULL,
    api_endpoint   VARCHAR(255) NOT NULL,
    "order"        INTEGER      NOT NULL DEFAULT 0,
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (frontend_route),
    UNIQUE (api_endpoint)
);

CREATE TABLE IF NOT EXISTS role_screens (
    role_id    UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    screen_id  UUID        NOT NULL REFERENCES screens(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, screen_id)
);

CREATE INDEX IF NOT EXISTS idx_roles_active ON roles (is_active);
CREATE INDEX IF NOT EXISTS idx_modules_active ON modules (is_active);
CREATE INDEX IF NOT EXISTS idx_modules_order ON modules ("order");
CREATE INDEX IF NOT EXISTS idx_screens_module_id ON screens (module_id);
CREATE INDEX IF NOT EXISTS idx_screens_active ON screens (is_active);
CREATE INDEX IF NOT EXISTS idx_screens_order ON screens (module_id, "order");
CREATE INDEX IF NOT EXISTS idx_role_screens_role_id ON role_screens (role_id);
CREATE INDEX IF NOT EXISTS idx_role_screens_screen_id ON role_screens (screen_id);

INSERT INTO roles (key, name, description, is_active)
VALUES
    ('admin', 'Administrador', 'Acceso total al sistema', true),
    ('bodeguero', 'Bodeguero', 'Gestión de inventario', true),
    ('vendedor', 'Vendedor', 'Gestión comercial y facturación', true),
    ('marketing', 'Marketing', 'Gestión de campañas y CRM', true),
    ('sales', 'Ventas', 'Ventas y CRM operativo', true),
    ('support', 'Soporte', 'Atención y seguimiento de clientes', true)
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = EXCLUDED.is_active,
    updated_at = now();

INSERT INTO modules (key, name, icon, "order", is_active)
VALUES
    ('inventory', 'Inventario', 'boxes', 1, true),
    ('billing', 'Facturación', 'receipt-text', 2, true),
    ('crm', 'CRM', 'users', 3, true)
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name,
    icon = EXCLUDED.icon,
    "order" = EXCLUDED."order",
    is_active = EXCLUDED.is_active,
    updated_at = now();

WITH inventory_module AS (
    SELECT id FROM modules WHERE key = 'inventory'
),
billing_module AS (
    SELECT id FROM modules WHERE key = 'billing'
),
crm_module AS (
    SELECT id FROM modules WHERE key = 'crm'
)
INSERT INTO screens (module_id, key, name, frontend_route, api_endpoint, "order", is_active)
SELECT inventory_module.id, 'inventory.products', 'Productos', '/inventory/products', '/api/products', 1, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.warehouses', 'Bodegas', '/inventory/warehouses', '/api/warehouses', 2, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.suppliers', 'Proveedores', '/inventory/suppliers', '/api/suppliers', 3, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.purchase-orders', 'Órdenes de compra', '/inventory/purchase-orders', '/api/purchase-orders', 4, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.movements', 'Movimientos', '/inventory/movements', '/api/inventory/movements', 5, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.stock', 'Stock', '/inventory/stock', '/api/inventory/stock', 6, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.replenishment', 'Reposición', '/inventory/replenishment', '/api/inventory/replenishment-list', 7, true FROM inventory_module
UNION ALL
SELECT inventory_module.id, 'inventory.stocktake', 'Conteo físico', '/inventory/stocktake', '/api/inventory/stocktake', 8, true FROM inventory_module
UNION ALL
SELECT billing_module.id, 'billing.customers', 'Clientes', '/billing/customers', '/api/customers', 1, true FROM billing_module
UNION ALL
SELECT billing_module.id, 'billing.invoices', 'Facturas', '/billing/invoices', '/api/invoices', 2, true FROM billing_module
UNION ALL
SELECT billing_module.id, 'billing.resolutions', 'Resoluciones', '/billing/resolutions', '/api/resolutions', 3, true FROM billing_module
UNION ALL
SELECT billing_module.id, 'billing.dian-settings', 'Configuración DIAN', '/billing/settings/dian', '/api/settings/dian', 4, true FROM billing_module
UNION ALL
SELECT billing_module.id, 'billing.emails', 'Correos', '/billing/emails', '/api/emails/send', 5, true FROM billing_module
UNION ALL
SELECT billing_module.id, 'billing.summary', 'Resumen DIAN', '/billing/dian/summary', '/api/billing/dian/summary', 6, true FROM billing_module
UNION ALL
SELECT crm_module.id, 'crm.customers', 'Clientes CRM', '/crm/customers', '/api/crm/customers', 1, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.categories', 'Categorías', '/crm/categories', '/api/crm/categories', 2, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.benefits', 'Beneficios', '/crm/benefits', '/api/crm/benefits', 3, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.campaigns', 'Campañas', '/crm/campaigns', '/api/crm/campaigns', 4, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.campaign-templates', 'Plantillas', '/crm/campaign-templates', '/api/crm/campaign-templates', 5, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.tasks', 'Tareas', '/crm/tasks', '/api/crm/tasks', 6, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.tickets', 'Tickets', '/crm/tickets', '/api/crm/tickets', 7, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.opportunities', 'Oportunidades', '/crm/opportunities', '/api/crm/opportunities', 8, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.interactions', 'Interacciones', '/crm/interactions', '/api/crm/interactions', 9, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.loyalty', 'Lealtad', '/crm/loyalty', '/api/crm/loyalty', 10, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.ai', 'IA CRM', '/crm/ai', '/api/crm/ai', 11, true FROM crm_module
ON CONFLICT (key) DO UPDATE
SET module_id = EXCLUDED.module_id,
    name = EXCLUDED.name,
    frontend_route = EXCLUDED.frontend_route,
    api_endpoint = EXCLUDED.api_endpoint,
    "order" = EXCLUDED."order",
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- Admin: acceso total a todas las pantallas.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON TRUE
WHERE r.key = 'admin'
ON CONFLICT DO NOTHING;

-- Inventario: bodeguero.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON s.key IN (
    'inventory.products',
    'inventory.warehouses',
    'inventory.suppliers',
    'inventory.purchase-orders',
    'inventory.movements',
    'inventory.stock',
    'inventory.replenishment',
    'inventory.stocktake'
)
WHERE r.key = 'bodeguero'
ON CONFLICT DO NOTHING;

-- Facturación: vendedor.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON s.key IN (
    'billing.customers',
    'billing.invoices',
    'billing.resolutions',
    'billing.dian-settings',
    'billing.emails',
    'billing.summary'
)
WHERE r.key = 'vendedor'
ON CONFLICT DO NOTHING;

-- Ventas: billing + CRM operativo.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON s.key IN (
    'billing.customers',
    'billing.invoices',
    'billing.summary',
    'crm.customers',
    'crm.opportunities',
    'crm.tasks',
    'crm.interactions',
    'crm.loyalty',
    'crm.ai'
)
WHERE r.key = 'sales'
ON CONFLICT DO NOTHING;

-- Marketing: CRM comercial.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON s.key IN (
    'crm.customers',
    'crm.categories',
    'crm.benefits',
    'crm.campaigns',
    'crm.campaign-templates',
    'crm.tasks',
    'crm.interactions',
    'crm.ai'
)
WHERE r.key = 'marketing'
ON CONFLICT DO NOTHING;

-- Soporte: CRM atención al cliente.
INSERT INTO role_screens (role_id, screen_id)
SELECT r.id, s.id
FROM roles r
JOIN screens s ON s.key IN (
    'crm.customers',
    'crm.tickets',
    'crm.interactions',
    'crm.loyalty',
    'crm.ai'
)
WHERE r.key = 'support'
ON CONFLICT DO NOTHING;


-- 041_rbac_crm_frontend_routes.up.sql
-- Registra nuevas pantallas CRM requeridas por frontend en catálogo RBAC.

WITH crm_module AS (
    SELECT id
    FROM modules
    WHERE key = 'crm'
)
INSERT INTO screens (module_id, key, name, frontend_route, api_endpoint, "order", is_active)
SELECT crm_module.id, 'crm.customers', 'Clientes CRM', '/crm/customers', '/api/crm/customers', 1, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.campaigns-recipients-resolve', 'Resolver destinatarios', '/crm/campaigns/recipients-resolve', '/api/crm/campaigns/recipients/resolve', 12, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.analytics-kpis', 'KPIs CRM', '/crm/analytics/kpis', '/api/crm/analytics/kpis', 13, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.analytics-segmentation', 'Segmentacion CRM', '/crm/analytics/segmentation', '/api/crm/analytics/segmentation', 14, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.analytics-monthly-evolution', 'Evolucion mensual CRM', '/crm/analytics/monthly-evolution', '/api/crm/analytics/monthly-evolution', 15, true FROM crm_module
ON CONFLICT (key) DO UPDATE
SET module_id = EXCLUDED.module_id,
    name = EXCLUDED.name,
    frontend_route = EXCLUDED.frontend_route,
    api_endpoint = EXCLUDED.api_endpoint,
    "order" = EXCLUDED."order",
    is_active = EXCLUDED.is_active,
    updated_at = now();

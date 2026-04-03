-- 041_rbac_crm_frontend_routes.down.sql

DELETE FROM role_screens
WHERE screen_id IN (
    SELECT id
    FROM screens
    WHERE key IN (
        'crm.campaigns-recipients-resolve',
        'crm.analytics-kpis',
        'crm.analytics-segmentation',
        'crm.analytics-monthly-evolution'
    )
);

DELETE FROM screens
WHERE key IN (
    'crm.campaigns-recipients-resolve',
    'crm.analytics-kpis',
    'crm.analytics-segmentation',
    'crm.analytics-monthly-evolution'
);

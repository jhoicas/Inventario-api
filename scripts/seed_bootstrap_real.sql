-- Bootstrap inicial (realista) para entorno limpio
-- Objetivo: iniciar sesión + módulos/menús activos + diccionarios base
-- Usuario admin: admin@empresa.com / Temporal$1

BEGIN;

-- Requerido para gen_random_uuid() y crypt()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- -----------------------------------------------------------------------------
-- 1) Empresa base
-- -----------------------------------------------------------------------------
INSERT INTO companies (
    id, name, nit, address, phone, email, status, created_at, updated_at
) VALUES (
    '55555555-0000-0000-0000-000000000010'::uuid,
    'Empresa Comercial Colombia S.A.S.',
    '900123456-7',
    'Calle 26 # 92-32, Bogotá D.C.',
    '6015800000',
    'contacto@empresa.com',
    'active',
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    nit = EXCLUDED.nit,
    address = EXCLUDED.address,
    phone = EXCLUDED.phone,
    email = EXCLUDED.email,
    status = EXCLUDED.status,
    updated_at = now();

-- -----------------------------------------------------------------------------
-- 2) Usuario administrador
--    Email: admin@empresa.com
--    Password: Temporal$1
-- -----------------------------------------------------------------------------
INSERT INTO users (
    id, company_id, email, password_hash, name, roles, status, created_at, updated_at
) VALUES (
    '36d47081-2235-45d5-b570-0736a7ae2493'::uuid,
    '55555555-0000-0000-0000-000000000010'::uuid,
    'admin@empresa.com',
    crypt('Temporal$1', gen_salt('bf', 10)),
    'Administrador General',
    ARRAY['admin']::text[],
    'active',
    now(),
    now()
)
ON CONFLICT (company_id, email) DO UPDATE SET
    password_hash = crypt('Temporal$1', gen_salt('bf', 10)),
    name = EXCLUDED.name,
    roles = ARRAY['admin']::text[],
    status = 'active',
    updated_at = now();

-- -----------------------------------------------------------------------------
-- 3) Módulos SaaS activos (menús)
-- -----------------------------------------------------------------------------
INSERT INTO company_modules (
    id, company_id, module_name, is_active, activated_at, expires_at, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    '55555555-0000-0000-0000-000000000010'::uuid,
    m.module_name,
    true,
    now(),
    NULL,
    now(),
    now()
FROM (
    VALUES
        ('inventory'),
        ('billing'),
        ('crm'),
        ('analytics'),
        ('purchasing')
) AS m(module_name)
ON CONFLICT (company_id, module_name) DO UPDATE SET
    is_active = true,
    activated_at = now(),
    expires_at = NULL,
    updated_at = now();

-- -----------------------------------------------------------------------------
-- 4) Diccionario: tipos de identificación DIAN
-- -----------------------------------------------------------------------------
INSERT INTO identification_types (code, name)
VALUES
    ('13', 'Cédula de Ciudadanía'),
    ('21', 'Tarjeta de Extranjería'),
    ('22', 'Cédula de Extranjería'),
    ('31', 'NIT'),
    ('41', 'Pasaporte'),
    ('33', 'Identificación Extranjeros Diferente a NIT Asignado DIAN'),
    ('42', 'Documento de Identificación Extranjero'),
    ('43', 'Sin Identificación del Exterior o para Uso Definido DIAN'),
    ('44', 'Documento de Identificación Extranjeros Persona Jurídica')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name;

-- -----------------------------------------------------------------------------
-- 5) Diccionario: unidades de medida (mínimo útil)
-- -----------------------------------------------------------------------------
INSERT INTO measurement_units (code, name)
VALUES
    ('94', 'Unidad'),
    ('KGM', 'Kilogramo'),
    ('LTR', 'Litro'),
    ('MTR', 'Metro')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name;

-- -----------------------------------------------------------------------------
-- 6) Diccionario: responsabilidades fiscales (mínimo útil)
-- -----------------------------------------------------------------------------
INSERT INTO fiscal_responsibilities (code, name)
VALUES
    ('R-99-PN', 'No aplica - Otros'),
    ('O-13', 'Gran contribuyente'),
    ('O-15', 'Autorretenedor'),
    ('O-47', 'Régimen simple de tributación')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name;

-- -----------------------------------------------------------------------------
-- 7) Diccionario: impuestos básicos
-- -----------------------------------------------------------------------------
INSERT INTO taxes (id, code, name, description, is_retention, rate)
VALUES
    (gen_random_uuid(), '01', 'IVA 19%', 'Impuesto al valor agregado 19%', false, 19.00),
    (gen_random_uuid(), '02', 'IVA 5%',  'Impuesto al valor agregado 5%',  false, 5.00),
    (gen_random_uuid(), '03', 'IVA 0%',  'Exento / excluido',              false, 0.00)
ON CONFLICT DO NOTHING;

-- -----------------------------------------------------------------------------
-- 8) Ubicaciones mínimas (Bogotá) para formularios
-- -----------------------------------------------------------------------------
INSERT INTO locations_departments (id, code, name)
VALUES
    ('11000000-0000-0000-0000-000000000001'::uuid, '11', 'Bogotá D.C.')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name;

INSERT INTO locations_cities (id, department_id, code, name)
VALUES
    (
        '11000000-0000-0000-0000-000000000010'::uuid,
        (SELECT id FROM locations_departments WHERE code = '11'),
        '11001',
        'Bogotá D.C.'
    )
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    department_id = EXCLUDED.department_id;

-- -----------------------------------------------------------------------------
-- 9) Canales de venta mínimos (útil para analytics)
-- -----------------------------------------------------------------------------
INSERT INTO sales_channels (
    id, company_id, name, channel_type, commission_rate, is_active, created_at, updated_at
)
VALUES
    (gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, 'Ecommerce', 'ecommerce', 3.50, true, now(), now()),
    (gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, 'Punto de Venta', 'pos', 1.20, true, now(), now())
ON CONFLICT (company_id, name) DO UPDATE SET
    channel_type = EXCLUDED.channel_type,
    commission_rate = EXCLUDED.commission_rate,
    is_active = true,
    updated_at = now();

-- -----------------------------------------------------------------------------
-- 10) Bodega inicial
-- -----------------------------------------------------------------------------
INSERT INTO warehouses (id, company_id, name, address, created_at, updated_at)
VALUES (
    '77000000-0000-0000-0000-000000000111'::uuid,
    '55555555-0000-0000-0000-000000000010'::uuid,
    'Bodega Principal',
    'Calle 26 # 92-32, Bogotá D.C.',
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    address = EXCLUDED.address,
    updated_at = now();

COMMIT;

-- Verificación rápida:
-- SELECT id, name, nit FROM companies WHERE id='55555555-0000-0000-0000-000000000010';
-- SELECT email, roles, status FROM users WHERE email='admin@empresa.com';
-- SELECT module_name, is_active FROM company_modules WHERE company_id='55555555-0000-0000-0000-000000000010' ORDER BY module_name;

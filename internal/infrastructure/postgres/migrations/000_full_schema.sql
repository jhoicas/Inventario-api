-- ==============================================================================
-- ESQUEMA COMPLETO EN ORDEN (InvoryaBack + Facturación Electrónica DIAN)
-- Ejecutar sobre una base vacía o usar migraciones individuales para actualizar.
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 001 - Companies
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS companies (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,
    nit        VARCHAR(20)  NOT NULL UNIQUE,
    address    TEXT,
    phone      VARCHAR(50),
    email      VARCHAR(255),
    status     VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'inactive')),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_companies_nit ON companies (nit);
CREATE INDEX IF NOT EXISTS idx_companies_status ON companies (status);

-- ------------------------------------------------------------------------------
-- 002 - Users
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(200) NOT NULL,
    role          VARCHAR(20)  NOT NULL CHECK (role IN ('admin', 'bodeguero', 'vendedor')),
    status        VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, email)
);
CREATE INDEX IF NOT EXISTS idx_users_company_id ON users (company_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- ------------------------------------------------------------------------------
-- 003 - Warehouses
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS warehouses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    address    TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_warehouses_company_id ON warehouses (company_id);

-- ------------------------------------------------------------------------------
-- 004 - Products
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS products (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    sku           VARCHAR(100) NOT NULL,
    name          VARCHAR(200) NOT NULL,
    description   TEXT,
    price         DECIMAL(15,2) NOT NULL DEFAULT 0,
    cost          DECIMAL(15,2) NOT NULL DEFAULT 0,
    tax_rate      DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate IN (0, 5, 19)),
    unspsc_code   VARCHAR(50),
    unit_measure  VARCHAR(10) NOT NULL DEFAULT '94',
    attributes    JSONB,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, sku)
);
CREATE INDEX IF NOT EXISTS idx_products_company_id ON products (company_id);
CREATE INDEX IF NOT EXISTS idx_products_sku ON products (sku);
CREATE INDEX IF NOT EXISTS idx_products_company_sku ON products (company_id, sku);

-- ------------------------------------------------------------------------------
-- 005 - Stock
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS stock (
    product_id   UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID         NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    quantity     DECIMAL(15,4) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (product_id, warehouse_id)
);
CREATE INDEX IF NOT EXISTS idx_stock_warehouse_id ON stock (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_product_id ON stock (product_id);

-- ------------------------------------------------------------------------------
-- 006 - Inventory movements
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS inventory_movements (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID         NOT NULL,
    product_id     UUID         NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    warehouse_id   UUID         NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    type           VARCHAR(20)  NOT NULL CHECK (type IN ('IN', 'OUT', 'ADJUSTMENT', 'TRANSFER')),
    quantity       DECIMAL(15,4) NOT NULL,
    unit_cost      DECIMAL(15,4) NOT NULL DEFAULT 0,
    total_cost     DECIMAL(15,4) NOT NULL DEFAULT 0,
    date           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by     UUID         REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_transaction_id ON inventory_movements (transaction_id);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_product_id ON inventory_movements (product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_warehouse_id ON inventory_movements (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_date ON inventory_movements (date);

-- ------------------------------------------------------------------------------
-- 007 - Customers
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    tax_id     VARCHAR(50)  NOT NULL,
    email      VARCHAR(255),
    phone      VARCHAR(50),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, tax_id)
);
CREATE INDEX IF NOT EXISTS idx_customers_company_id ON customers (company_id);
CREATE INDEX IF NOT EXISTS idx_customers_tax_id ON customers (company_id, tax_id);

-- ------------------------------------------------------------------------------
-- 008 - Invoices
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoices (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id  UUID         NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    prefix       VARCHAR(20)  NOT NULL,
    number       VARCHAR(50)  NOT NULL,
    date         DATE         NOT NULL DEFAULT current_date,
    net_total    DECIMAL(15,2) NOT NULL DEFAULT 0,
    tax_total    DECIMAL(15,2) NOT NULL DEFAULT 0,
    grand_total  DECIMAL(15,2) NOT NULL DEFAULT 0,
    dian_status  VARCHAR(20)  NOT NULL DEFAULT 'Pending' CHECK (dian_status IN ('Pending', 'Sent', 'Error')),
    cufe         VARCHAR(255),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, prefix, number)
);
CREATE INDEX IF NOT EXISTS idx_invoices_company_id ON invoices (company_id);
CREATE INDEX IF NOT EXISTS idx_invoices_customer_id ON invoices (customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_date ON invoices (date);

CREATE TABLE IF NOT EXISTS invoice_details (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID         NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id UUID         NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity   DECIMAL(15,4) NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(15,2) NOT NULL,
    tax_rate   DECIMAL(5,2) NOT NULL DEFAULT 0,
    subtotal   DECIMAL(15,2) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invoice_details_invoice_id ON invoice_details (invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_details_product_id ON invoice_details (product_id);

-- ------------------------------------------------------------------------------
-- 009 - Updates DIAN (tablas paramétricas, resoluciones, columnas fiscales)
-- ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS identification_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS locations_departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS locations_cities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID NOT NULL REFERENCES locations_departments(id),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS measurement_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS fiscal_responsibilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL
);

CREATE TABLE IF NOT EXISTS billing_resolutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    resolution_number VARCHAR(50) NOT NULL,
    prefix VARCHAR(10) NOT NULL,
    technical_key TEXT NOT NULL,
    range_from INT NOT NULL,
    range_to INT NOT NULL,
    current_number INT NOT NULL,
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'depleted')),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_billing_resolutions_company ON billing_resolutions(company_id);

ALTER TABLE companies
ADD COLUMN IF NOT EXISTS identification_type_id UUID REFERENCES identification_types(id),
ADD COLUMN IF NOT EXISTS city_id UUID REFERENCES locations_cities(id),
ADD COLUMN IF NOT EXISTS legal_organization_type VARCHAR(20) DEFAULT 'company' CHECK (legal_organization_type IN ('person', 'company')),
ADD COLUMN IF NOT EXISTS tribute_name VARCHAR(10) DEFAULT '01',
ADD COLUMN IF NOT EXISTS economic_activity_code VARCHAR(10);

ALTER TABLE customers
ADD COLUMN IF NOT EXISTS identification_type_id UUID REFERENCES identification_types(id),
ADD COLUMN IF NOT EXISTS city_id UUID REFERENCES locations_cities(id),
ADD COLUMN IF NOT EXISTS first_name VARCHAR(100),
ADD COLUMN IF NOT EXISTS last_name VARCHAR(100),
ADD COLUMN IF NOT EXISTS commercial_name VARCHAR(200),
ADD COLUMN IF NOT EXISTS address TEXT,
ADD COLUMN IF NOT EXISTS email_billing VARCHAR(255);

ALTER TABLE invoices
ADD COLUMN IF NOT EXISTS resolution_id UUID REFERENCES billing_resolutions(id),
ADD COLUMN IF NOT EXISTS payment_form VARCHAR(20) DEFAULT '1',
ADD COLUMN IF NOT EXISTS payment_method_code VARCHAR(10) DEFAULT '10',
ADD COLUMN IF NOT EXISTS due_date DATE,
ADD COLUMN IF NOT EXISTS notes TEXT,
ADD COLUMN IF NOT EXISTS xml_url TEXT,
ADD COLUMN IF NOT EXISTS qr_data TEXT,
ADD COLUMN IF NOT EXISTS issue_date TIMESTAMPTZ DEFAULT now();

CREATE TABLE IF NOT EXISTS taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_retention BOOLEAN DEFAULT false,
    rate DECIMAL(5,2) NOT NULL
);

CREATE TABLE IF NOT EXISTS invoice_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    tax_id UUID REFERENCES taxes(id),
    tax_code VARCHAR(10) NOT NULL,
    base_amount DECIMAL(15,2) NOT NULL,
    rate DECIMAL(5,2) NOT NULL,
    amount DECIMAL(15,2) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invoice_taxes_invoice ON invoice_taxes(invoice_id);

ALTER TABLE products
DROP COLUMN IF EXISTS unit_measure;
ALTER TABLE products ADD COLUMN IF NOT EXISTS measurement_unit_id UUID REFERENCES measurement_units(id);
ALTER TABLE products ADD COLUMN IF NOT EXISTS unit_measure VARCHAR(10) NOT NULL DEFAULT '94';

-- ------------------------------------------------------------------------------
-- 010 - Seed tipos de identificación (DIAN)
-- ------------------------------------------------------------------------------
INSERT INTO identification_types (code, name) VALUES
('13', 'Cédula de Ciudadanía'),
('21', 'Tarjeta de Extranjería'),
('22', 'Cédula de Extranjería'),
('31', 'NIT'),
('41', 'Pasaporte'),
('33', 'Identificación Extranjeros Diferente a NIT Asignado DIAN'),
('42', 'Documento de Identificación Extranjero'),
('43', 'Sin Identificación del Exterior o para Uso Definido DIAN'),
('44', 'Documento de Identificación Extranjeros Persona Jurídica')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

-- ------------------------------------------------------------------------------
-- 011 - Seed ubicaciones (departamentos y municipios Colombia)
-- Ejecutar por separado si se desea: 011_seed_locations.sql (archivo grande)
-- ------------------------------------------------------------------------------
-- Incluir aquí o ejecutar: \i 011_seed_locations.sql

-- ------------------------------------------------------------------------------
-- 012 - Invoices: columna uuid (mismo valor que CUFE para XML DIAN)
-- ------------------------------------------------------------------------------
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS uuid TEXT;

-- ------------------------------------------------------------------------------
-- 013 - Invoices: contenido XML firmado
-- ------------------------------------------------------------------------------
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS xml_signed TEXT;

-- ------------------------------------------------------------------------------
-- 014 - Invoices: estados DIAN (DRAFT, SIGNED, ERROR_GENERATION)
-- ------------------------------------------------------------------------------
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_dian_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_dian_status_check
  CHECK (dian_status IN ('Pending', 'Sent', 'Error', 'DRAFT', 'SIGNED', 'ERROR_GENERATION'));

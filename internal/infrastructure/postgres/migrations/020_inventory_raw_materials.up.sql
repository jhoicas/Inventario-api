-- 020 - Materias primas, recetas y costos adicionales en facturas

-- 1. Materias primas
CREATE TABLE IF NOT EXISTS raw_materials (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name          VARCHAR(200) NOT NULL,
    sku           VARCHAR(100) NOT NULL,
    cost          DECIMAL(15,4) NOT NULL DEFAULT 0,
    unit_measure  VARCHAR(10)  NOT NULL DEFAULT '94',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_raw_materials_company_id ON raw_materials(company_id);

-- 2. Recetas / Bill of Materials: relación producto ↔ materia prima
CREATE TABLE IF NOT EXISTS bill_of_materials (
    product_id        UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    raw_material_id   UUID         NOT NULL REFERENCES raw_materials(id) ON DELETE CASCADE,
    quantity_required DECIMAL(15,4) NOT NULL CHECK (quantity_required > 0),
    waste_percentage  DECIMAL(5,4)  NOT NULL DEFAULT 0 CHECK (waste_percentage >= 0),
    PRIMARY KEY (product_id, raw_material_id)
);

-- 3. Costos adicionales en facturas
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS logistics_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount_total DECIMAL(15,2) NOT NULL DEFAULT 0;


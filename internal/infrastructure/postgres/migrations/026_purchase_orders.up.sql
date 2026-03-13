CREATE TABLE IF NOT EXISTS purchase_orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    supplier_id UUID         NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    number      VARCHAR(100) NOT NULL,
    date        DATE         NOT NULL,
    status      VARCHAR(30)  NOT NULL CHECK (status IN ('BORRADOR', 'ENVIADA', 'CONFIRMADA', 'RECIBIDA_PARCIAL', 'CERRADA')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (company_id, number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_company_id ON purchase_orders(company_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier_id ON purchase_orders(supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_status ON purchase_orders(status);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID          NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id        UUID          NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity          DECIMAL(15,4) NOT NULL CHECK (quantity > 0),
    unit_cost         DECIMAL(15,4) NOT NULL CHECK (unit_cost >= 0),
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (purchase_order_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_purchase_order_id ON purchase_order_items(purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_product_id ON purchase_order_items(product_id);

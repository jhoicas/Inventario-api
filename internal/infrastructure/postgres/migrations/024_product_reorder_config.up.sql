CREATE TABLE IF NOT EXISTS product_reorder_config (
    product_id      UUID            NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id    UUID            NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    reorder_point   DECIMAL(15,4)   NOT NULL DEFAULT 0 CHECK (reorder_point >= 0),
    min_stock       DECIMAL(15,4)   NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    max_stock       DECIMAL(15,4)   NOT NULL DEFAULT 0 CHECK (max_stock >= 0),
    lead_time_days  INTEGER         NOT NULL DEFAULT 0 CHECK (lead_time_days >= 0),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT uq_product_reorder_config_product_warehouse UNIQUE (product_id, warehouse_id),
    CONSTRAINT chk_product_reorder_config_min_max CHECK (max_stock >= min_stock)
);

CREATE INDEX IF NOT EXISTS idx_product_reorder_config_product_id
    ON product_reorder_config(product_id);

CREATE INDEX IF NOT EXISTS idx_product_reorder_config_warehouse_id
    ON product_reorder_config(warehouse_id);

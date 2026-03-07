-- 021 - CRM Operativo y Fidelización

-- 1. Categorías de fidelización (Oro, Plata, Bronce)
CREATE TABLE IF NOT EXISTS crm_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    min_ltv    DECIMAL(15,2),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_categories_company_id ON crm_categories(company_id);

-- 2. Beneficios por categoría
CREATE TABLE IF NOT EXISTS crm_benefits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id UUID         NOT NULL REFERENCES crm_categories(id) ON DELETE CASCADE,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_benefits_company_id ON crm_benefits(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_benefits_category_id ON crm_benefits(category_id);

-- 3. Perfil CRM del cliente (extiende customers con categoría y LTV)
CREATE TABLE IF NOT EXISTS crm_customer_profiles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID         NOT NULL UNIQUE REFERENCES customers(id) ON DELETE CASCADE,
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id UUID        REFERENCES crm_categories(id) ON DELETE SET NULL,
    ltv        DECIMAL(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_customer_profiles_company_id ON crm_customer_profiles(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_customer_profiles_customer_id ON crm_customer_profiles(customer_id);
CREATE INDEX IF NOT EXISTS idx_crm_customer_profiles_category_id ON crm_customer_profiles(category_id);

-- 4. Interacciones (llamadas, emails)
CREATE TABLE IF NOT EXISTS crm_interactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id UUID         NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    type        VARCHAR(50)  NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'other')),
    subject     VARCHAR(500),
    body        TEXT,
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_interactions_company_id ON crm_interactions(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_interactions_customer_id ON crm_interactions(customer_id);
CREATE INDEX IF NOT EXISTS idx_crm_interactions_created_at ON crm_interactions(created_at);

-- 5. Tareas
CREATE TABLE IF NOT EXISTS crm_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id UUID         REFERENCES customers(id) ON DELETE SET NULL,
    title       VARCHAR(300) NOT NULL,
    description TEXT,
    due_at      TIMESTAMPTZ,
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'done', 'cancelled')),
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_tasks_company_id ON crm_tasks(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_tasks_customer_id ON crm_tasks(customer_id);
CREATE INDEX IF NOT EXISTS idx_crm_tasks_status ON crm_tasks(status);
CREATE INDEX IF NOT EXISTS idx_crm_tasks_due_at ON crm_tasks(due_at);

-- 6. Tickets PQR (casos con análisis de sentimiento)
CREATE TABLE IF NOT EXISTS crm_tickets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id UUID         NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subject     VARCHAR(500) NOT NULL,
    description TEXT         NOT NULL,
    status      VARCHAR(50)  NOT NULL DEFAULT 'open',
    sentiment   VARCHAR(20)  CHECK (sentiment IN ('positive', 'neutral', 'negative')),
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_crm_tickets_company_id ON crm_tickets(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_tickets_customer_id ON crm_tickets(customer_id);
CREATE INDEX IF NOT EXISTS idx_crm_tickets_status ON crm_tickets(status);

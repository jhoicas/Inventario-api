-- ============================================================
-- Seed realista FE (analytics + invoices + loyalty)
-- Motor: PostgreSQL
-- Empresa: 55555555-0000-0000-0000-000000000010
-- Idempotente: sí (ON CONFLICT + claves naturales)
-- ============================================================

BEGIN;

-- 0) Empresa requerida
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM companies WHERE id = '55555555-0000-0000-0000-000000000010'::uuid
	) THEN
		RAISE EXCEPTION 'No existe company_id 55555555-0000-0000-0000-000000000010 en companies';
	END IF;
END $$;

-- 1) Asegurar check de estados DIAN incluyendo CONTINGENCIA/EXITOSO/RECHAZADO
DO $$
DECLARE c record;
BEGIN
	FOR c IN
		SELECT conname
		FROM pg_constraint
		WHERE conrelid = 'invoices'::regclass
			AND contype = 'c'
			AND pg_get_constraintdef(oid) ILIKE '%dian_status%'
	LOOP
		EXECUTE format('ALTER TABLE invoices DROP CONSTRAINT %I', c.conname);
	END LOOP;

	EXECUTE $sql$
		ALTER TABLE invoices
		ADD CONSTRAINT invoices_dian_status_check
		CHECK (dian_status IN (
			'Pending','Sent','Error','DRAFT','SIGNED','ERROR_GENERATION',
			'EXITOSO','RECHAZADO','CONTINGENCIA'
		))
	$sql$;
END $$;

-- 2) Canales (>=3)
INSERT INTO sales_channels (
	id, company_id, name, channel_type, commission_rate, is_active, created_at, updated_at
) VALUES
	('77000000-0000-0000-0000-000000000001','55555555-0000-0000-0000-000000000010','Ecommerce Web','ecommerce',3.50,true,NOW(),NOW()),
	('77000000-0000-0000-0000-000000000002','55555555-0000-0000-0000-000000000010','POS Tienda Física','pos',1.20,true,NOW(),NOW()),
	('77000000-0000-0000-0000-000000000003','55555555-0000-0000-0000-000000000010','Canal B2B','b2b',2.80,true,NOW(),NOW())
ON CONFLICT (company_id, name) DO UPDATE SET
	channel_type = EXCLUDED.channel_type,
	commission_rate = EXCLUDED.commission_rate,
	is_active = EXCLUDED.is_active,
	updated_at = NOW();

CREATE TEMP TABLE tmp_seed_channels (
	name text PRIMARY KEY,
	channel_id uuid NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_seed_channels(name, channel_id)
SELECT name, id
FROM sales_channels
WHERE company_id = '55555555-0000-0000-0000-000000000010'::uuid
	AND name IN ('Ecommerce Web', 'POS Tienda Física', 'Canal B2B')
ON CONFLICT (name) DO UPDATE SET channel_id = EXCLUDED.channel_id;

-- 3) Clientes seed (por clave natural company_id + tax_id)
WITH seed_customers AS (
	SELECT * FROM (VALUES
		('María Fernanda Ospina','1098765432','maria.ospina@demo.com','3001111111'),
		('Andrés Felipe Castaño','1144556677','andres.castano@demo.com','3002222222'),
		('Valentina Muñoz Ríos','1007788990','valentina.munoz@demo.com','3003333333'),
		('Gym ProFit SAS','900123456','compras@gymprofit.co','6023456789'),
		('Nutrición Express Ltda','900234567','pedidos@nutricionexpress.co','6024567890'),
		('Camila Díaz Montoya','1130998877','camila.diaz@demo.com','3006666666'),
		('PowerFit Gym SAS','900456789','compras@powerfit.co','6026789012'),
		('Isabella Correa Hurtado','1007223344','isabella.correa@demo.com','3008888888')
	) AS t(name, tax_id, email, phone)
)
INSERT INTO customers (id, company_id, name, tax_id, email, phone, created_at, updated_at)
SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, name, tax_id, email, phone, NOW(), NOW()
FROM seed_customers
ON CONFLICT (company_id, tax_id) DO UPDATE SET
	name = EXCLUDED.name,
	email = EXCLUDED.email,
	phone = EXCLUDED.phone,
	updated_at = NOW();

CREATE TEMP TABLE tmp_seed_customers (
	tax_id text PRIMARY KEY,
	customer_id uuid NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_seed_customers(tax_id, customer_id)
SELECT c.tax_id, c.id
FROM customers c
WHERE c.company_id = '55555555-0000-0000-0000-000000000010'::uuid
	AND c.tax_id IN (
		'1098765432','1144556677','1007788990','900123456',
		'900234567','1130998877','900456789','1007223344'
	)
ON CONFLICT (tax_id) DO UPDATE SET customer_id = EXCLUDED.customer_id;

-- 4) Productos (24 SKUs)
WITH seed_products AS (
	SELECT * FROM (VALUES
		('MAG-001','Magnesio Quelado 400mg',45000::numeric,22000::numeric),
		('OMG-001','Omega 3 Fish Oil 1000mg',58000,28000),
		('COL-001','Colágeno Hidrolizado + Vit C',72000,35000),
		('MUL-001','Multivitamínico Completo',38000,18000),
		('PRO-001','Probiótico 10 Cepas',65000,32000),
		('VTD-001','Vitamina D3 5000 UI',28000,12000),
		('CUR-001','Cúrcuma + Pimienta Negra',42000,20000),
		('ZNC-001','Zinc + Vitamina C',32000,15000),
		('ARG-001','Aceite de Argán Puro',55000,25000),
		('SER-001','Sérum Vitamina C 20%',78000,38000),
		('JAB-001','Jabón de Cúrcuma Artesanal',18000,8000),
		('MAS-001','Mascarilla de Arcilla Verde',35000,16000),
		('SHA-001','Shampoo Romero Natural',42000,19000),
		('CRE-001','Crema Corporal de Cacao',38000,17000),
		('TE-001','Té Verde Premium',48000,22000),
		('GRA-001','Granola Artesanal',28000,13000),
		('SPI-001','Spirulina en Polvo',52000,25000),
		('CHI-001','Semillas de Chía',22000,10000),
		('MIE-001','Miel Raw',35000,16000),
		('CAC-001','Cacao en Polvo Orgánico',32000,14000),
		('HAR-001','Harina de Almendra',38000,18000),
		('SNK-001','Snack Mix Frutos Secos',25000,12000),
		('WHY-001','Proteína Whey Chocolate',145000,72000),
		('CRE-002','Creatina Monohidratada',85000,42000)
	) AS t(sku, name, price, cost)
)
INSERT INTO products (
	id, company_id, sku, name, description,
	price, cost, tax_rate, unit_measure, cogs, reorder_point, created_at, updated_at
)
SELECT
	gen_random_uuid(),
	'55555555-0000-0000-0000-000000000010'::uuid,
	sku,
	name,
	'Seed realista frontend',
	price,
	cost,
	19,
	'Unidad',
	cost,
	20,
	NOW(), NOW()
FROM seed_products
ON CONFLICT (company_id, sku) DO UPDATE SET
	name = EXCLUDED.name,
	price = EXCLUDED.price,
	cost = EXCLUDED.cost,
	tax_rate = EXCLUDED.tax_rate,
	updated_at = NOW();

CREATE TEMP TABLE tmp_seed_products (
	sku text PRIMARY KEY,
	product_id uuid NOT NULL,
	price numeric NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_seed_products(sku, product_id, price)
SELECT p.sku, p.id, p.price
FROM products p
WHERE p.company_id = '55555555-0000-0000-0000-000000000010'::uuid
	AND p.sku IN (
		'MAG-001','OMG-001','COL-001','MUL-001','PRO-001','VTD-001','CUR-001','ZNC-001',
		'ARG-001','SER-001','JAB-001','MAS-001','SHA-001','CRE-001','TE-001','GRA-001',
		'SPI-001','CHI-001','MIE-001','CAC-001','HAR-001','SNK-001','WHY-001','CRE-002'
	)
ON CONFLICT (sku) DO UPDATE SET product_id = EXCLUDED.product_id, price = EXCLUDED.price;

-- 5) Facturas últimos 6 meses con estados mixtos + hoy
WITH seed_invoices AS (
	SELECT * FROM (VALUES
		('SFD','1001', CURRENT_DATE - INTERVAL '175 days','Sent',         '1098765432', 'Ecommerce Web'),
		('SFD','1002', CURRENT_DATE - INTERVAL '170 days','Sent',         '1144556677', 'POS Tienda Física'),
		('SFD','1003', CURRENT_DATE - INTERVAL '165 days','Pending',      '1007788990', 'Canal B2B'),
		('SFD','1004', CURRENT_DATE - INTERVAL '158 days','Error',        '900123456',  NULL),
		('SFD','1005', CURRENT_DATE - INTERVAL '150 days','CONTINGENCIA', '900234567',  'Ecommerce Web'),
		('SFD','1006', CURRENT_DATE - INTERVAL '140 days','DRAFT',        '1130998877', 'POS Tienda Física'),
		('SFD','1007', CURRENT_DATE - INTERVAL '132 days','Sent',         '900456789',  'Canal B2B'),
		('SFD','1008', CURRENT_DATE - INTERVAL '125 days','Sent',         '1007223344', NULL),
		('SFD','1009', CURRENT_DATE - INTERVAL '115 days','Pending',      '1098765432', 'Ecommerce Web'),
		('SFD','1010', CURRENT_DATE - INTERVAL '105 days','Sent',         '1144556677', 'POS Tienda Física'),
		('SFD','1011', CURRENT_DATE - INTERVAL '95 days', 'Error',        '1007788990', NULL),
		('SFD','1012', CURRENT_DATE - INTERVAL '85 days', 'CONTINGENCIA', '900123456',  'Canal B2B'),
		('SFD','1013', CURRENT_DATE - INTERVAL '75 days', 'Sent',         '900234567',  'Ecommerce Web'),
		('SFD','1014', CURRENT_DATE - INTERVAL '65 days', 'Pending',      '1130998877', 'POS Tienda Física'),
		('SFD','1015', CURRENT_DATE - INTERVAL '55 days', 'Sent',         '900456789',  'Canal B2B'),
		('SFD','1016', CURRENT_DATE - INTERVAL '45 days', 'DRAFT',        '1007223344', NULL),
		('SFD','1017', CURRENT_DATE - INTERVAL '35 days', 'Sent',         '1098765432', 'Ecommerce Web'),
		('SFD','1018', CURRENT_DATE - INTERVAL '25 days', 'Pending',      '1144556677', 'POS Tienda Física'),
		('SFD','1019', CURRENT_DATE - INTERVAL '15 days', 'Sent',         '1007788990', 'Canal B2B'),
		('SFD','1020', CURRENT_DATE - INTERVAL '8 days',  'Error',        '900123456',  NULL),
		('SFD','1021', CURRENT_DATE,                      'Sent',         '900234567',  'Ecommerce Web'),
		('SFD','1022', CURRENT_DATE,                      'Sent',         '1130998877', 'POS Tienda Física'),
		('SFD','1023', CURRENT_DATE,                      'Pending',      '900456789',  'Canal B2B'),
		('SFD','1024', CURRENT_DATE,                      'DRAFT',        '1007223344', NULL),
		('SFD','1025', CURRENT_DATE,                      'Error',        '1098765432', NULL),
		('SFD','1026', CURRENT_DATE,                      'CONTINGENCIA', '1144556677', 'Ecommerce Web')
	) AS t(prefix, number, issue_date, dian_status, customer_tax_id, channel_name)
)
INSERT INTO invoices (
	id, company_id, customer_id, prefix, number, date,
	net_total, tax_total, grand_total,
	dian_status, cufe, payment_form, payment_method_code,
	channel_id, logistics_cost, discount_total,
	created_at, updated_at
)
SELECT
	gen_random_uuid(),
	'55555555-0000-0000-0000-000000000010'::uuid,
	c.customer_id,
	s.prefix,
	s.number,
	s.issue_date::date,
	0, 0, 0,
	s.dian_status,
	'seed-' || s.prefix || '-' || s.number,
	'CONTADO',
	'10',
	ch.channel_id,
	CASE WHEN (right(s.number,1)::int % 3)=0 THEN 8000 ELSE 2500 END,
	CASE WHEN (right(s.number,1)::int % 4)=0 THEN 3000 ELSE 0 END,
	NOW(), NOW()
FROM seed_invoices s
JOIN tmp_seed_customers c ON c.tax_id = s.customer_tax_id
LEFT JOIN tmp_seed_channels ch ON ch.name = s.channel_name
ON CONFLICT (company_id, prefix, number) DO UPDATE SET
	customer_id = EXCLUDED.customer_id,
	date = EXCLUDED.date,
	dian_status = EXCLUDED.dian_status,
	cufe = EXCLUDED.cufe,
	channel_id = EXCLUDED.channel_id,
	logistics_cost = EXCLUDED.logistics_cost,
	discount_total = EXCLUDED.discount_total,
	updated_at = NOW();

-- 6) Detalles (2 líneas por factura)
WITH inv AS (
	SELECT id, number
	FROM invoices
	WHERE company_id = '55555555-0000-0000-0000-000000000010'::uuid
		AND prefix = 'SFD'
		AND number BETWEEN '1001' AND '1026'
),
lines AS (
	SELECT i.id AS invoice_id, i.number, 1 AS line_no,
				 CASE WHEN right(i.number,1) IN ('1','5','9') THEN 'WHY-001' ELSE 'MAG-001' END AS sku
	FROM inv i
	UNION ALL
	SELECT i.id AS invoice_id, i.number, 2 AS line_no,
				 CASE WHEN right(i.number,1) IN ('2','6','0') THEN 'CRE-002' ELSE 'CHI-001' END AS sku
	FROM inv i
),
resolved AS (
	SELECT
		(
			substr(md5(number || '-' || line_no::text),1,8) || '-' ||
			substr(md5(number || '-' || line_no::text),9,4) || '-' ||
			substr(md5(number || '-' || line_no::text),13,4) || '-' ||
			substr(md5(number || '-' || line_no::text),17,4) || '-' ||
			substr(md5(number || '-' || line_no::text),21,12)
		)::uuid AS id,
		l.invoice_id,
		p.product_id,
		(1 + ((right(l.number,1)::int + l.line_no) % 4))::numeric AS quantity,
		p.price AS unit_price
	FROM lines l
	JOIN tmp_seed_products p ON p.sku = l.sku
)
INSERT INTO invoice_details (id, invoice_id, product_id, quantity, unit_price, tax_rate, subtotal)
SELECT
	id,
	invoice_id,
	product_id,
	quantity,
	unit_price,
	19,
	ROUND(quantity * unit_price, 2)
FROM resolved
ON CONFLICT (id) DO UPDATE SET
	invoice_id = EXCLUDED.invoice_id,
	product_id = EXCLUDED.product_id,
	quantity = EXCLUDED.quantity,
	unit_price = EXCLUDED.unit_price,
	tax_rate = EXCLUDED.tax_rate,
	subtotal = EXCLUDED.subtotal;

-- Recalcular totales de cabecera
WITH sums AS (
	SELECT d.invoice_id, SUM(d.subtotal) AS gross
	FROM invoice_details d
	JOIN invoices i ON i.id = d.invoice_id
	WHERE i.company_id = '55555555-0000-0000-0000-000000000010'::uuid
		AND i.prefix = 'SFD'
		AND i.number BETWEEN '1001' AND '1026'
	GROUP BY d.invoice_id
)
UPDATE invoices i
SET
	net_total = ROUND(s.gross / 1.19, 2),
	tax_total = ROUND(s.gross - (s.gross / 1.19), 2),
	grand_total = ROUND(s.gross, 2),
	updated_at = NOW()
FROM sums s
WHERE i.id = s.invoice_id;

-- 7) Loyalty: perfiles + historial de puntos
WITH loyalty_profiles AS (
	SELECT * FROM (VALUES
		('1098765432', 450000::numeric),
		('1144556677', 220000::numeric),
		('1007788990', 125000::numeric)
	) AS t(tax_id, ltv)
)
INSERT INTO crm_customer_profiles (id, customer_id, company_id, ltv, created_at, updated_at)
SELECT gen_random_uuid(), c.customer_id, '55555555-0000-0000-0000-000000000010'::uuid, lp.ltv, NOW(), NOW()
FROM loyalty_profiles lp
JOIN tmp_seed_customers c ON c.tax_id = lp.tax_id
ON CONFLICT (customer_id) DO UPDATE SET
	ltv = EXCLUDED.ltv,
	updated_at = NOW();

INSERT INTO crm_interactions (
	id, company_id, customer_id, type, subject, body, created_by, created_at
)
SELECT * FROM (
	SELECT gen_random_uuid() AS id, '55555555-0000-0000-0000-000000000010'::uuid AS company_id, c1.customer_id, 'other'::varchar, 'LOYALTY_POINTS_AWARD'::varchar, '{"delta":200,"reason":"Compra factura SFD-1021","reference_id":"SFD-1021"}'::text, NULL::uuid, NOW()-INTERVAL '20 days' FROM tmp_seed_customers c1 WHERE c1.tax_id='1098765432'
	UNION ALL
	SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, c1.customer_id, 'other', 'LOYALTY_POINTS_REDEEM', '{"delta":-50,"reason":"Canje cupón 5%"}', NULL::uuid, NOW()-INTERVAL '10 days' FROM tmp_seed_customers c1 WHERE c1.tax_id='1098765432'
	UNION ALL
	SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, c1.customer_id, 'other', 'LOYALTY_POINTS_AWARD', '{"delta":120,"reason":"Compra factura SFD-1025","reference_id":"SFD-1025"}', NULL::uuid, NOW()-INTERVAL '1 day' FROM tmp_seed_customers c1 WHERE c1.tax_id='1098765432'
	UNION ALL
	SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, c2.customer_id, 'other', 'LOYALTY_POINTS_AWARD', '{"delta":80,"reason":"Compra factura SFD-1010","reference_id":"SFD-1010"}', NULL::uuid, NOW()-INTERVAL '15 days' FROM tmp_seed_customers c2 WHERE c2.tax_id='1144556677'
	UNION ALL
	SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, c2.customer_id, 'other', 'LOYALTY_POINTS_REDEEM', '{"delta":-20,"reason":"Canje envío gratis"}', NULL::uuid, NOW()-INTERVAL '7 days' FROM tmp_seed_customers c2 WHERE c2.tax_id='1144556677'
	UNION ALL
	-- Cliente con balance bajo para probar 409 en redeem (intenta canjear 100)
	SELECT gen_random_uuid(), '55555555-0000-0000-0000-000000000010'::uuid, c3.customer_id, 'other', 'LOYALTY_POINTS_AWARD', '{"delta":30,"reason":"Bienvenida loyalty"}', NULL::uuid, NOW()-INTERVAL '3 days' FROM tmp_seed_customers c3 WHERE c3.tax_id='1007788990'
) s
ON CONFLICT (id) DO NOTHING;

-- 8) RBAC: registrar nuevas rutas CRM para menú/pantallas frontend
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
SELECT crm_module.id, 'crm.analytics-segmentation', 'Segmentación CRM', '/crm/analytics/segmentation', '/api/crm/analytics/segmentation', 14, true FROM crm_module
UNION ALL
SELECT crm_module.id, 'crm.analytics-monthly-evolution', 'Evolución mensual CRM', '/crm/analytics/monthly-evolution', '/api/crm/analytics/monthly-evolution', 15, true FROM crm_module
ON CONFLICT (key) DO UPDATE
SET module_id = EXCLUDED.module_id,
	name = EXCLUDED.name,
	frontend_route = EXCLUDED.frontend_route,
	api_endpoint = EXCLUDED.api_endpoint,
	"order" = EXCLUDED."order",
	is_active = EXCLUDED.is_active,
	updated_at = NOW();

COMMIT;

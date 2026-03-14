-- Verificación post-seed para FE

-- 1) analytics/margins: canales y SKUs con data útil
SELECT
  COUNT(DISTINCT COALESCE(sc.id::text, 'direct')) AS channels_with_sales,
  COUNT(DISTINCT p.sku) AS skus_with_sales
FROM invoices i
JOIN invoice_details d ON d.invoice_id = i.id
JOIN products p ON p.id = d.product_id
LEFT JOIN sales_channels sc ON sc.id = i.channel_id
WHERE i.company_id = '55555555-0000-0000-0000-000000000010'::uuid
  AND i.date >= CURRENT_DATE - INTERVAL '6 months'
  AND i.dian_status NOT IN ('DRAFT', 'ERROR_GENERATION');

-- 2) invoices: mezcla de estados
SELECT dian_status, COUNT(*)
FROM invoices
WHERE company_id = '55555555-0000-0000-0000-000000000010'::uuid
  AND prefix = 'SFD'
GROUP BY dian_status
ORDER BY dian_status;

-- 3) resumen DIAN (misma regla que endpoint)
SELECT
  COUNT(*) FILTER (WHERE date = CURRENT_DATE AND dian_status = 'Sent') AS sent_today,
  COUNT(*) FILTER (WHERE dian_status IN ('Pending', 'DRAFT')) AS pending,
  COUNT(*) FILTER (WHERE dian_status = 'Error') AS rejected
FROM invoices
WHERE company_id = '55555555-0000-0000-0000-000000000010'::uuid;

-- 4) loyalty balances por customer
SELECT
  ci.customer_id,
  SUM(COALESCE((NULLIF(ci.body::jsonb ->> 'delta',''))::int, 0)) AS loyalty_balance,
  COUNT(*) AS events
FROM crm_interactions ci
WHERE ci.company_id = '55555555-0000-0000-0000-000000000010'::uuid
  AND ci.type = 'other'
  AND ci.subject LIKE 'LOYALTY_POINTS_%'
GROUP BY ci.customer_id
ORDER BY ci.customer_id;

-- 5) customer recomendado para probar 409 (pocos puntos)
SELECT c.id AS customer_id_409_candidate, c.tax_id
FROM customers c
WHERE c.company_id = '55555555-0000-0000-0000-000000000010'::uuid
  AND c.tax_id = '1007788990';

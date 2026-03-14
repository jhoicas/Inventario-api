# Frontend real data: migración + seed + smoke

## 1) Migraciones (PostgreSQL)

Ejecuta en orden (si tu base ya está al día, estos scripts son `IF NOT EXISTS` en gran parte):

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f internal/infrastructure/postgres/migrations/000_full_schema.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f internal/infrastructure/postgres/migrations/016_saas_modules_and_channels.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f internal/infrastructure/postgres/migrations/018_invoice_dian_tracking.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f internal/infrastructure/postgres/migrations/020_inventory_raw_materials.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f internal/infrastructure/postgres/migrations/021_crm_module.up.sql
```

## 2) Seed

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed_frontend_endpoints.sql
```

## 3) Verificación SQL

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/verify_frontend_endpoints.sql
```

## 4) Levantar API

```bash
go run ./cmd/api
```

## 5) Smoke tests HTTP

Define token y base URL:

```bash
export BASE_URL="http://localhost:8080"
export TOKEN="<JWT_VALIDO_CON_COMPANY_ID=55555555-0000-0000-0000-000000000010>"
```

### 5.1 Analytics margins

```bash
curl -s "$BASE_URL/api/analytics/margins?top_n=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

Esperado: `channel_profitability`, `sku_ranking`, `pareto_skus` con datos.

### 5.2 Invoices list

```bash
curl -s "$BASE_URL/api/invoices?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

Esperado: estados mixtos (`Sent`, `Pending`, `DRAFT`, `Error`, `CONTINGENCIA`).

### 5.3 DIAN summary

```bash
curl -s "$BASE_URL/api/billing/dian/summary" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

Esperado:

```json
{
  "sent_today": 2,
  "pending": 8,
  "rejected": 4
}
```

(Los valores exactos pueden variar si ya existían facturas para la misma empresa.)

### 5.4 Loyalty balance

Busca un customer_id para tax_id `1007788990`:

```bash
psql "$DATABASE_URL" -Atc "select id from customers where company_id='55555555-0000-0000-0000-000000000010' and tax_id='1007788990' limit 1"
```

Consulta balance:

```bash
curl -s "$BASE_URL/api/crm/customers/<CUSTOMER_ID>/loyalty" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

### 5.5 Redeem OK y conflicto (409)

Canje válido:

```bash
curl -s -X POST "$BASE_URL/api/crm/loyalty/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"<CUSTOMER_ID_CON_BALANCE_ALTO>","points":20,"reason":"Canje QA"}'
```

Canje inválido por puntos insuficientes:

```bash
curl -i -X POST "$BASE_URL/api/crm/loyalty/redeem" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"<CUSTOMER_ID_tax_id_1007788990>","points":100,"reason":"Canje imposible"}'
```

Esperado: HTTP `409` y mensaje `puntos insuficientes`.

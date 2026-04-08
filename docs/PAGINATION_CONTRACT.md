# Pagination Contract

## Standard response for paginated endpoints

All paginated listing endpoints MUST return this shape:

```json
{
  "items": [],
  "total": 0,
  "limit": 20,
  "offset": 0
}
```

Rules:
- `items`: current page data.
- `total`: real total count in DB after applying the same filters as `items`.
- `limit`: requested page size after normalization.
- `offset`: requested displacement after normalization.

## Temporary legacy compatibility mode

Default response stays canonical (`v2`):

```json
{
  "items": [],
  "total": 0,
  "limit": 20,
  "offset": 0
}
```

Legacy mode can be requested temporarily with:
- Query param: `?legacy=true`
- Header: `X-Pagination-Format: legacy`
- Header: `X-Legacy-Pagination: true`

Legacy response shape:

```json
{
  "items": [],
  "page": {
    "total": 0,
    "limit": 20,
    "offset": 0
  }
}
```

Feature flags:
- `PAGINATION_LEGACY_COMPAT_ENABLED=true|false`
- `PAGINATION_LEGACY_COMPAT_ENDPOINTS=*|comma,separated,endpoints`

Legacy deprecation target: `2026-07-31`.

This contract allows frontend to render labels like:
`Mostrando 1-5 de 120`
without heuristic calculations.

## Example: with results

```json
{
  "items": [
    {
      "id": "a3f24f26-4d93-4d68-84ff-1f44ef7b2281",
      "name": "Proveedor Andino SAS"
    },
    {
      "id": "df431354-d8e2-4b8d-b6fd-4296fca8f5f0",
      "name": "Comercializadora Norte"
    }
  ],
  "total": 120,
  "limit": 5,
  "offset": 0
}
```

## Example: no results

```json
{
  "items": [],
  "total": 0,
  "limit": 20,
  "offset": 0
}
```

## Legacy flat-array endpoints

These endpoints are marked as `LEGACY_ARRAY` in Swagger descriptions and still return plain arrays:

- `GET /api/analytics/raw-materials-impact`
- `GET /api/companies/{id}/resolutions`
- `GET /api/resolutions`
- `GET /api/rbac/roles`
- `GET /api/crm/remarketing`
- `GET /api/crm/opportunities/funnel`
- `GET /api/crm/campaign-templates`
- `GET /api/crm/tickets/overdue`

Notes:
- `GET /api/inventory/replenishment-list` returns `{"total": number, "replenishments": []}` and is also marked as legacy payload.
- Non-paginated collection endpoints may remain array-based by design.

# DIAN Multi-Tenant Configuration

## Overview

El sistema soporta múltiples ambientes de DIAN (habilitación y producción) por empresa. Cada empresa (`Company`) puede tener su propio certificado y ambiente configurado.

## Company Model

La entidad `Company` ahora incluye:

```go
type Company struct {
    ID           string  // UUID
    Name         string
    NIT          string
    Address      string
    Phone        string
    Email        string
    Status       string
    Environment  string  // "habilitacion" | "produccion"
    CertHab      string  // Certificado de habilitación (PEM o base64)
    CertProd     string  // Certificado de producción (PEM o base64)
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// DianConfig() retorna la URL y certificado según el environment
func (c *Company) DianConfig() (url, cert string) {
    switch c.Environment {
    case "produccion":
        return "https://vpfe.dian.gov.co/WcfDianCustomerServices.svc", c.CertProd
    default:
        return "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc", c.CertHab
    }
}
```

## DIANConfigMiddleware

El middleware `DIANConfigMiddleware` inyecta automáticamente la configuración de DIAN correcta en el contexto de la solicitud.

### Ubicación
- Archivo: `internal/billing/dian_middleware.go`
- Paquete: `billing`

### Uso

```go
// En router.go
cust.Get("/lookup",
    RequireModule(entity.ModuleBilling, deps.ModuleService),
    dianws.DIANConfigMiddleware(deps.CompanyRepo),  // ← Middleware
    deps.CustomerLookup.Lookup,
)
```

### Valores Inyectados

El middleware inyecta los siguientes valores en `c.Locals()`:

| Clave | Tipo | Descripción |
|-------|------|-------------|
| `dian_url` | `string` | URL SOAP del endpoint DIAN (vpfe-hab o vpfe) |
| `dian_cert` | `string` | Certificado para autenticación cliente (puede estar vacío) |
| `company` | `*entity.Company` | Entidad Company completa |

### Comportamiento

1. **Lee `company_id` del contexto** (inyectado por `AuthMiddleware` via JWT)
2. **Consulta `CompanyRepository.GetByID()`** para obtener la empresa
3. **Llama `company.DianConfig()`** para resolver URL y certificado
4. **Inyecta valores en Fiber context** para uso downstream
5. **Fallback seguro**: Si falla cualquier paso, usa defaults (habilitación)

## GetAcquirer Function

```go
// Antes: env string
// Ahora: url, cert string
func GetAcquirer(ctx context.Context, url, idType, idNumber, cert string) (*AcquirerInfo, error) {
    // url: "https://vpfe-hab.dian.gov.co/..." o "https://vpfe.dian.gov.co/..."
    // cert: certificado PEM (uso futuro para mTLS)
    // ...
}
```

## CustomerLookupHandler Integration

```go
// En handlers.go - Lookup()
func (h *CustomerLookupHandler) Lookup(c *fiber.Ctx) error {
    // ... validación de params ...
    
    // Obtener configuración DIAN del middleware
    dianURL, dianCert, err := GetDIANConfigFromContext(c)
    if err != nil {
        // Fallback a defaults
        dianURL = "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc"
        dianCert = ""
    }
    
    // Usar URL y cert específicos de la empresa
    info, err := GetAcquirer(c.Context(), dianURL, idType, idNumber, dianCert)
    // ...
}
```

## Database Schema

Asume que la tabla `companies` tiene estas columnas:

```sql
ALTER TABLE companies ADD COLUMN environment VARCHAR(20) DEFAULT 'habilitacion';
ALTER TABLE companies ADD COLUMN cert_hab TEXT;
ALTER TABLE companies ADD COLUMN cert_prod TEXT;
```

## Configuration in main.go

Actualizar `RouterDeps` construction:

```go
deps := RouterDeps{
    CompanyUC: companyUseCase,
    CompanyRepo: companyRepository,  // ← Nuevo: requerido para middleware
    // ... resto de dependencias ...
}

Router(app, deps)
```

## Helper Functions

### GetDIANConfigFromContext
```go
dianURL, dianCert, err := GetDIANConfigFromContext(c)
```
Extrae configuración inyectada por el middleware. Retorna error si no está disponible.

### GetCompanyFromContext
```go
company, err := GetCompanyFromContext(c)
```
Extrae la entidad Company completa inyectada por el middleware.

## Usage Flow

```
1. Request: GET /api/customers/lookup?id_type=31&id_number=123456789-0
   ├─ AuthMiddleware → valida JWT, carga company_id
   ├─ DIANConfigMiddleware → carga company, extrae dian_url y dian_cert
   └─ CustomerLookupHandler.Lookup → usa dian_url, dian_cert
2. GetAcquirer(ctx, dianURL, idType, idNumber, dianCert)
   └─ POST a vpfe-hab.dian.gov.co o vpfe.dian.gov.co según environment
3. Response: 200 OK con AcquirerInfo
```

## Future Enhancements

- **mTLS Support**: Usar `cert` para autenticación cliente bidireccional
- **Certificate Expiry Alerts**: Monitorear vencimiento de certificados
- **Per-Company IP Whitelisting**: Diferentes IPs permitidas por environment
- **Async Certificate Refresh**: Renovación automática de certificados

# Inventory Pro - Backend SaaS de Inventario (Colombia)

Backend construido con **Clean Architecture + Hexagonal** en Go.

## 🚀 Inicio Rápido

### 1. Instalar dependencias

```powershell
go mod tidy
```

### 2. Configurar Base de Datos

**Opción A: Ejecutar migración manualmente en Supabase**

1. Ve al SQL Editor en tu proyecto Supabase
2. Ejecuta el contenido de `internal/infrastructure/postgres/migrations/001_companies.up.sql`

**Opción B: Usar psql desde terminal**

```powershell
# Si tienes psql instalado
psql "postgresql://postgres:xF#*W3&%mDceWPL@db.ghewngonrvknaqdvigkp.supabase.co:5432/postgres" -f internal/infrastructure/postgres/migrations/001_companies.up.sql
```

### 3. Configurar Variables de Entorno

El archivo `.env` ya está creado con tus credenciales de Supabase. Si necesitas cambiarlas, edita `.env`.

### 4. Compilar y Ejecutar

**Opción A: Ejecutar directamente con `go run`**

```powershell
go run ./cmd/api
```

**Opción B: Compilar y ejecutar binario**

```powershell
go build -o api.exe ./cmd/api
.\api.exe
```

### 5. Probar la API

Una vez que el servidor esté corriendo (verás logs en consola), prueba:

**Health Check:**
```powershell
curl http://localhost:8080/health
# o en navegador: http://localhost:8080/health
```

**Crear una empresa:**
```powershell
curl -X POST http://localhost:8080/api/companies `
  -H "Content-Type: application/json" `
  -d '{"name":"Mi Empresa","nit":"900123456-1","address":"Calle 1","email":"test@example.com"}'
```

**Listar empresas:**
```powershell
curl http://localhost:8080/api/companies
```

**Obtener empresa por ID:**
```powershell
curl http://localhost:8080/api/companies/{id-aqui}
```

## 📁 Estructura del Proyecto

```
/cmd/api/main.go              # Punto de entrada
/internal/
  /domain/                    # Entidades y puertos (sin dependencias externas)
    /entity/                  # Entidades de dominio
    /repository/              # Interfaces de repositorio (puertos)
  /application/              # Casos de uso y DTOs
    /dto/                     # Data Transfer Objects
    /usecase/                 # Casos de uso (lógica de negocio)
  /infrastructure/           # Adaptadores de persistencia
    /postgres/               # Implementación PostgreSQL
  /interfaces/http/          # Handlers y routers HTTP
/pkg/
  /config/                   # Configuración (Viper)
  /logger/                   # Logger (Zerolog)
```

## 🔧 Variables de Entorno

Ver `.env` para todas las variables disponibles. Las principales:

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `HTTP_HOST`, `HTTP_PORT`
- `JWT_SECRET`, `JWT_EXPIRATION_MINUTES`
- `APP_ENV` (development/staging/production)

## 🛑 Detener el Servidor

Presiona `Ctrl+C` para hacer un graceful shutdown.

---

## 🐳 Ejecutar con Docker

### Opción 1: Docker Compose (Recomendado)

**Producción:**
```powershell
# Construir y levantar
docker-compose up -d

# Ver logs
docker-compose logs -f api

# Detener
docker-compose down
```

**Desarrollo:**
```powershell
# Levantar en modo desarrollo
docker-compose -f docker-compose.dev.yml up --build
```

### Opción 2: Docker directamente

```powershell
# Construir imagen
docker build -t inventory-pro-api .

# Ejecutar contenedor
docker run -d \
  --name inventory-pro-api \
  -p 8080:8080 \
  -e DB_HOST=db.ghewngonrvknaqdvigkp.supabase.co \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=xF#*W3&%mDceWPL \
  -e DB_NAME=postgres \
  -e DB_SSLMODE=require \
  inventory-pro-api

# Ver logs
docker logs -f inventory-pro-api

# Detener
docker stop inventory-pro-api
docker rm inventory-pro-api
```

### Opción 3: Usar Makefile (si tienes Make instalado)

```powershell
# Construir
make build

# Levantar
make up

# Ver logs
make logs

# Detener
make down

# Limpiar todo
make clean
```

### Variables de Entorno en Docker

Puedes crear un archivo `.env` y Docker Compose lo leerá automáticamente, o pasarlas directamente en `docker-compose.yml`.

**Nota:** Las credenciales de Supabase están hardcodeadas en `docker-compose.yml` para facilitar el inicio rápido. En producción, usa secrets de Docker o variables de entorno del sistema.

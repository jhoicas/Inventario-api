# Script PowerShell para ejecutar la API con Docker
param(
    [string]$Mode = "prod"  # prod o dev
)

Write-Host "🐳 Inventory Pro - Docker Setup" -ForegroundColor Cyan
Write-Host ""

# Verificar que Docker esté instalado
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Docker no está instalado o no está en PATH" -ForegroundColor Red
    Write-Host "   Instala Docker Desktop desde: https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
    exit 1
}

# Verificar que Docker esté corriendo
try {
    docker info | Out-Null
} catch {
    Write-Host "❌ Docker no está corriendo. Inicia Docker Desktop." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Docker está disponible" -ForegroundColor Green
Write-Host ""

if ($Mode -eq "dev") {
    Write-Host "🔧 Modo: Desarrollo" -ForegroundColor Yellow
    Write-Host "▶️  Construyendo y levantando contenedor..." -ForegroundColor Cyan
    docker-compose -f docker-compose.dev.yml up --build
} else {
    Write-Host "🚀 Modo: Producción" -ForegroundColor Green
    Write-Host "▶️  Construyendo y levantando contenedor..." -ForegroundColor Cyan
    docker-compose up --build
}

Write-Host ""
Write-Host "💡 Comandos útiles:" -ForegroundColor Yellow
Write-Host "   Ver logs:     docker-compose logs -f api" -ForegroundColor Gray
Write-Host "   Detener:      docker-compose down" -ForegroundColor Gray
Write-Host "   Reiniciar:    docker-compose restart" -ForegroundColor Gray
Write-Host "   Shell:        docker-compose exec api sh" -ForegroundColor Gray

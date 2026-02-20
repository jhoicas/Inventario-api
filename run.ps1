# Script PowerShell para ejecutar la API
Write-Host "🚀 Iniciando Inventory Pro API..." -ForegroundColor Green

# Verificar que Go esté instalado
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Go no está instalado o no está en PATH" -ForegroundColor Red
    exit 1
}

# Instalar/actualizar dependencias
Write-Host "📦 Instalando dependencias..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error al instalar dependencias" -ForegroundColor Red
    exit 1
}

# Ejecutar la API
Write-Host "▶️  Ejecutando servidor en http://localhost:8080" -ForegroundColor Cyan
Write-Host "   Presiona Ctrl+C para detener" -ForegroundColor Gray
Write-Host ""

go run ./cmd/api

.PHONY: build up down logs restart shell clean test test-coverage lint vendor deploy deploy-local deploy-do logs-do rollback build-no-cache

# ── Vendoring ──────────────────────────────────────────────────────────────────

## vendor: Regenera la carpeta vendor/ a partir de go.mod (para builds sin red).
vendor:
	go mod vendor
	@echo "✓ vendor/ actualizado."

# ── Testing ───────────────────────────────────────────────────────────────────

## test: Ejecuta todos los tests unitarios con salida verbose (usa vendor/).
test:
	go test -mod=vendor -v -count=1 -timeout 120s ./...

## test-coverage: Ejecuta tests y genera reporte de cobertura HTML.
test-coverage:
	go test -mod=vendor -count=1 -timeout 120s -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Reporte de cobertura generado en coverage.html"

## lint: Verifica formato y errores estáticos del código (usa vendor/).
lint:
	go vet -mod=vendor ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "(golangci-lint no instalado, saltando)"

# ── Docker / despliegue ───────────────────────────────────────────────────────

## build: Construye la imagen Docker (incluye los tests dentro del Dockerfile).
build:
	docker-compose build

## build-no-cache: Build sin caché de Docker.
build-no-cache:
	docker-compose build --no-cache

## up: Construye y levanta todos los servicios.
up:
	docker-compose up -d

## dev: Levanta en modo desarrollo con hot-reload.
dev:
	docker-compose -f docker-compose.dev.yml up --build

## down: Detiene y elimina los contenedores.
down:
	docker-compose down

## logs: Muestra los logs del servicio api en tiempo real.
logs:
	docker-compose logs -f api

## restart: Reinicia todos los servicios.
restart:
	docker-compose restart

## shell: Abre una shell dentro del contenedor api.
shell:
	docker-compose exec api sh

## clean: Elimina contenedores, volúmenes e imagen.
clean:
	docker-compose down -v
	docker rmi inventory-pro-api-api || true

## deploy: Pipeline completo: tests locales → rsync → build en servidor → levanta producción.
## Requiere REMOTE_HOST y SSH_KEY configurados (o exportados como variables de entorno).
## Falla y aborta si cualquier test no pasa (los tests también corren dentro del Dockerfile).
deploy: test
	@echo "✓ Tests locales pasaron. Iniciando despliegue remoto..."
	bash deploy.sh
	@echo "✓ Despliegue completado."

## deploy-local: Levanta el stack de producción localmente (útil para staging en CI).
deploy-local: test
	@echo "✓ Tests pasaron. Levantando stack de producción localmente..."
	docker compose -f docker-compose.prod.yml up -d --build
	@echo "✓ Stack levantado."

## rollback: Revierte al tag anterior en producción.
rollback:
	bash deploy.sh --rollback

## deploy-do: Despliega directamente en el Droplet usando docker-compose.prod.yml.
deploy-do:
	docker compose -f docker-compose.prod.yml up -d --build

## logs-do: Sigue los logs del entorno de producción en el Droplet local.
logs-do:
	docker compose -f docker-compose.prod.yml logs -f

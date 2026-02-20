.PHONY: build up down logs restart shell clean

# Construir la imagen
build:
	docker-compose build

# Construir y levantar servicios
up:
	docker-compose up -d

# Levantar en modo desarrollo
dev:
	docker-compose -f docker-compose.dev.yml up --build

# Detener servicios
down:
	docker-compose down

# Ver logs
logs:
	docker-compose logs -f api

# Reiniciar servicios
restart:
	docker-compose restart

# Shell en el contenedor
shell:
	docker-compose exec api sh

# Limpiar (detener y eliminar contenedores, volúmenes)
clean:
	docker-compose down -v
	docker rmi inventory-pro-api-api || true

# Build sin cache
build-no-cache:
	docker-compose build --no-cache

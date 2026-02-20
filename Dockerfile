# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Instalar dependencias del sistema necesarias para compilación
RUN apk add --no-cache git

# Copiar go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Instalar ca-certificates y curl para conexiones HTTPS/SSL y healthcheck
RUN apk --no-cache add ca-certificates tzdata curl

# Copiar el binario compilado desde builder
COPY --from=builder /app/api .

# Exponer puerto
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Ejecutar la aplicación
CMD ["./api"]

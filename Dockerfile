# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage build: compilación estática y imagen mínima para producción.
# Base de datos PostgreSQL es externa (servicio administrado); no va en Compose.
# ─────────────────────────────────────────────────────────────────────────────

# Stage 1: builder
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o api \
    ./cmd/api

# Stage 2: runner
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Binario compilado
COPY --from=builder /build/api .

# Swagger: el servidor sirve ./docs/swagger.json (FilePath en main.go)
COPY --from=builder /build/docs ./docs

# Opcional: certificado DIAN (.p12). Si usa certificados en la imagen, cree
# la carpeta certs/ en la raíz del proyecto, coloque allí su .p12 y descomente:
# COPY --from=builder /build/certs ./certs
# Luego en .env.prod: DIAN_CERT_PATH=/app/certs/certificado_prueba.p12

EXPOSE 8080

USER 1000:1000

CMD ["./api"]

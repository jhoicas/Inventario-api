# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  ETAPA 1 – Builder (modo vendor: sin descargas externas)                    ║
# ║  • COPY . . incluye vendor/; no se ejecuta go mod download                   ║
# ║  • Tests y build usan -mod=vendor                                            ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
# Copiar todo el árbol (incluye go.mod, go.sum y vendor/)
COPY . .

# ── CI Gate (usa -mod=vendor, sin descargas externas) ───────────────────────────
RUN go test -mod=vendor -count=1 -timeout 120s ./...

# ── Compilación usando solo vendor/ ───────────────────────────────────────────
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=vendor -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o erp-api ./cmd/api

# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  ETAPA 2 – Runtime                                                          ║
# ║  Imagen mínima: solo el binario + librerías del sistema esenciales          ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
FROM alpine:3.20

# ca-certificates → necesario para TLS saliente (Anthropic, DIAN, Supabase, etc.)
# tzdata         → permite configurar la zona horaria del proceso
# curl           → healthcheck del contenedor
RUN apk --no-cache add ca-certificates tzdata curl

# Zona horaria Colombia para que los timestamps del servidor sean correctos.
ENV TZ=America/Bogota

WORKDIR /app

# ── Usuario no-root (principio de mínimo privilegio) ──────────────────────────
# El proceso de la API no necesita privilegios de root en runtime.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder --chown=appuser:appgroup /app/erp-api .

USER appuser

EXPOSE 8080

# Healthcheck: verifica que la API responde antes de que el load balancer
# (Caddy) empiece a enviarle tráfico.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./erp-api"]

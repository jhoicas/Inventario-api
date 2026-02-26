#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  deploy.sh — Despliegue automatizado en DigitalOcean Droplet (Ubuntu)       ║
# ║                                                                              ║
# ║  Uso:                                                                        ║
# ║    ./deploy.sh                    # despliega la rama actual                 ║
# ║    ./deploy.sh --skip-tests       # omite tests locales (no recomendado)     ║
# ║    ./deploy.sh --rollback         # revierte al tag anterior                 ║
# ║                                                                              ║
# ║  Pre-requisitos en el Droplet:                                               ║
# ║    - Docker + Docker Compose v2                                              ║
# ║    - .env.prod en el directorio del proyecto                                 ║
# ║    - Puertos 80 y 443 abiertos (ufw allow 80 && ufw allow 443)              ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
set -euo pipefail

# ── Configuración ─────────────────────────────────────────────────────────────
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_HOST="${REMOTE_HOST:-IP_DEL_DROPLET}"         # ← cambiar o exportar
REMOTE_DIR="${REMOTE_DIR:-/opt/invorya-erp}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_rsa}"
COMPOSE_FILE="docker-compose.prod.yml"
SKIP_TESTS=false
ROLLBACK=false

# ── Colores para output ────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()    { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()      { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Argumentos ────────────────────────────────────────────────────────────────
for arg in "$@"; do
    case $arg in
        --skip-tests) SKIP_TESTS=true ;;
        --rollback)   ROLLBACK=true ;;
        *) log_warn "Argumento desconocido: $arg" ;;
    esac
done

# ── Rollback ───────────────────────────────────────────────────────────────────
if [ "$ROLLBACK" = true ]; then
    log_warn "Iniciando ROLLBACK en $REMOTE_HOST..."
    ssh -i "$SSH_KEY" "$REMOTE_USER@$REMOTE_HOST" "
        cd $REMOTE_DIR
        git log --oneline -5
        PREV_TAG=\$(git describe --abbrev=0 --tags \$(git rev-list --tags --skip=1 --max-count=1) 2>/dev/null || echo '')
        if [ -z \"\$PREV_TAG\" ]; then
            echo 'No hay tag anterior disponible para rollback.'
            exit 1
        fi
        echo \"Revirtiendo a: \$PREV_TAG\"
        git checkout \$PREV_TAG
        docker compose -f $COMPOSE_FILE up -d --build
    "
    log_ok "Rollback completado."
    exit 0
fi

# ── Verificaciones previas (locales) ──────────────────────────────────────────
log_info "Verificando pre-requisitos locales..."

command -v git  >/dev/null 2>&1 || log_error "git no encontrado"
command -v ssh  >/dev/null 2>&1 || log_error "ssh no encontrado"
command -v rsync>/dev/null 2>&1 || log_error "rsync no encontrado"

# El Caddyfile debe existir y tener un dominio real configurado
if grep -q "tudominio.com" Caddyfile 2>/dev/null; then
    log_error "Caddyfile contiene 'tudominio.com'. Edítalo con tu dominio real antes de desplegar."
fi

# El .env.example no debe usarse directamente en producción
if [ ! -f ".env.prod" ]; then
    log_warn ".env.prod no encontrado localmente (se buscará en el servidor)."
fi

# ── Tests locales (CI Gate local) ─────────────────────────────────────────────
if [ "$SKIP_TESTS" = false ]; then
    log_info "Ejecutando tests locales..."
    if go test -mod=vendor -count=1 -timeout 120s ./...; then
        log_ok "Todos los tests pasaron."
    else
        log_error "Tests fallaron. Despliegue abortado."
    fi
else
    log_warn "Tests locales omitidos (--skip-tests)."
fi

# ── Tag de versión ─────────────────────────────────────────────────────────────
VERSION=$(git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD)
log_info "Versión a desplegar: $VERSION"

# ── Sincronizar código al servidor ────────────────────────────────────────────
log_info "Sincronizando código a $REMOTE_HOST:$REMOTE_DIR ..."
ssh -i "$SSH_KEY" "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_DIR"

rsync -avz --progress \
    --exclude='.git' \
    --exclude='.env' \
    --exclude='.env.prod' \
    --exclude='coverage.out' \
    --exclude='coverage.html' \
    --exclude='*.p12' \
    --exclude='*.pfx' \
    --exclude='*.pem' \
    --exclude='*.key' \
    -e "ssh -i $SSH_KEY" \
    ./ "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"

log_ok "Código sincronizado."

# ── Verificar .env.prod en el servidor ────────────────────────────────────────
ssh -i "$SSH_KEY" "$REMOTE_USER@$REMOTE_HOST" "
    if [ ! -f '$REMOTE_DIR/.env.prod' ]; then
        echo 'ERROR: .env.prod no existe en $REMOTE_DIR'
        echo 'Crea el archivo con: cp .env.example .env.prod && nano .env.prod'
        exit 1
    fi
"
log_ok ".env.prod verificado en servidor."

# ── Despliegue en el servidor ──────────────────────────────────────────────────
log_info "Desplegando en $REMOTE_HOST ..."
ssh -i "$SSH_KEY" "$REMOTE_USER@$REMOTE_HOST" "
    set -euo pipefail
    cd '$REMOTE_DIR'

    echo '→ Exportando versión...'
    export APP_VERSION='$VERSION'

    echo '→ Construyendo imagen (incluye tests en Dockerfile)...'
    docker compose -f $COMPOSE_FILE build --no-cache api

    echo '→ Levantando servicios (rolling update)...'
    docker compose -f $COMPOSE_FILE up -d --remove-orphans

    echo '→ Esperando healthcheck de la API (máx 60s)...'
    for i in \$(seq 1 12); do
        STATUS=\$(docker compose -f $COMPOSE_FILE ps api --format json 2>/dev/null | grep -o '\"Health\":\"[^\"]*\"' | cut -d'\"' -f4 || echo 'unknown')
        echo \"  Intento \$i/12: estado = \$STATUS\"
        if [ \"\$STATUS\" = 'healthy' ]; then
            echo 'API saludable.'
            break
        fi
        sleep 5
    done

    echo '→ Limpiando imágenes huérfanas...'
    docker image prune -f --filter 'dangling=true'

    echo '→ Estado final:'
    docker compose -f $COMPOSE_FILE ps
"

log_ok "Despliegue de $VERSION completado exitosamente."
log_info "API disponible en: https://$(grep -oP 'api\.\K[^\s{]+' Caddyfile | head -1 2>/dev/null || echo 'tu-dominio.com')"

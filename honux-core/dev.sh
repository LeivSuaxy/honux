#!/bin/bash
set -euo pipefail

# Uso: ./dev.sh <servicio> [VAR=valor ...]
# Ejemplos:
#   ./dev.sh api
#   ./dev.sh api PORT=9090
#   ./dev.sh migrate DATABASE_URL=postgres://other/db LOG_LEVEL=debug

SERVICE="${1:-}"
if [ -z "$SERVICE" ]; then
    echo "Uso: $0 <servicio> [VAR=valor ...]"
    echo "Servicios disponibles: api, migrate"
    exit 1
fi
shift   # Quitamos el nombre del servicio; ahora $@ son solo las variables extra

# Mapa de nombre de servicio -> directorio del paquete
declare -A PKG_DIRS=(
    ["api"]="./cmd/api"
    ["migrate"]="./cmd/migrate"
)

PKG="${PKG_DIRS[$SERVICE]:-}"
if [ -z "$PKG" ]; then
    echo "Servicio '$SERVICE' no reconocido"
    exit 1
fi

# 1. Cargar variables desde archivos .env (sin sobrescribir las explícitas)
set -a
[ -f .env ] && source .env
[ -f ".env.${SERVICE}" ] && source ".env.${SERVICE}"
set +a

# 2. Lanzar 'go run' con las variables de entorno cargadas y las pasadas por CLI
echo "🚀 Ejecutando $SERVICE desde $PKG..."
# Si hay argumentos adicionales (VAR=valor), los evaluamos antes del comando
if [ $# -gt 0 ]; then
    eval $(printf '%q ' "$@") go run "$PKG"
else
    go run "$PKG"
fi

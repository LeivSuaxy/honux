#!/bin/bash
set -euo pipefail

# Uso: ./dev.sh <servicio> [VAR=valor ...] [argumentos...]
# Ejemplos:
#   ./dev.sh api
#   ./dev.sh api PORT=9090
#   ./dev.sh migrate up
#   ./dev.sh migrate to 2 DATABASE_URL=postgres://other/db

SERVICE="${1:-}"
if [ -z "$SERVICE" ]; then
    echo "Uso: $0 <servicio> [VAR=valor ...] [argumentos...]"
    echo "Servicios disponibles: api, migrate"
    exit 1
fi
shift   # Quitamos el nombre del servicio; $@ ahora contiene opciones

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

# 2. Separar los argumentos en:
#    - variables de entorno (contienen '=')
#    - argumentos para el programa (resto)
env_vars=()
args=()
for arg in "$@"; do
    if [[ "$arg" == *"="* ]]; then
        env_vars+=("$arg")
    else
        args+=("$arg")
    fi
done

# 3. Ejecutar 'go run' con las variables de entorno acumuladas y los argumentos
echo "🚀 Ejecutando $SERVICE desde $PKG..."

# Si hay variables de entorno, las anteponemos al comando
if [ ${#env_vars[@]} -gt 0 ]; then
    # Exportar las variables temporalmente para este comando
    env $(printf '%s ' "${env_vars[@]}") go run "$PKG" "${args[@]}"
else
    go run "$PKG" "${args[@]}"
fi

#!/bin/bash
set -euo pipefail
OUT_DIR="./dist"
mkdir -p "$OUT_DIR"

PACKETS=(
    "./cmd/api"
    "./cmd/migrate"
)

for pkg in "${PACKETS[@]}"; do
    APP_NAME=$(basename "$pkg")
    OUTPUT="${OUT_DIR}/${APP_NAME}"

    echo "Building $pkg -> $OUTPUT"
    go build -o "$OUTPUT" "$pkg"
    echo "✅ $APP_NAME built"
done

echo "🚀 All binaries built in $OUT_DIR"

#!/usr/bin/env bash
set -euo pipefail

APP_NAME="proxy"
OUT_DIR="build"
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}"

cd "$(dirname "$0")"
mkdir -p "${OUT_DIR}"

targets=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "windows/amd64"
    "darwin/arm64"
)

for t in "${targets[@]}"; do
    os="${t%%/*}"
    arch="${t##*/}"
    ext=""
    [[ "${os}" == "windows" ]] && ext=".exe"

    out="${OUT_DIR}/${APP_NAME}-${os}-${arch}${ext}"
    echo "==> ${t} -> ${out}"

    GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 \
    go build -trimpath -ldflags="${LDFLAGS}" -o "${out}" .
done

echo
ls -lh "${OUT_DIR}"
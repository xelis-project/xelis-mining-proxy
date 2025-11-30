#!/usr/bin/env bash
set -euo pipefail

PROJECT="xelis-mining-proxy"
GOFLAGS=("-trimpath" "-ldflags=-s -w")

rm -rf build
mkdir -p build
cp LICENSE.txt README.md build/
cd build

build() {
    local GOOS=$1
    local GOARCH=$2
    local EXT=""

    [[ "$GOOS" == "windows" ]] && EXT=".exe"

    echo "==> Building $GOOS/$GOARCH"

    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
        go build $GOFLAGS -o "${PROJECT}-${GOOS}-${GOARCH}${EXT}" ..
}

build linux amd64
build windows amd64
build darwin amd64

# Package results
tar -cJf "${PROJECT}-linux-amd64.tar.xz" \
  "${PROJECT}-linux-amd64" LICENSE.txt README.md

tar -cJf "${PROJECT}-darwin-amd64.tar.xz" \
  "${PROJECT}-darwin-amd64" LICENSE.txt README.md

zip -9 "${PROJECT}-windows-amd64.zip" \
  "${PROJECT}-windows-amd64.exe" LICENSE.txt README.md

echo "==> Done."
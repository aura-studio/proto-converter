#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

echo "== Go mod tidy =="
go mod tidy

OUTPUT_DIR="build"
mkdir -p "$OUTPUT_DIR"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

output="$OUTPUT_DIR/proto-converter-${GOOS}-${GOARCH}"
if [ "$GOOS" = "windows" ]; then
  output="${output}.exe"
fi

echo "== Build ${GOOS}/${GOARCH} =="
go build -o "$output" ./

echo ""
echo "Build complete:"
ls -lh "$output"

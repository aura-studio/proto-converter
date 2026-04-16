#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

echo "== Go mod tidy =="
go mod tidy

OUTPUT_DIR="build"
mkdir -p "$OUTPUT_DIR"

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="$OUTPUT_DIR/proto-converter-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output="${output}.exe"
  fi
  echo "== Build ${GOOS}/${GOARCH} =="
  GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$output" ./
done

echo ""
echo "Build complete:"
ls -lh "$OUTPUT_DIR"/proto-converter-*

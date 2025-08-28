#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

echo "== Go mod tidy =="
go mod tidy

echo "== Build proto-converter.exe =="
go build -o proto-converter.exe ./

echo "Build complete: proto-converter.exe"

#!/usr/bin/env bash
set -e

VERSION=${1:-0.1.0}

mkdir -p dist

# Linux amd64
GOOS=linux GOARCH=amd64 \
  go build -o dist/forge-linux-amd64 ./cmd/main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 \
  go build -o dist/forge-darwin-amd64 ./cmd/main.go

# macOS ARM (M1/M2/M3)
GOOS=darwin GOARCH=arm64 \
  go build -o dist/forge-darwin-arm64 ./cmd/main.go

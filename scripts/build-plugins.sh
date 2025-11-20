#!/usr/bin/env bash
set -e

VERSION=${1:-0.1.0}

mkdir -p dist

# Linux amd64
#GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
#go build -buildmode=plugin -o forge-project/.forge/plugins/acolev/example-linux-amd64 ./plugins/example/example.go

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
  go build -buildmode=plugin -o  forge-project/.forge/plugins/acolev/example_darwin_amd64.so ./plugins/example/example.go

# macOS ARM (M1/M2/M3)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
  go build -buildmode=plugin -o  forge-project/.forge/plugins/acolev/example_darwin_arm64.so ./plugins/example/example.go

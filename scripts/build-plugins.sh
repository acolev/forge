#!/usr/bin/env bash
set -euo pipefail

# Directory where plugin source packages live
PLUGINS_SRC_DIR="plugins"

# Output root for built plugins
OUT_ROOT="forge-project/.forge/plugins/acolev"

# Detect current platform
HOST_GOOS="$(go env GOOS)"
HOST_GOARCH="$(go env GOARCH)"

echo "==> Building plugins for current platform: ${HOST_GOOS}/${HOST_GOARCH}"
echo "    Source: ${PLUGINS_SRC_DIR}"
echo "    Output: ${OUT_ROOT}"

mkdir -p "${OUT_ROOT}"

# Iterate over plugin packages inside ./plugins/*
for plugin_path in "${PLUGINS_SRC_DIR}"/*; do
  if [ ! -d "${plugin_path}" ]; then
    # Skip non-directories
    continue
  fi

  plugin_name="$(basename "${plugin_path}")"
  out_file="${OUT_ROOT}/${plugin_name}_${HOST_GOOS}_${HOST_GOARCH}.so"

  echo "  -> ${plugin_name} -> ${out_file}"

  GOOS="${HOST_GOOS}" GOARCH="${HOST_GOARCH}" CGO_ENABLED=1 \
    go build -buildmode=plugin -o "${out_file}" "./${plugin_path}"
done

echo "==> Done"

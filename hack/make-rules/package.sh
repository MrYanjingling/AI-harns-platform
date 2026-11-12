#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

HARNSGATEWAY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${HARNSGATEWAY_ROOT}/hack/lib/init.sh"

# Package name and the module directories laid out inside the archive.
PACKAGE_NAME="${PACKAGE_NAME:-AI-harns-platform}"
GO_MODULES=("gateway" "platform" "eam" "ems")
# 前端工作区以 host 为主应用发布；platform-ui / gateway-ui 如需单独打包，可用
# UI_MODULES="platform-ui gateway-ui" 覆盖。
UI_MODULES="${UI_MODULES:-host}"
UI_BUILD="${UI_BUILD:-1}"

# Build the Go binaries first, reusing the standard build rules.
"${HARNSGATEWAY_ROOT}/hack/make-rules/build.sh" "$@"

# Resolve the platform that was built (defaults to the host platform).
platforms=()
IFS=" " read -ra platforms <<< "${HARNSGATEWAY_BUILD_PLATFORMS:-}"
target_platform="${platforms[0]:-$(harnsgateway::golang::host_platform)}"

host_os="${target_platform%%/*}"
host_arch="${target_platform##*/}"
bin_dir="${HARNSGATEWAY_OUTPUT_BINPATH}/${target_platform}"

release_dir="${HARNSGATEWAY_OUTPUT}/release"
staging_dir="${release_dir}/${PACKAGE_NAME}"

echo "Packaging ${PACKAGE_NAME} for ${target_platform}"

rm -rf "${staging_dir}"
mkdir -p "${staging_dir}"

# Go services keep the bin/etc/log layout.
for module in "${GO_MODULES[@]}"; do
  mkdir -p "${staging_dir}/${module}/bin" "${staging_dir}/${module}/etc" "${staging_dir}/${module}/log"

  if [[ -f "${bin_dir}/${module}" ]]; then
    cp -p "${bin_dir}/${module}" "${staging_dir}/${module}/bin/"
  else
    echo "Warning: binary '${module}' not found in ${bin_dir}"
  fi

  if [[ -d "${HARNSGATEWAY_ROOT}/etc/${module}" ]]; then
    cp -Rp "${HARNSGATEWAY_ROOT}/etc/${module}/." "${staging_dir}/${module}/etc/"
  fi
done

# UI modules live under ui/<module>. Only modules with package.json are built;
# the rest are kept as empty placeholders for the future layout.
IFS=" " read -ra ui_modules <<< "${UI_MODULES}"
mkdir -p "${staging_dir}/ui"
for module in "${ui_modules[@]}"; do
  ui_src="${HARNSGATEWAY_ROOT}/ui/${module}"
  ui_dest="${staging_dir}/ui/${module}"
  mkdir -p "${ui_dest}"

  if [[ ! -f "${ui_src}/package.json" ]]; then
    echo "UI module '${module}' has no package.json, skipping build"
    continue
  fi

  if [[ "${UI_BUILD}" == "0" ]]; then
    if [[ -d "${ui_src}/dist" ]]; then
      cp -Rp "${ui_src}/dist/." "${ui_dest}/"
    else
      echo "Warning: UI build skipped and no dist found for '${module}'"
    fi
    continue
  fi

  if ! command -v pnpm >/dev/null 2>&1; then
    echo "Error: pnpm is required to build UI module '${module}' (set UI_BUILD=0 to skip)" >&2
    exit 1
  fi

  echo "Building UI module '${module}'"
  (
    cd "${ui_src}"
    pnpm install --frozen-lockfile
    pnpm build
  )

  if [[ ! -d "${ui_src}/dist" ]]; then
    echo "Error: UI module '${module}' produced no dist/" >&2
    exit 1
  fi
  cp -Rp "${ui_src}/dist/." "${ui_dest}/"
done

# Determine a version suffix, falling back to the short commit when there is no tag.
harnsgateway::version::get_version_vars
version="${HARNSGATEWAY_GIT_VERSION:-}"
if [[ -z "${version}" ]]; then
  version="$(git --work-tree "${HARNSGATEWAY_ROOT}" rev-parse --short HEAD 2>/dev/null || echo dev)"
fi

archive_name="${PACKAGE_NAME}-${version}-${host_os}_${host_arch}.zip"
archive="${release_dir}/${archive_name}"

rm -f "${archive}"
(
  cd "${release_dir}"
  zip -qr "${archive_name}" "${PACKAGE_NAME}"
)

echo "Packaged: ${archive}"

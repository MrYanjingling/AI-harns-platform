#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

HARNSGATEWAY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${HARNSGATEWAY_ROOT}/hack/lib/init.sh"

PACKAGE_NAME="${PACKAGE_NAME:-AI-harns-platform}"
GO_MODULES=("gateway" "platform" "eam" "ems")
UI_MODULES="${UI_MODULES:-host gateway-ui}"
UI_BUILD="${UI_BUILD:-1}"
RELEASE_PLATFORMS="${RELEASE_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64}"

IFS=" " read -ra platforms <<< "${RELEASE_PLATFORMS}"
if [[ ${#platforms[@]} -eq 0 ]]; then
  echo "Error: RELEASE_PLATFORMS is empty" >&2
  exit 1
fi

if ! command -v zip >/dev/null 2>&1; then
  echo "Error: zip is required for release packaging" >&2
  exit 1
fi

ui_cache="${HARNSGATEWAY_OUTPUT}/release-ui"
release_dir="${HARNSGATEWAY_OUTPUT}/release"

rm -rf "${ui_cache}" "${release_dir}"
mkdir -p "${ui_cache}" "${release_dir}"

build_ui() {
  if [[ "${UI_BUILD}" == "0" ]]; then
    return 0
  fi
  if ! command -v pnpm >/dev/null 2>&1; then
    echo "Error: pnpm is required (set UI_BUILD=0 to use existing dist)" >&2
    exit 1
  fi

  IFS=" " read -ra ui_modules <<< "${UI_MODULES}"
  for module in "${ui_modules[@]}"; do
    local ui_src="${HARNSGATEWAY_ROOT}/ui/${module}"
    local ui_out="${ui_cache}/${module}"
    if [[ ! -f "${ui_src}/package.json" ]]; then
      echo "UI module '${module}' has no package.json, skipping build"
      continue
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
    mkdir -p "${ui_out}"
    cp -Rp "${ui_src}/dist/." "${ui_out}/"
  done
}

copy_plugins() {
  local staging_dir=$1
  local plugins_src="${HARNSGATEWAY_ROOT}/plugins"
  if [[ -d "${plugins_src}" ]]; then
    mkdir -p "${staging_dir}/gateway/plugins"
    cp -Rp "${plugins_src}/." "${staging_dir}/gateway/plugins/"
  fi
}

copy_ui() {
  local staging_dir=$1
  IFS=" " read -ra ui_modules <<< "${UI_MODULES}"
  for module in "${ui_modules[@]}"; do
    local ui_src="${HARNSGATEWAY_ROOT}/ui/${module}"
    local ui_cache_dir="${ui_cache}/${module}"
    local ui_dest="${staging_dir}/ui/${module}"
    if [[ "${module}" == "gateway-ui" ]]; then
      ui_dest="${staging_dir}/gateway/ui/gateway-ui"
    fi
    mkdir -p "${ui_dest}"
    if [[ -d "${ui_cache_dir}" ]]; then
      cp -Rp "${ui_cache_dir}/." "${ui_dest}/"
    elif [[ -d "${ui_src}/dist" ]]; then
      cp -Rp "${ui_src}/dist/." "${ui_dest}/"
    else
      echo "Warning: no dist found for UI module '${module}'"
    fi
  done
}

build_ui

# Build every platform in one build.sh invocation so version metadata and
# build timestamps are identical across the release artifacts.
export HARNSGATEWAY_BUILD_PLATFORMS="${platforms[*]}"
export GOFLAGS="${GOFLAGS:-} -trimpath -buildvcs=false"
"${HARNSGATEWAY_ROOT}/hack/make-rules/build.sh" "$@"

harnsgateway::version::get_version_vars
version="${HARNSGATEWAY_GIT_VERSION:-}"
if [[ -z "${version}" ]]; then
  version="$(git --work-tree "${HARNSGATEWAY_ROOT}" rev-parse --short HEAD 2>/dev/null || echo dev)"
fi

archive_names=()
for platform in "${platforms[@]}"; do
  target_os="${platform%%/*}"
  target_arch="${platform##*/}"
  bin_dir="${HARNSGATEWAY_OUTPUT_BINPATH}/${platform}"
  platform_name="${platform//\//_}"
  staging_root="${release_dir}/${platform_name}"
  staging_dir="${staging_root}/${PACKAGE_NAME}"

  echo "Packaging ${PACKAGE_NAME} for ${platform}"
  mkdir -p "${staging_dir}" "${staging_dir}/ui"

  for module in "${GO_MODULES[@]}"; do
    log_dir="log"
    if [[ "${module}" == "gateway" ]]; then
      log_dir="logs"
    fi
    mkdir -p "${staging_dir}/${module}/bin" "${staging_dir}/${module}/etc" "${staging_dir}/${module}/${log_dir}"
    bin_name="${module}"
    if [[ "${target_os}" == "windows" ]]; then
      bin_name="${bin_name}.exe"
    fi
    if [[ -f "${bin_dir}/${bin_name}" ]]; then
      cp -p "${bin_dir}/${bin_name}" "${staging_dir}/${module}/bin/"
    else
      echo "Warning: binary '${bin_name}' not found in ${bin_dir}"
    fi
    if [[ -d "${HARNSGATEWAY_ROOT}/etc/${module}" ]]; then
      cp -Rp "${HARNSGATEWAY_ROOT}/etc/${module}/." "${staging_dir}/${module}/etc/"
      if [[ "${module}" == "gateway" && -f "${HARNSGATEWAY_ROOT}/etc/gateway/config.package.yaml" ]]; then
        cp -p "${HARNSGATEWAY_ROOT}/etc/gateway/config.package.yaml" "${staging_dir}/gateway/etc/config.yaml"
        rm -f "${staging_dir}/gateway/etc/config.package.yaml"
        mkdir -p "${staging_dir}/gateway/data" "${staging_dir}/gateway/ui"
        cp -p "${HARNSGATEWAY_ROOT}/hack/install-gateway.sh" "${staging_dir}/gateway/install.sh"
        chmod +x "${staging_dir}/gateway/install.sh"
      fi
    fi
  done

  copy_plugins "${staging_dir}"
  copy_ui "${staging_dir}"

  archive_name="${PACKAGE_NAME}-${version}-${platform_name}.zip"
  archive="${release_dir}/${archive_name}"
  rm -f "${archive}"
  (
    cd "${staging_root}"
    zip -qr "${archive}" "${PACKAGE_NAME}"
  )
  archive_names+=("${archive}")
  echo "Packaged: ${archive}"
done

echo
echo "Release artifacts:"
for archive in "${archive_names[@]}"; do
  echo "  ${archive}"
done

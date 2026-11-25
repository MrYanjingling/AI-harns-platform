#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

HARNSGATEWAY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${HARNSGATEWAY_ROOT}/hack/lib/init.sh"

PACKAGE_NAME="${PACKAGE_NAME:-harns-gateway}"
OTA_PLATFORM="${OTA_PLATFORM:-linux/amd64}"
UI_BUILD="${UI_BUILD:-1}"
BACKEND_VERSION="${BACKEND_VERSION:-}"
FRONTEND_VERSION="${FRONTEND_VERSION:-}"
RELEASE_VERSION="${RELEASE_VERSION:-}"
LICENSE_SIGNING_KEY="${LICENSE_SIGNING_KEY:-${HARNSGATEWAY_ROOT}/cmd/licensegen/dev-private.pem}"
OTA_OUTPUT_DIR="${OTA_OUTPUT_DIR:-${HARNSGATEWAY_OUTPUT}/release/ota}"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$1" | awk '{print $1}'
}

target_os="${OTA_PLATFORM%%/*}"
target_arch="${OTA_PLATFORM##*/}"
if [[ -z "${target_os}" || -z "${target_arch}" ]]; then
  echo "OTA_PLATFORM must be GOOS/GOARCH, got: ${OTA_PLATFORM}" >&2
  exit 1
fi

harnsgateway::version::get_version_vars
if [[ -z "${RELEASE_VERSION}" ]]; then
  RELEASE_VERSION="${HARNSGATEWAY_GIT_VERSION:-3.0.0}"
fi
RELEASE_VERSION="${RELEASE_VERSION#v}"
if [[ -z "${BACKEND_VERSION}" ]]; then
  BACKEND_VERSION="${RELEASE_VERSION}"
fi
if [[ -z "${FRONTEND_VERSION}" ]]; then
  FRONTEND_VERSION="${RELEASE_VERSION}"
fi
export HARNSGATEWAY_GIT_VERSION="v${BACKEND_VERSION}"
if [[ ! -f "${LICENSE_SIGNING_KEY}" ]]; then
  echo "License/OTA signing private key not found: ${LICENSE_SIGNING_KEY}" >&2
  exit 1
fi

if [[ "${UI_BUILD}" != "0" ]]; then
  if ! command -v pnpm >/dev/null 2>&1; then
    echo "pnpm is required to build gateway-ui (set UI_BUILD=0 to reuse dist)" >&2
    exit 1
  fi
  (
    cd "${HARNSGATEWAY_ROOT}"
    pnpm --filter @harns/gateway-ui build
  )
fi

if [[ ! -d "${HARNSGATEWAY_ROOT}/ui/gateway-ui/dist" ]]; then
  echo "gateway-ui dist is missing" >&2
  exit 1
fi

work_dir="$(mktemp -d "${HARNSGATEWAY_OUTPUT}/ota-build.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
mkdir -p "${work_dir}/payload/bin" "${work_dir}/payload/ui/gateway-ui"

echo "Building gateway for ${OTA_PLATFORM}"
(
  cd "${HARNSGATEWAY_ROOT}"
  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
    go build -trimpath -buildvcs=false \
    -ldflags "$(harnsgateway::version::ldflags)" \
    -o "${work_dir}/payload/bin/gateway" ./cmd/gateway
)
cp -Rp "${HARNSGATEWAY_ROOT}/ui/gateway-ui/dist/." "${work_dir}/payload/ui/gateway-ui/"

package_name="${PACKAGE_NAME}-${RELEASE_VERSION}-${target_os}_${target_arch}.zip"
package_path="${work_dir}/${package_name}"
(
  cd "${work_dir}"
  zip -qr "${package_path}" payload
)

backend_sha="$(sha256_file "${work_dir}/payload/bin/gateway")"
frontend_sha="$(sha256_file "${work_dir}/payload/ui/gateway-ui/index.html")"
package_sha="$(sha256_file "${package_path}")"
package_size="$(wc -c < "${package_path}" | tr -d ' ')"

manifest_path="${work_dir}/manifest.json"
cat > "${manifest_path}" <<JSON
{
  "format": 1,
  "apiVersion": 1,
  "product": "harns-gateway",
  "releaseVersion": "${RELEASE_VERSION}",
  "os": "${target_os}",
  "arch": "${target_arch}",
  "backend": {
    "version": "${BACKEND_VERSION}",
    "sha256": "${backend_sha}"
  },
  "frontend": {
    "version": "${FRONTEND_VERSION}",
    "sha256": "${frontend_sha}"
  },
  "package": {
    "path": "${package_name}",
    "sha256": "${package_sha}",
    "size": ${package_size}
  },
  "notes": "${OTA_NOTES:-}"
}
JSON

openssl dgst -sha256 -sign "${LICENSE_SIGNING_KEY}" -out "${work_dir}/manifest.sig.raw" "${manifest_path}"
openssl base64 -A -in "${work_dir}/manifest.sig.raw" -out "${work_dir}/manifest.json.sig"

dest_dir="${OTA_OUTPUT_DIR}/${RELEASE_VERSION}/${target_os}_${target_arch}"
rm -rf "${dest_dir}"
mkdir -p "${dest_dir}"
cp -p "${manifest_path}" "${dest_dir}/manifest.json"
cp -p "${work_dir}/manifest.json.sig" "${dest_dir}/manifest.json.sig"
cp -p "${package_path}" "${dest_dir}/${package_name}"

echo "OTA package: ${dest_dir}/${package_name}"
echo "OTA manifest: ${dest_dir}/manifest.json"
echo "OTA signature: ${dest_dir}/manifest.json.sig"

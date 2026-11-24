#!/usr/bin/env bash

set -euo pipefail

TARGET_DIR="/data/harns-gateway"
SERVICE_NAME="harns-gateway"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Please run install.sh as root." >&2
  exit 1
fi

for item in etc logs plugins data bin ui; do
  if [[ ! -d "${SOURCE_DIR}/${item}" ]]; then
    echo "Missing install directory: ${SOURCE_DIR}/${item}" >&2
    exit 1
  fi
done

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop "${SERVICE_NAME}.service" 2>/dev/null || true
fi

mkdir -p "${TARGET_DIR}"
for item in etc logs plugins data bin ui; do
  mkdir -p "${TARGET_DIR}/${item}"
  cp -a "${SOURCE_DIR}/${item}/." "${TARGET_DIR}/${item}/"
done

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl is not available; files copied to ${TARGET_DIR}, service was not registered." >&2
  exit 0
fi

cat > "${SERVICE_FILE}" <<SERVICE
[Unit]
Description=Harns Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${TARGET_DIR}/etc
ExecStart=${TARGET_DIR}/bin/gateway -c ../etc/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"

echo "Installed ${TARGET_DIR}"
echo "Service ${SERVICE_NAME}.service is enabled and running."

#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/var/www/csu-star-backend"
APP_NAME="csu-star-backend"
BRANCH="main"
BACKUP_DIR="$HOME/backup"

DATE_TAG="$(date +%Y%m%d)"
TIME_TAG="$(date +%H%M%S)"
BACKUP_FILE="${BACKUP_DIR}/${APP_NAME}_${DATE_TAG}_${TIME_TAG}"

# Ensure Go is resolvable in non-interactive SSH sessions.
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

echo "[deploy] start version:2026-04-10-1"
echo "[deploy] user: $(whoami)"
echo "[deploy] PATH: ${PATH}"
mkdir -p "${BACKUP_DIR}"

if [[ ! -d "${APP_DIR}" ]]; then
  echo "[deploy] error: app dir not found: ${APP_DIR}" >&2
  exit 1
fi

cd "${APP_DIR}"

echo "[deploy] sync code from origin/${BRANCH}"
git fetch origin "${BRANCH}"
git checkout "${BRANCH}"
echo "[deploy] reset local changes to keep deploy workspace clean"
git reset --hard "origin/${BRANCH}"
git clean -fd

echo "[deploy] run tests"
command -v go >/dev/null 2>&1 || {
  echo "[deploy] error: go command not found. PATH=${PATH}" >&2
  exit 127
}
go version
go test ./...

if [[ -f "${APP_NAME}" ]]; then
  cp "${APP_NAME}" "${BACKUP_FILE}"
  echo "[deploy] backup created: ${BACKUP_FILE}"
else
  echo "[deploy] old binary not found, skip backup"
fi

echo "[deploy] cleanup backups older than 30 days"
find "${BACKUP_DIR}" -maxdepth 1 -type f -name "${APP_NAME}_*" -mtime +30 -delete

echo "[deploy] build linux binary"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${APP_NAME}" ./cmd/main.go

if [[ ! -f "${APP_NAME}" ]]; then
  echo "[deploy] error: build output not found: ${APP_NAME}" >&2
  exit 1
fi

if [[ ! -x "${APP_NAME}" ]]; then
  chmod +x "${APP_NAME}"
fi

echo "[deploy] restart service"
sudo pm2 restart "${APP_NAME}"

echo "[deploy] wait 5s then show recent logs"
sleep 5
sudo pm2 logs "${APP_NAME}" --lines 20 --nostream

echo "[deploy] done"

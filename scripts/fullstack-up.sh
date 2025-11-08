#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

compose_file="${PROJECT_ROOT}/deploy/fullstack/docker-compose.yml"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required to run the fullstack environment." >&2
  exit 1
fi

echo "Starting Synaptica fullstack stack..."
docker compose -f "$compose_file" up --build

#!/bin/sh

set -eu

if command -v docker-compose >/dev/null 2>&1; then
  compose_cmd="docker-compose"
else
  compose_cmd="docker compose"
fi

required_files="
docker-compose.yml
Makefile
infra/nginx/nginx.conf
infra/traefik/traefik.yml
infra/postgres/init/001-create-schemas.sql
infra/rabbitmq/rabbitmq.conf
infra/rabbitmq/definitions.json
infra/monitoring/prometheus.yml
recipients/go.mod
recipients/cmd/api/main.go
recipients/Dockerfile
sources/go.mod
sources/cmd/api/main.go
sources/Dockerfile
notifications-service/go.mod
notifications-service/cmd/api/main.go
notifications-service/Dockerfile
delivery/go.mod
delivery/cmd/consumer/main.go
delivery/Dockerfile
frontend/package.json
frontend/public/index.html
frontend/Dockerfile
README.md
docs/platform-baseline.md
.github/workflows/ci.yml
scripts/lint-platform.sh
scripts/validate-contracts.sh
"

missing=0
for file in $required_files; do
  if [ ! -f "$file" ]; then
    printf 'missing required file: %s\n' "$file"
    missing=1
  fi
done

if [ "$missing" -ne 0 ]; then
  exit 1
fi

$compose_cmd config >/dev/null

printf 'platform structure checks passed\n'

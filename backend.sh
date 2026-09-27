#!/usr/bin/env bash
# One-command control for the near-stocks local backend stack. Run this from
# anywhere inside this repo, e.g. `./backend.sh up`.
# Usage: backend.sh [up|down|status|build]   (default: up)
#   up     - start docker (postgres+redis), then all 8 services, in dependency order. Safe to re-run.
#   down   - stop all services and docker containers.
#   status - show what's running.
#   build  - rebuild every service binary from source into bin/.
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
d="$repo/local"
cmd="${1:-up}"

# order matters: oracle/balance-server/engine before api-server; api-server before the rest
SERVICES=(nsoracle balance-server engine api-server cron-server indexer-server liquidation amm)

BUILD_PATHS=(
  "nsoracle:./nearchain/cmd/nsoracle"
  "api-server:./services/api-server/cmd"
  "engine:./services/engine/cmd"
  "balance-server:./services/balance-server/cmd"
  "liquidation:./services/liquidation/cmd"
  "cron-server:./services/cron-server/cmd"
  "indexer-server:./services/indexer-server/cmd"
  "amm:./services/amm/cmd"
  "nstestnet:./nearchain/cmd/nstestnet"
)

start_docker() {
  if docker inspect ns-postgres >/dev/null 2>&1; then
    docker start ns-postgres >/dev/null
  else
    echo "creating ns-postgres (was missing)"
    docker run -d --name ns-postgres -p 55432:5432 \
      -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=nearstocks \
      -v ns-pgdata:/var/lib/postgresql/data postgres:16-alpine >/dev/null
  fi
  if docker inspect ns-redis >/dev/null 2>&1; then
    docker start ns-redis >/dev/null
  else
    echo "creating ns-redis (was missing)"
    docker run -d --name ns-redis -p 56379:6379 redis:7-alpine >/dev/null
  fi
  echo -n "waiting for postgres/redis..."
  until docker exec ns-postgres pg_isready -U postgres >/dev/null 2>&1; do echo -n "."; sleep 1; done
  until docker exec ns-redis redis-cli ping >/dev/null 2>&1; do echo -n "."; sleep 1; done
  echo " ready"
}

case "$cmd" in
  build)
    cd "$repo"
    for entry in "${BUILD_PATHS[@]}"; do
      svc="${entry%%:*}"; path="${entry#*:}"
      echo "building $svc"
      go build -o "$d/bin/$svc" "$path"
    done
    echo "build complete"
    ;;
  up)
    start_docker
    for svc in "${SERVICES[@]}"; do
      if pgrep -f "$d/bin/$svc" >/dev/null 2>&1; then
        echo "$svc: already running"
      else
        "$d/run.sh" start "$svc"
        sleep 1
      fi
    done
    echo "backend up. api-server: http://localhost:8080"
    ;;
  down)
    for ((i=${#SERVICES[@]}-1; i>=0; i--)); do
      "$d/run.sh" stop "${SERVICES[i]}" || true
    done
    docker stop ns-postgres ns-redis >/dev/null
    echo "backend down"
    ;;
  status)
    docker ps -a --filter name=ns-postgres --filter name=ns-redis --format '{{.Names}}: {{.Status}}'
    for svc in "${SERVICES[@]}"; do
      if pgrep -f "$d/bin/$svc" >/dev/null 2>&1; then
        echo "$svc: running"
      else
        echo "$svc: stopped"
      fi
    done
    ;;
  *)
    echo "usage: $0 [up|down|status|build]" >&2
    exit 1
    ;;
esac

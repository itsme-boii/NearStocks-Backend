#!/usr/bin/env bash
# One-command control for the near-stocks local backend stack. Run this from
# anywhere inside this repo, e.g. `./backend.sh up`.
# Usage: backend.sh [up|down|status|build|up-mainnet|down-mainnet|status-mainnet]   (default: up)
#   up             - start docker (postgres+redis), then the testnet-profile services. Safe to re-run.
#   down           - stop the testnet-profile services and their docker containers.
#   status         - show what's running (testnet profile + the shared oracle).
#   build          - rebuild every service binary from source into bin/, and services/oracle's dist/.
#   up-mainnet     - start the mainnet-profile services against REAL NEAR MAINNET. Real money the
#                    moment cron-server starts (it signs with the real sequencer key and submits
#                    real batches). Requires ./backend.sh up to have built bin/ already, and the
#                    shared "oracle" service (started by the testnet profile) to be running.
#   down-mainnet   - stop the mainnet-profile services and their docker containers.
#   status-mainnet - show what's running in the mainnet profile.
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
d="$repo/local"
cmd="${1:-up}"

# order matters: oracle/balance-server/engine before api-server; api-server before the rest
# "oracle" is the real production price aggregator (services/oracle, NestJS — vendored from the
# standalone 100exhange-oracle repo); "nsoracle" is the Yahoo Finance test stand-in still used as
# ORACLE_SERVER_URL's default in local/common.env. Both run side by side — nothing points at
# "oracle" by default yet, point ORACLE_SERVER_URL at http://localhost:8097 to use it instead.
# Both are shared with the mainnet profile (they're just price sources, not network-specific) — they
# only ever run under the testnet profile's `up`/`down`/`status`, never duplicated for mainnet.
SERVICES=(nsoracle oracle balance-server engine api-server cron-server indexer-server liquidation amm)

# mainnet profile: no amm (no market-maker — the mainnet test trades two real wallets against each
# other directly) and no oracle/nsoracle (shared with the testnet profile, see above).
MAINNET_SERVICES=(balance-server engine api-server cron-server indexer-server liquidation)

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

# Every service (Go or the oracle's node process) is launched via `exec -a nearstocks-<profile>-<svc>`
# (see run.sh) so pgrep/pkill can tell apart a testnet vs. mainnet copy of the exact same binary.
match_pattern() {
  local svc="$1" profile="$2"
  if [[ "$svc" == "oracle" || "$svc" == "nsoracle" ]]; then
    echo "nearstocks-testnet-$svc" # shared: always the testnet-profile process name regardless of $2
  else
    echo "nearstocks-$profile-$svc"
  fi
}

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

start_docker_mainnet() {
  if docker inspect ns-postgres >/dev/null 2>&1; then
    docker start ns-postgres >/dev/null
  else
    echo "ns-postgres is missing — run ./backend.sh up first (mainnet reuses it, just a different database)" >&2
    exit 1
  fi
  docker exec ns-postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='nearstocksmainnet'" | grep -q 1 \
    || docker exec ns-postgres createdb -U postgres nearstocksmainnet
  if docker inspect ns-redis-mainnet >/dev/null 2>&1; then
    docker start ns-redis-mainnet >/dev/null
  else
    echo "creating ns-redis-mainnet (was missing)"
    docker run -d --name ns-redis-mainnet -p 56380:6379 redis:7-alpine >/dev/null
  fi
  echo -n "waiting for postgres/redis-mainnet..."
  until docker exec ns-postgres pg_isready -U postgres >/dev/null 2>&1; do echo -n "."; sleep 1; done
  until docker exec ns-redis-mainnet redis-cli ping >/dev/null 2>&1; do echo -n "."; sleep 1; done
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
    echo "building oracle"
    (cd "$repo/services/oracle" && npm install --no-audit --no-fund >/dev/null && npm run build)
    echo "build complete"
    ;;
  up)
    start_docker
    for svc in "${SERVICES[@]}"; do
      if pgrep -f "$(match_pattern "$svc" testnet)" >/dev/null 2>&1; then
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
      if pgrep -f "$(match_pattern "$svc" testnet)" >/dev/null 2>&1; then
        echo "$svc: running"
      else
        echo "$svc: stopped"
      fi
    done
    ;;
  up-mainnet)
    if ! pgrep -f "$(match_pattern oracle testnet)" >/dev/null 2>&1; then
      echo "the shared oracle service isn't running — start it first: ./backend.sh up (or ./local/run.sh start oracle)" >&2
      exit 1
    fi
    echo "REAL MAINNET, REAL MONEY: cron-server will sign with the real sequencer key and submit real batches." >&2
    start_docker_mainnet
    for svc in "${MAINNET_SERVICES[@]}"; do
      if pgrep -f "$(match_pattern "$svc" mainnet)" >/dev/null 2>&1; then
        echo "$svc: already running"
      else
        "$d/run.sh" start "$svc" mainnet.env
        sleep 1
      fi
    done
    echo "mainnet profile up. api-server: http://localhost:8190"
    ;;
  down-mainnet)
    for ((i=${#MAINNET_SERVICES[@]}-1; i>=0; i--)); do
      "$d/run.sh" stop "${MAINNET_SERVICES[i]}" mainnet.env || true
    done
    echo "mainnet profile down (ns-postgres/ns-redis-mainnet left running — shared/persistent; docker stop ns-redis-mainnet if you want them down too)"
    ;;
  status-mainnet)
    docker ps -a --filter name=ns-redis-mainnet --format '{{.Names}}: {{.Status}}'
    for svc in "${MAINNET_SERVICES[@]}"; do
      if pgrep -f "$(match_pattern "$svc" mainnet)" >/dev/null 2>&1; then
        echo "$svc: running"
      else
        echo "$svc: stopped"
      fi
    done
    ;;
  *)
    echo "usage: $0 [up|down|status|build|up-mainnet|down-mainnet|status-mainnet]" >&2
    exit 1
    ;;
esac

#!/usr/bin/env bash
# Gate G1: Go (reference) regenerates, Rust and TypeScript independently re-derive.
# Gate G3 (core): Go's production ledger code regenerates parity.json, the contract replays it.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT

echo "== Go: regenerate and diff against committed vectors"
(cd "$here/.." && go run ./nearchain/cmd/genvectors -out "$tmp" >/dev/null)
diff -r -x ts -x '*.sh' -x '*.md' -x parity.json -x events.json "$here" "$tmp" && echo "Go output matches committed vectors"

echo "== Rust"
(cd "$here/../near-stocks-contracts" && cargo test -q -p near-stocks-core --test vectors --test math --test payloads)

echo "== TypeScript"
(cd "$here/ts" && ([ -d node_modules ] || npm ci --silent) && npx tsx verify.ts)

echo "G1 vectors: PASS"

echo "== G3: Go ledger regenerates parity.json, contract replays it"
(cd "$here/../services/balance-server" && PARITY_OUT="$tmp/parity.json" go test -count=1 ./tests/ -run TestGenerateParity >/dev/null)
cmp "$here/parity.json" "$tmp/parity.json" && echo "Go parity scenarios match committed parity.json"
(cd "$here/../near-stocks-contracts" && cargo test -q -p near-stocks-core --test parity)
echo "G3 core parity: PASS"

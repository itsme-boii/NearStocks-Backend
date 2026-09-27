#!/usr/bin/env bash
# Runs the near-workspaces sandbox tests. The prebuilt nearcore sandbox needs glibc >= 2.38;
# on older hosts (e.g. Ubuntu/Pop!_OS 22.04, glibc 2.35) the tests run inside ubuntu:24.04.
set -euo pipefail
cd "$(dirname "$0")/.."
root="$(cd .. && pwd)"
(cd core && cargo near build non-reproducible-wasm >/dev/null 2>&1)
(cd mock-ft && cargo near build non-reproducible-wasm >/dev/null 2>&1)
(cd logx-token && cargo near build non-reproducible-wasm >/dev/null 2>&1)
bins=$(cargo test -p near-stocks-core --test sandbox_flows --no-run --message-format=json 2>/dev/null \
  | python3 -c "import sys,json
for l in sys.stdin:
    try: d=json.loads(l)
    except Exception: continue
    if d.get('executable') and '/sandbox_' in d['executable']: print(d['executable'])")
sandbox=$(find "$PWD"/target/debug/build -path "*/out/.near/near-sandbox-*/near-sandbox" -type f | head -1)
glibc=$(ldd --version 2>&1 | awk 'NR==1{print $NF}')
run() { for b in $bins; do "$b" --ignored --nocapture; done; }
if [ "$(printf '%s\n2.38\n' "$glibc" | sort -V | head -1)" = "2.38" ]; then
  NEAR_SANDBOX_BIN_PATH="$sandbox" run
else
  docker run --rm -v "$root:$root" -w "$PWD" -e NEAR_SANDBOX_BIN_PATH="$sandbox" ubuntu:24.04 \
    bash -c "$(declare -f run); bins='$bins'; run"
fi

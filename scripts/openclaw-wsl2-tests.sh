#!/usr/bin/env bash
set -euo pipefail

if [[ "${CRABBOX_LIVE:-}" != "1" ]]; then
  echo "set CRABBOX_LIVE=1 to create a Windows WSL2 lease" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cb="${CRABBOX_BIN:-$root/bin/crabbox}"
repo="${CRABBOX_OPENCLAW_REPO:-${CRABBOX_LIVE_REPO:-/Users/steipete/Projects/openclaw}}"
lease="${CRABBOX_OPENCLAW_WSL2_ID:-}"
provider="${CRABBOX_OPENCLAW_WSL2_PROVIDER:-aws}"
class="${CRABBOX_OPENCLAW_WSL2_CLASS:-beast}"
market="${CRABBOX_OPENCLAW_WSL2_MARKET:-on-demand}"
idle_timeout="${CRABBOX_OPENCLAW_WSL2_IDLE_TIMEOUT:-240m}"
hydrate_wait="${CRABBOX_OPENCLAW_WSL2_HYDRATE_WAIT:-45m}"
keep_alive="${CRABBOX_OPENCLAW_WSL2_KEEP_ALIVE_MINUTES:-240}"
reclaim="${CRABBOX_OPENCLAW_WSL2_RECLAIM:-0}"
stop_after="${CRABBOX_OPENCLAW_WSL2_STOP:-0}"
test_command="${CRABBOX_OPENCLAW_TEST_COMMAND:-corepack enable && pnpm install --frozen-lockfile && CI=1 NODE_OPTIONS=--max-old-space-size=4096 OPENCLAW_TEST_PROJECTS_PARALLEL=6 OPENCLAW_VITEST_MAX_WORKERS=1 pnpm test}"
crabbox_target_args=(
  --provider "$provider"
  --target windows
  --windows-mode wsl2
)
export CRABBOX_PROVIDER="$provider"
export CRABBOX_TARGET=windows
export CRABBOX_WINDOWS_MODE=wsl2

if [[ ! -d "$repo/.git" ]]; then
  echo "OpenClaw repo not found: $repo" >&2
  echo "set CRABBOX_OPENCLAW_REPO=/path/to/openclaw" >&2
  exit 2
fi

run_in_repo() {
  (cd "$repo" && "$@")
}

run_crabbox_wsl2() {
  (
    cd "$repo"
    CRABBOX_PROVIDER="$provider" CRABBOX_TARGET=windows CRABBOX_WINDOWS_MODE=wsl2 "$cb" "$@"
  )
}

extract_lease_or_slug() {
  sed -n 's/.*slug=\([^ ]*\).*/\1/p' | head -1
}

if [[ -z "$lease" ]]; then
  if out="$(run_in_repo "$cb" warmup \
    "${crabbox_target_args[@]}" \
    --class "$class" \
    --market "$market" \
    --idle-timeout "$idle_timeout" \
    --timing-json 2>&1)"; then
    printf '%s\n' "$out"
  else
    rc=$?
    printf '%s\n' "$out"
    exit "$rc"
  fi
  lease="$(printf '%s\n' "$out" | extract_lease_or_slug)"
  if [[ -z "$lease" ]]; then
    lease="$(printf '%s\n' "$out" | sed -n 's/.*\(cbx_[a-f0-9]\{12\}\).*/\1/p' | head -1)"
  fi
  test -n "$lease"
fi

cleanup() {
  if [[ "$stop_after" == "1" && -n "$lease" ]]; then
    run_crabbox_wsl2 stop "$lease" || true
  fi
}
trap cleanup EXIT

hydrate_args=()
if [[ "$reclaim" == "1" ]]; then
  hydrate_args+=(--reclaim)
fi

run_crabbox_wsl2 actions hydrate \
  --id "$lease" \
  "${hydrate_args[@]}" \
  --wait-timeout "$hydrate_wait" \
  --keep-alive-minutes "$keep_alive" \
  --timing-json

run_crabbox_wsl2 run --id "$lease" --shell -- "$test_command"

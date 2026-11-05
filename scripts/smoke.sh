#!/usr/bin/env bash
set -euo pipefail

# Run after make build. API_BASE_URL at build time must match the selected API port.
# Optional ports allow isolated checks alongside a running development stack.
api_port="${SCHEWISE_SMOKE_API_PORT:-8080}"
web_port="${SCHEWISE_SMOKE_WEB_PORT:-3000}"
export SCHEWISE_SMOKE_API_PORT="$api_port" SCHEWISE_SMOKE_WEB_PORT="$web_port"
# Check both direct API traffic and the production Next.js rewrite.
for port in "$api_port" "$web_port"; do
  if curl --silent --output /dev/null --max-time 1 "http://127.0.0.1:${port}"; then
    echo "Port ${port} is already serving HTTP. Stop local servers before smoke testing." >&2
    exit 1
  fi
done

api_pid=""
web_pid=""
cleanup() {
  if [[ -n "$web_pid" ]]; then kill "$web_pid" 2>/dev/null || true; fi
  if [[ -n "$api_pid" ]]; then kill "$api_pid" 2>/dev/null || true; fi
  wait 2>/dev/null || true
}
trap cleanup EXIT

SCHEWISE_HTTP_ADDR="127.0.0.1:${api_port}" ./bin/schewise-api &
api_pid=$!
NEXT_TELEMETRY_DISABLED=1 node apps/web/node_modules/next/dist/bin/next start apps/web --hostname 127.0.0.1 --port "$web_port" &
web_pid=$!

ready=false
for attempt in {1..60}; do
  kill -0 "$api_pid"
  kill -0 "$web_pid"
  if curl --fail --silent --output /dev/null "http://127.0.0.1:${web_port}/api/v1/health"; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo "Timed out waiting for the production web/API connection." >&2
  exit 1
fi

node --input-type=module <<'JS'
import assert from 'node:assert/strict';

const api = `http://127.0.0.1:${process.env.SCHEWISE_SMOKE_API_PORT}`;
const web = `http://127.0.0.1:${process.env.SCHEWISE_SMOKE_WEB_PORT}`;
for (const origin of [api, web]) {
  const response = await fetch(`${origin}/api/v1/health`);
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { status: 'ok', service: 'schewise-api' });
  const mutation = await fetch(`${origin}/api/v1/health`, { method: 'POST' });
  assert.equal(mutation.status, 405);
}
const page = await fetch(web, { redirect: 'manual' });
assert.equal(page.status, 307);
console.log('Smoke checks passed: API contract, protected page redirect, and web-to-API proxy.');
JS

SCHEWISE_SMOKE_BASE_URL="http://127.0.0.1:${web_port}" node scripts/auth-smoke.mjs

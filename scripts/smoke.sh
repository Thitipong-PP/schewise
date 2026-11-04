#!/usr/bin/env bash
set -euo pipefail

# Run from the repository root after `make build`. Use the default API_BASE_URL.
# Check both direct API traffic and the production Next.js rewrite.
for port in 8080 3000; do
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

SCHEWISE_HTTP_ADDR=127.0.0.1:8080 ./bin/schewise-api &
api_pid=$!
NEXT_TELEMETRY_DISABLED=1 node apps/web/node_modules/next/dist/bin/next start apps/web --hostname 127.0.0.1 --port 3000 &
web_pid=$!

ready=false
for attempt in {1..60}; do
  kill -0 "$api_pid"
  kill -0 "$web_pid"
  if curl --fail --silent --output /dev/null http://127.0.0.1:3000/api/v1/health; then
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

for (const origin of ['http://127.0.0.1:8080', 'http://127.0.0.1:3000']) {
  const response = await fetch(`${origin}/api/v1/health`);
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { status: 'ok', service: 'schewise-api' });
  const mutation = await fetch(`${origin}/api/v1/health`, { method: 'POST' });
  assert.equal(mutation.status, 405);
}
const page = await fetch('http://127.0.0.1:3000', { redirect: 'manual' });
assert.equal(page.status, 307);
console.log('Smoke checks passed: API contract, protected page redirect, and web-to-API proxy.');
JS

SCHEWISE_SMOKE_BASE_URL=http://127.0.0.1:3000 node scripts/auth-smoke.mjs

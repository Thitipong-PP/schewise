#!/usr/bin/env bash
set -euo pipefail
mkdir -p .local
for attempt in {1..20}; do
  if docker compose cp gateway:/data/caddy/pki/authorities/local/root.crt .local/root.crt >/dev/null 2>&1; then
    echo "Exported the local HTTPS public CA to .local/root.crt."
    exit 0
  fi
  sleep 1
done
echo "Could not export the local CA. Start the HTTPS gateway first." >&2
exit 1

#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$HERE")"
COMPOSE_FILE="$PROJECT_DIR/compose.yml"

# Ensure data directories exist so volumes mount cleanly.
for node in redis-01 redis-02 redis-03 redis-04 redis-05 redis-06; do
  mkdir -p "$PROJECT_DIR/data/$node"
done

pushd "$PROJECT_DIR" > /dev/null

echo "[*] Starting Redis containers..."
docker compose -f "$COMPOSE_FILE" up -d

NODES=(
  "redis-01:6379"
  "redis-02:6379"
  "redis-03:6379"
  "redis-04:6379"
  "redis-05:6379"
  "redis-06:6379"
)

# Wait for each node to accept connections.
echo "[*] Waiting for Redis nodes to become ready..."
for node in "${NODES[@]}"; do
  host="${node%%:*}"
  until docker exec "$host" redis-cli ping >/dev/null 2>&1; do
    sleep 0.5
  done
  echo "  - $host ready"
done

# Create the cluster with 3 masters and 3 replicas (replica per master).
CREATE_CMD=(redis-cli --cluster create "${NODES[@]}" --cluster-replicas 1)

echo "[*] Creating cluster topology..."
if ! docker exec -i redis-01 "${CREATE_CMD[@]}" <<<"yes"; then
  echo "[!] Cluster create command failed (maybe already initialised?)."
  echo "[!] You can inspect the cluster with: docker exec -it redis-01 redis-cli cluster nodes"
  exit 1
fi

echo "[*] Cluster created successfully. Verifying..."
docker exec redis-01 redis-cli --cluster check redis-01:6379

echo "[*] Done. Use 'docker compose -f $COMPOSE_FILE down -v' to tear everything down."

popd > /dev/null

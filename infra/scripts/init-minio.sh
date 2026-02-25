#!/usr/bin/env bash
set -euo pipefail

# Wait for MinIO to be healthy, then create the ser-bundles bucket.
# Idempotent — safe to run multiple times.

MINIO_HOST="${MINIO_HOST:-http://localhost:9000}"
MINIO_USER="${MINIO_ROOT_USER:-minioadmin}"
MINIO_PASS="${MINIO_ROOT_PASSWORD:-minioadmin}"
BUCKET="ser-bundles"

echo "Waiting for MinIO at ${MINIO_HOST}..."
until curl -sf "${MINIO_HOST}/minio/health/live" > /dev/null 2>&1; do
  sleep 1
done
echo "MinIO is ready."

# Configure mc alias
mc alias set local "${MINIO_HOST}" "${MINIO_USER}" "${MINIO_PASS}" > /dev/null 2>&1

# Create bucket (--ignore-existing makes this idempotent)
if mc ls "local/${BUCKET}" > /dev/null 2>&1; then
  echo "Bucket '${BUCKET}' already exists — skipping."
else
  mc mb "local/${BUCKET}"
  echo "Bucket '${BUCKET}' created."
fi

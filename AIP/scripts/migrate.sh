#!/usr/bin/env bash
# Aplica migrations SQL em ordem (001 → 005).
# Requer: psql no PATH e variáveis AIP_DB_* (ou DATABASE_URL).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MIG="$ROOT/migrations"

if [[ -n "${DATABASE_URL:-}" ]]; then
  PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)
else
  host="${AIP_DB_HOST:-localhost}"
  port="${AIP_DB_PORT:-5432}"
  user="${AIP_DB_USER:-postgres}"
  pass="${AIP_DB_PASSWORD:-postgres}"
  name="${AIP_DB_NAME:-aip}"
  export PGPASSWORD="$pass"
  PSQL=(psql -h "$host" -p "$port" -U "$user" -d "$name" -v ON_ERROR_STOP=1)
fi

echo "A aplicar migrations em $MIG ..."
for f in "$MIG"/00*.sql; do
  echo "→ $(basename "$f")"
  "${PSQL[@]}" -f "$f"
done
echo "Migrations concluídas."

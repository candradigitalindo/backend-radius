#!/usr/bin/env bash
# Restore hasil backup.sh ke server (baru). Prasyarat: Docker terpasang, kelima repo
# sudah di-clone di $BASE_DIR (install.sh mengurus itu). Script ini:
#   1. taruh kembali file .env + keys WireGuard
#   2. isi volume Docker dari arsip (sebelum container dibuat)
#   3. nyalakan database saja, load dump Postgres/Mongo/MySQL
#   4. build & up seluruh stack (backend -> landing -> proxy)
#
# Pemakaian: ./restore.sh /path/radius-backup-YYYYmmdd-HHMMSS.tar.gz
set -euo pipefail

BACKUP_FILE="${1:?pemakaian: restore.sh <tarball-backup>}"
BASE_DIR="${BASE_DIR:-/home/daniswara/radius-server}"

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
echo "[1/6] Ekstrak backup..."
tar xzf "$BACKUP_FILE" -C "$STAGE"
cat "$STAGE/meta.txt" || true

echo "[2/6] Pasang konfigurasi & kunci..."
install -m 600 "$STAGE/config/root.env"     "$BASE_DIR/.env"
install -m 600 "$STAGE/config/backend.env"  "$BASE_DIR/backend-radius/.env"
install -m 600 "$STAGE/config/frontend.env" "$BASE_DIR/frontend-radius/.env"
install -m 600 "$STAGE/config/wa.env"       "$BASE_DIR/wa-radius/.env"
tar xzf "$STAGE/config/keys.tar.gz" -C "$BASE_DIR/backend-radius"

echo "[3/6] Isi volume Docker dari arsip..."
docker network inspect web >/dev/null 2>&1 || docker network create web
# Streaming via stdin (bukan bind-mount) supaya tidak tergantung kesamaan path host/container.
for VOL in backend-radius_redisdata backend-radius_n8n_data backend-radius_storage landing-page_landing_wp_data; do
  docker volume create "$VOL" >/dev/null
  docker run --rm -i -v "$VOL":/data alpine \
    sh -c 'rm -rf /data/* /data/..?* /data/.[!.]* 2>/dev/null; tar xzf - -C /data' \
    < "$STAGE/volumes/${VOL}.tar.gz"
done

DB_USER=$(grep -E '^DB_USER=' "$BASE_DIR/backend-radius/.env" | cut -d= -f2- || true); DB_USER="${DB_USER:-backend_radius}"
DB_NAME=$(grep -E '^DB_NAME=' "$BASE_DIR/backend-radius/.env" | cut -d= -f2- || true); DB_NAME="${DB_NAME:-backend_radius}"

echo "[4/6] Nyalakan database & load dump..."
docker compose -f "$BASE_DIR/backend-radius/docker-compose.yml" up -d db acs_mongo
docker compose -f "$BASE_DIR/landing-page/docker-compose.yml" up -d landing_db

until docker exec radius_db pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do sleep 2; done
docker cp "$STAGE/dumps/postgres.dump" radius_db:/tmp/restore.dump
docker exec radius_db pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner /tmp/restore.dump || true
docker exec radius_db rm -f /tmp/restore.dump
docker exec -i radius_acs_mongo mongorestore --archive --gzip --drop --quiet < "$STAGE/dumps/mongo.archive.gz"

until docker exec landing_db mysqladmin ping -h localhost -ulanding_user -planding_pass_2024 --silent >/dev/null 2>&1; do sleep 2; done
gunzip -c "$STAGE/dumps/landing_mysql.sql.gz" | docker exec -i landing_db mysql -ulanding_user -planding_pass_2024 landing_wp

echo "[5/6] Build & nyalakan seluruh stack..."
docker compose -f "$BASE_DIR/backend-radius/docker-compose.yml" up -d --build
docker compose -f "$BASE_DIR/landing-page/docker-compose.yml" up -d
docker compose -f "$BASE_DIR/nginx-proxy/docker-compose.yml" up -d

echo "[6/6] Verifikasi..."
sleep 5
docker ps --format '{{.Names}}\t{{.Status}}' | sort
echo
echo "Restore selesai. JANGAN LUPA (lihat deploy/README.md):"
echo "  - Set VPN_PUBLIC_IP di backend-radius/.env kalau IP server berubah, lalu: docker compose up -d app"
echo "  - Arahkan DNS app/acs/dradius.net ke IP server ini"
echo "  - Update endpoint WireGuard di router bila IP berubah"

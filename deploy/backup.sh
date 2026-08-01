#!/usr/bin/env bash
# Backup lengkap stack D Radius: dump database + volume Docker + file konfigurasi.
# Hasil: satu tarball terenkripsi-permission (600) yang bisa langsung dipakai restore.sh
# di server baru. Aman dijalankan saat stack hidup (pg_dump/mongodump/mysqldump konsisten;
# volume redis/n8n/storage/wp dicopy live — lihat catatan di deploy/README.md).
#
# Pemakaian:
#   ./backup.sh                       # tarball ke /var/backups/radius (butuh root) atau $HOME/backups/radius
#   BACKUP_DIR=/mnt/usb ./backup.sh   # tujuan custom
set -euo pipefail

BASE_DIR="${BASE_DIR:-/home/daniswara/radius-server}"
if [[ -z "${BACKUP_DIR:-}" ]]; then
  if [[ $(id -u) -eq 0 ]]; then BACKUP_DIR=/var/backups/radius; else BACKUP_DIR="$HOME/backups/radius"; fi
fi
RETENTION="${RETENTION:-7}"   # simpan N tarball terakhir

TS=$(date +%Y%m%d-%H%M%S)
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE"/{dumps,volumes,config} "$BACKUP_DIR"

# Kredensial mengikuti default docker-compose.yml; override lewat backend-radius/.env kalau ada.
DB_USER="${DB_USER:-backend_radius}"
DB_NAME="${DB_NAME:-backend_radius}"
if [[ -f "$BASE_DIR/backend-radius/.env" ]]; then
  DB_USER=$(grep -E '^DB_USER=' "$BASE_DIR/backend-radius/.env" | cut -d= -f2- || true); DB_USER="${DB_USER:-backend_radius}"
  DB_NAME=$(grep -E '^DB_NAME=' "$BASE_DIR/backend-radius/.env" | cut -d= -f2- || true); DB_NAME="${DB_NAME:-backend_radius}"
fi

echo "[1/4] Dump database..."
docker exec radius_db pg_dump -U "$DB_USER" -Fc "$DB_NAME" > "$STAGE/dumps/postgres.dump"
docker exec radius_acs_mongo mongodump --archive --gzip --quiet > "$STAGE/dumps/mongo.archive.gz"
docker exec landing_db sh -c 'exec mysqldump -ulanding_user -planding_pass_2024 landing_wp' 2>/dev/null | gzip > "$STAGE/dumps/landing_mysql.sql.gz"

echo "[2/4] Arsip volume Docker..."
# Streaming via stdout (bukan bind-mount) supaya tidak tergantung kesamaan path host/container.
for VOL in backend-radius_redisdata backend-radius_n8n_data backend-radius_storage landing-page_landing_wp_data; do
  docker run --rm -v "$VOL":/data:ro alpine tar czf - -C /data . > "$STAGE/volumes/${VOL}.tar.gz"
done

echo "[3/4] Salin konfigurasi & kunci..."
cp "$BASE_DIR/.env"                 "$STAGE/config/root.env"
cp "$BASE_DIR/backend-radius/.env"  "$STAGE/config/backend.env"
cp "$BASE_DIR/frontend-radius/.env" "$STAGE/config/frontend.env"
cp "$BASE_DIR/wa-radius/.env"       "$STAGE/config/wa.env"
tar czf "$STAGE/config/keys.tar.gz" -C "$BASE_DIR/backend-radius" keys

{
  echo "created: $TS"
  echo "host: $(hostname)"
  for R in backend-radius frontend-radius landing-page nginx-proxy wa-radius; do
    echo "$R: $(git -C "$BASE_DIR/$R" rev-parse --short HEAD 2>/dev/null || echo '-')"
  done
} > "$STAGE/meta.txt"

echo "[4/4] Sanity check & bungkus tarball..."
for F in dumps/postgres.dump dumps/mongo.archive.gz dumps/landing_mysql.sql.gz \
         config/root.env config/backend.env config/frontend.env config/wa.env config/keys.tar.gz \
         volumes/backend-radius_redisdata.tar.gz volumes/backend-radius_n8n_data.tar.gz \
         volumes/backend-radius_storage.tar.gz volumes/landing-page_landing_wp_data.tar.gz; do
  [[ -s "$STAGE/$F" ]] || { echo "GAGAL: $F kosong/hilang — backup dibatalkan."; exit 1; }
done
OUT="$BACKUP_DIR/radius-backup-$TS.tar.gz"
tar czf "$OUT" -C "$STAGE" .
chmod 600 "$OUT"

# Rotasi: simpan N terakhir
ls -1t "$BACKUP_DIR"/radius-backup-*.tar.gz 2>/dev/null | tail -n +$((RETENTION + 1)) | xargs -r rm -f

echo "Selesai: $OUT ($(du -h "$OUT" | cut -f1))"

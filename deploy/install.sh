#!/usr/bin/env bash
# Installer server D Radius — dari Ubuntu/Debian kosong sampai seluruh stack jalan.
# Idempoten: aman dijalankan ulang.
#
# Pemakaian (sebagai root di server baru):
#   scp deploy/install.sh deploy/backup-terbaru.tar.gz root@SERVER_BARU:
#   GITHUB_TOKEN=ghp_xxx ./install.sh --backup radius-backup-XXXX.tar.gz
#
# Tanpa --backup = instalasi kosong (butuh mengisi .env manual sebelum stack jalan).
set -euo pipefail

RADIUS_USER="${RADIUS_USER:-daniswara}"
BASE_DIR="${BASE_DIR:-/home/$RADIUS_USER/radius-server}"
GITHUB_ORG="${GITHUB_ORG:-candradigitalindo}"
REPOS=(backend-radius frontend-radius landing-page nginx-proxy wa-radius)

BACKUP_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --backup) BACKUP_FILE="$2"; shift 2 ;;
    *) echo "argumen tidak dikenal: $1"; exit 1 ;;
  esac
done

[[ $(id -u) -eq 0 ]] || { echo "Jalankan sebagai root (sudo)."; exit 1; }

echo "==> [1/6] Paket dasar + Docker"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq curl git ca-certificates gnupg rsync >/dev/null
if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
fi
systemctl enable --now docker
id "$RADIUS_USER" >/dev/null 2>&1 || useradd -m -s /bin/bash "$RADIUS_USER"
usermod -aG docker "$RADIUS_USER"

echo "==> [2/6] Clone repo aplikasi"
mkdir -p "$BASE_DIR"
for R in "${REPOS[@]}"; do
  if [[ -d "$BASE_DIR/$R/.git" ]]; then
    echo "    $R sudah ada, skip"
  else
    [[ -n "${GITHUB_TOKEN:-}" ]] || { echo "GITHUB_TOKEN wajib diisi untuk clone repo."; exit 1; }
    git clone "https://${GITHUB_ORG}:${GITHUB_TOKEN}@github.com/${GITHUB_ORG}/${R}.git" "$BASE_DIR/$R"
    # simpan remote tanpa token; kredensial push pakai token di root .env (lihat repo-layout)
    git -C "$BASE_DIR/$R" remote set-url origin "https://github.com/${GITHUB_ORG}/${R}.git"
  fi
done
chown -R "$RADIUS_USER":"$RADIUS_USER" "$BASE_DIR"

DEPLOY="$BASE_DIR/backend-radius/deploy"

echo "==> [3/6] Script host + unit systemd (watchdog, recovery, timer CS bot, backup nightly)"
install -m 755 "$DEPLOY"/host/bin/*.sh /usr/local/bin/
install -m 755 "$DEPLOY"/backup.sh /usr/local/bin/radius-backup.sh
install -m 644 "$DEPLOY"/host/systemd/* /etc/systemd/system/
systemctl daemon-reload

echo "==> [4/6] Jaringan Docker bersama"
docker network inspect web >/dev/null 2>&1 || docker network create web

echo "==> [5/6] Data & stack"
if [[ -n "$BACKUP_FILE" ]]; then
  BASE_DIR="$BASE_DIR" bash "$DEPLOY/restore.sh" "$BACKUP_FILE"
else
  echo "    Tidak ada --backup: instalasi kosong."
  if [[ -f "$BASE_DIR/backend-radius/.env" ]]; then
    docker compose -f "$BASE_DIR/backend-radius/docker-compose.yml" up -d --build
    docker compose -f "$BASE_DIR/landing-page/docker-compose.yml" up -d
    docker compose -f "$BASE_DIR/nginx-proxy/docker-compose.yml" up -d
  else
    echo "    backend-radius/.env belum ada — isi dulu lalu jalankan compose up manual."
  fi
fi

echo "==> [6/6] Aktifkan watchdog & timer (setelah stack hidup, supaya tidak false alarm)"
systemctl enable --now docker-iptables-fix.service docker-radius-recovery.service
systemctl enable --now radius-health-check.timer radius-cs-bot-health.timer \
                        radius-cs-escalation.timer radius-cs-followup.timer radius-backup.timer

echo
echo "================= SELESAI ================="
docker ps --format '{{.Names}}\t{{.Status}}' | sort
cat <<'EOF'

Checklist pasca-install (WAJIB dicek, lihat deploy/README.md):
  1. VPN_PUBLIC_IP di backend-radius/.env = IP publik server INI, lalu `docker compose up -d app`
  2. DNS: app.dradius.net, acs.dradius.net, dradius.net -> IP server ini
  3. Endpoint WireGuard semua router masih menunjuk IP lama? Pasang relay DNAT
     di server lama atau update endpoint router (lihat README bagian Migrasi).
  4. Tes: login app, auth PPPoE 1 pelanggan, inform CWMP, kirim WA dari bot.
  5. Backup nightly aktif (radius-backup.timer, 02:00) -> atur offsite (rclone) menyusul.
EOF

#!/bin/bash
# Radius stack health check + alert WhatsApp.
# Dijalankan tiap 5 menit oleh radius-health-check.timer.
# Kirim alert via gateway WA (sesi superadmin) kalau ada container kritis mati
# atau Redis tak merespons. Ada cooldown biar tidak spam, dan notif pulih.

set -u

ALERT_PHONE="6285121398354"          # nomor tujuan alert (gateway D Radius Net)
COOLDOWN=1800                         # detik: jangan ulang alert masalah sama < 30 menit
STATE="/var/lib/radius-health/state" # simpan status terakhir
LOG="/var/log/radius-health-check.log"
ENV_BACKEND="/home/daniswara/radius-server/backend-radius/.env"
CRITICAL="proxy_nginx radius_app radius_worker radius_redis radius_db radius_wa radius_nginx"

mkdir -p "$(dirname "$STATE")"
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG"; }

# Kirim WA via container gateway (port tidak dipublish ke host).
# API_SECRET diambil dari env container, tidak diekspos ke host.
send_wa() {
    local msg="$1"
    docker exec radius_wa node -e '
        const phone=process.argv[1], msg=process.argv[2];
        fetch("http://localhost:3002/api/messages/send",{
            method:"POST",
            headers:{"Content-Type":"application/json","Authorization":"Bearer "+process.env.API_SECRET},
            body:JSON.stringify({tenantId:"superadmin",phone,message:msg})
        }).then(r=>r.text()).then(t=>{console.log(t)}).catch(e=>{console.log("ERR "+e.message);process.exit(1)});
    ' "$ALERT_PHONE" "$msg" 2>>"$LOG"
}

problems=""

# 1) Container kritis harus running
for c in $CRITICAL; do
    st=$(docker inspect -f '{{.State.Status}}' "$c" 2>/dev/null)
    [ "$st" = "running" ] || problems="${problems}- ${c}: ${st:-tidak ada}\n"
done

# 2) Redis harus PONG (hanya bila container-nya running)
if [ "$(docker inspect -f '{{.State.Status}}' radius_redis 2>/dev/null)" = "running" ]; then
    RP=$(grep -E "^REDIS_PASSWORD" "$ENV_BACKEND" 2>/dev/null | cut -d= -f2-)
    pong=$(docker exec radius_redis redis-cli -a "$RP" PING 2>/dev/null | tr -d '\r')
    [ "$pong" = "PONG" ] || problems="${problems}- redis: tidak merespons PING\n"
fi

now=$(date +%s)
prev_status="ok"; prev_ts=0
[ -f "$STATE" ] && { prev_status=$(sed -n 1p "$STATE"); prev_ts=$(sed -n 2p "$STATE"); }

if [ -n "$problems" ]; then
    log "MASALAH terdeteksi:\n$problems"
    # kirim kalau status berubah (ok->down) atau sudah lewat cooldown
    if [ "$prev_status" = "ok" ] || [ $(( now - prev_ts )) -ge "$COOLDOWN" ]; then
        host=$(hostname)
        msg=$(printf "🚨 *D Radius — ALERT*\n\nServer: %s\nWaktu: %s\n\nLayanan bermasalah:\n%b\nCek: docker ps / journalctl -u docker-radius-recovery" \
            "$host" "$(date '+%Y-%m-%d %H:%M')" "$problems")
        send_wa "$msg" && log "Alert WA terkirim ke $ALERT_PHONE" || log "Gagal kirim alert WA"
        echo -e "down\n$now" > "$STATE"
    else
        log "Masih dalam cooldown, alert tidak diulang"
        echo -e "down\n$prev_ts" > "$STATE"
    fi
else
    # Pulih: kalau sebelumnya down, kirim notif pulih sekali
    if [ "$prev_status" = "down" ]; then
        send_wa "✅ *D Radius — PULIH*\n\nSemua layanan kritis kembali normal pada $(date '+%Y-%m-%d %H:%M')." \
            && log "Notif pulih terkirim"
    fi
    echo -e "ok\n$now" > "$STATE"
    log "OK semua layanan kritis sehat"
fi

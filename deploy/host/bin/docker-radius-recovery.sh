#!/bin/bash
# Docker recovery script — dijalankan setelah boot untuk memastikan semua container radius berjalan bersih

COMPOSE_DIR="/home/daniswara/radius-server/backend-radius"
LANDING_DIR="/home/daniswara/radius-server/landing-page"
PROXY_DIR="/home/daniswara/radius-server/nginx-proxy"
LOG="/var/log/docker-radius-recovery.log"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG"; }

log "=== Docker Radius Recovery Start ==="

# Tunggu Docker daemon siap
until docker info >/dev/null 2>&1; do
    log "Menunggu Docker daemon..."
    sleep 3
done
log "Docker daemon ready"

# Bersihkan stale docker-proxy yang masih pegang port
for PORT in 80 3000 6379 5433 1812 1813; do
    PIDS=$(ss -tlnp 2>/dev/null | grep ":${PORT} " | grep -o 'pid=[0-9]*' | cut -d= -f2)
    PIDS_UDP=$(ss -ulnp 2>/dev/null | grep ":${PORT} " | grep -o 'pid=[0-9]*' | cut -d= -f2)
    for PID in $PIDS $PIDS_UDP; do
        PROC=$(cat /proc/$PID/comm 2>/dev/null)
        if [ "$PROC" = "docker-proxy" ]; then
            log "Kill stale docker-proxy PID=$PID port=$PORT"
            kill -9 $PID 2>/dev/null
        fi
    done
done
sleep 2

# Force-remove semua container yang statusnya Restarting atau Exited
for NAME in proxy_nginx radius_app radius_worker radius_nginx radius_wa radius_redis radius_db landing_nginx landing_wp landing_db; do
    STATUS=$(docker inspect "$NAME" --format '{{.State.Status}}' 2>/dev/null)
    if [ "$STATUS" = "restarting" ] || [ "$STATUS" = "exited" ]; then
        PID=$(docker inspect "$NAME" --format '{{.State.Pid}}' 2>/dev/null)
        if [ -n "$PID" ] && [ "$PID" != "0" ]; then
            log "Kill PID=$PID container=$NAME"
            kill -9 "$PID" 2>/dev/null
            sleep 1
        fi
        log "Remove container=$NAME (status=$STATUS)"
        docker rm -f "$NAME" 2>/dev/null
    fi
done

# Bersihkan container dengan nama aneh (ada container ID di depan, misal cce079944ba5_radius_app)
docker ps -a --format '{{.Names}}' | grep -E '^[a-f0-9]{12}_radius_' | while read CNAME; do
    log "Remove stale container: $CNAME"
    PID=$(docker inspect "$CNAME" --format '{{.State.Pid}}' 2>/dev/null)
    [ -n "$PID" ] && [ "$PID" != "0" ] && kill -9 "$PID" 2>/dev/null && sleep 1
    docker rm -f "$CNAME" 2>/dev/null
done

# Integritas data Redis sebelum start. Sejak 2026-06-27 AOF aktif (primary),
# RDB jadi fallback. redis-check-* TIDAK ada di host → jalankan via image redis
# (container belum naik di titik ini, jadi aman validasi file on-disk).
REDIS_VOL="backend-radius_redisdata"
REDIS_IMG="redis:7-alpine"
RCHK() { docker run --rm -v "${REDIS_VOL}:/data" "$REDIS_IMG" "$@"; }
AOF_MANIFEST="/data/appendonlydir/appendonly.aof.manifest"
if RCHK test -f "$AOF_MANIFEST" 2>/dev/null; then
    log "Cek integritas AOF Redis..."
    if ! RCHK redis-check-aof "$AOF_MANIFEST" >/dev/null 2>&1; then
        # Redis auto-truncate tail rusak (aof-load-truncated yes default),
        # jadi JANGAN auto-fix/hapus (hindari data loss diam-diam) — cukup warning.
        log "WARNING: AOF Redis gagal validasi. Redis akan coba load-truncated; cek manual bila gagal start."
    else
        log "AOF Redis OK"
    fi
elif RCHK test -f /data/dump.rdb 2>/dev/null; then
    log "Cek integritas dump.rdb Redis (tanpa AOF)..."
    if ! RCHK redis-check-rdb /data/dump.rdb >/dev/null 2>&1; then
        log "Redis dump.rdb korup & tak ada AOF — dihapus agar Redis tetap bisa start"
        RCHK rm -f /data/dump.rdb >> "$LOG" 2>&1
    fi
fi

# Jalankan docker compose up — urutan: backend dulu, lalu landing, lalu proxy
log "Running docker compose up -d (backend)"
cd "$COMPOSE_DIR" && docker compose up -d >> "$LOG" 2>&1
log "Running docker compose up -d (landing)"
cd "$LANDING_DIR" && docker compose up -d >> "$LOG" 2>&1

# Verifikasi radius_app terhubung ke network sebelum proxy naik
sleep 10
NETWORKS=$(docker inspect radius_app --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' 2>/dev/null)
if [ -z "$NETWORKS" ]; then
    log "WARNING: radius_app tidak terhubung ke network, force recreate..."
    cd "$COMPOSE_DIR" && docker compose up -d --no-deps --force-recreate app >> "$LOG" 2>&1
    sleep 5
fi

log "Running docker compose up -d (proxy)"
cd "$PROXY_DIR" && docker compose up -d >> "$LOG" 2>&1

log "=== Docker Radius Recovery Done ==="

#!/bin/sh
# Konsentrator L2TP/SSTP (accel-ppp) untuk router non-WireGuard.
# - IP statis per router dibaca dari /vpn/chap-secrets (ditulis backend).
# - RADIUS: tenant menunjuk GW_IP; DNAT ke container app TANPA SNAT sehingga
#   source IP router (10.78.0.x) tetap utuh untuk identifikasi & secret per-router.
# - CoA: paket dari app/worker diforward ke ppp client; MASQUERADE ke arah ppp
#   supaya router membalas ke GW_IP (yang dikenalnya), bukan IP docker internal.
set -e

GW_IP="${LEGACY_VPN_GW:-10.78.0.1}"
SUBNET="${LEGACY_VPN_SUBNET:-10.78.0.0/24}"
L2TP_PORT="${LEGACY_VPN_L2TP_PORT:-1701}"
SSTP_PORT="${LEGACY_VPN_SSTP_PORT:-5443}"
APP_HOST="${LEGACY_VPN_APP_HOST:-app}"

for m in ppp_generic ppp_async ppp_mppe pppox l2tp_core l2tp_netlink l2tp_ppp; do
  modprobe "$m" 2>/dev/null || echo "[vpn] WARN: modprobe $m gagal (cek kernel host)"
done

sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || echo "[vpn] WARN: gagal set ip_forward (butuh privileged)"

# Sertifikat self-signed untuk SSTP (client MikroTik: verify-server-certificate=no).
mkdir -p /vpn/certs
if [ ! -f /vpn/certs/server.crt ]; then
  openssl req -x509 -newkey rsa:2048 -keyout /vpn/certs/server.key \
    -out /vpn/certs/server.crt -days 3650 -nodes -subj "/CN=vpn.dradius" 2>/dev/null
  echo "[vpn] sertifikat SSTP self-signed dibuat"
fi

touch /vpn/chap-secrets
chmod 600 /vpn/chap-secrets

sed -e "s|@GW_IP@|$GW_IP|g" \
    -e "s|@L2TP_PORT@|$L2TP_PORT|g" \
    -e "s|@SSTP_PORT@|$SSTP_PORT|g" \
    /etc/accel-ppp.conf.tpl > /etc/accel-ppp.conf

iptables -t nat -N DRADIUS_POST 2>/dev/null || iptables -t nat -F DRADIUS_POST
iptables -t nat -C POSTROUTING -j DRADIUS_POST 2>/dev/null || iptables -t nat -A POSTROUTING -j DRADIUS_POST
iptables -t nat -A DRADIUS_POST -o 'ppp+' -j MASQUERADE

# DNAT RADIUS ke app; refresh berkala karena IP container app berubah saat recreate.
(
  LAST_APP_IP=""
  while true; do
    APP_IP="$(getent hosts "$APP_HOST" 2>/dev/null | awk '{print $1; exit}')"
    if [ -n "$APP_IP" ] && [ "$APP_IP" != "$LAST_APP_IP" ]; then
      iptables -t nat -N DRADIUS_PRE 2>/dev/null || iptables -t nat -F DRADIUS_PRE
      iptables -t nat -C PREROUTING -j DRADIUS_PRE 2>/dev/null || iptables -t nat -A PREROUTING -j DRADIUS_PRE
      iptables -t nat -A DRADIUS_PRE -d "$GW_IP" -p udp --dport 1812 -j DNAT --to-destination "$APP_IP:1812"
      iptables -t nat -A DRADIUS_PRE -d "$GW_IP" -p udp --dport 1813 -j DNAT --to-destination "$APP_IP:1813"
      LAST_APP_IP="$APP_IP"
      echo "[vpn] RADIUS DNAT $GW_IP:1812/1813 -> $APP_IP"
    fi
    sleep 30
  done
) &

# Reload accel-ppp bila chap-secrets berubah (router baru diaktifkan dari panel).
(
  LAST=""
  while true; do
    CUR="$(stat -c %Y /vpn/chap-secrets 2>/dev/null || echo 0)"
    if [ -n "$LAST" ] && [ "$CUR" != "$LAST" ]; then
      timeout 5 accel-cmd -H 127.0.0.1 -p 2001 reload 2>/dev/null \
        || echo "[vpn] WARN: accel-cmd reload gagal (chap-secrets tetap dibaca per-auth)"
      echo "[vpn] chap-secrets berubah — reload"
    fi
    LAST="$CUR"
    sleep 15
  done
) &

# Rotasi sederhana: pangkas log accel bila >10MB supaya tidak menguras disk.
# tail -F otomatis mengikuti file yang dipangkas.
(
  while true; do
    for f in /var/log/accel-ppp.log /var/log/accel-ppp-emerg.log; do
      SZ="$(stat -c %s "$f" 2>/dev/null || echo 0)"
      if [ "$SZ" -gt 10485760 ]; then
        : > "$f"
        echo "[vpn] log $f dipangkas (>10MB)"
      fi
    done
    sleep 60
  done
) &

touch /var/log/accel-ppp.log /var/log/accel-ppp-emerg.log
tail -F /var/log/accel-ppp.log /var/log/accel-ppp-emerg.log &

exec accel-pppd -c /etc/accel-ppp.conf -p /run/accel-pppd.pid

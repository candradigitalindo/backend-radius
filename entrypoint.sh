#!/bin/sh
# Bring up the WireGuard interface (wg0) before starting the app.
# The Go app manages PEERS via wgctrl, but the interface itself must be created
# at the OS level. Requires NET_ADMIN cap + the wireguard kernel module on host.
# If WG_PRIVATE_KEY is unset, WireGuard is simply skipped (Direct/IP-Publik mode).
set -e

if [ -n "$WG_PRIVATE_KEY" ]; then
  IFACE="${VPN_INTERFACE:-wg0}"
  SRV_IP="${VPN_SERVER_IP:-10.10.0.1}"
  PORT="${VPN_LISTEN_PORT:-51820}"
  SUBNET_CIDR="$(echo "${VPN_SUBNET:-10.10.0.0/24}" | sed 's#.*/##')"

  if ip link add dev "$IFACE" type wireguard 2>/dev/null; then
    echo "[entrypoint] created WireGuard interface $IFACE"
  else
    echo "[entrypoint] $IFACE already exists or cannot be created (continuing)"
  fi

  if printf '%s' "$WG_PRIVATE_KEY" | wg set "$IFACE" private-key /dev/stdin listen-port "$PORT"; then
    echo "[entrypoint] configured $IFACE private-key + listen-port $PORT"
  else
    echo "[entrypoint] WARN: failed to set wg private-key (WireGuard disabled)"
  fi

  ip addr add "$SRV_IP/$SUBNET_CIDR" dev "$IFACE" 2>/dev/null || true
  ip link set "$IFACE" up 2>/dev/null || true
  echo "[entrypoint] $IFACE up at $SRV_IP/$SUBNET_CIDR"
else
  echo "[entrypoint] WG_PRIVATE_KEY not set — running in Direct/IP-Publik mode (no WireGuard)"
fi

# Route to the legacy VPN (L2TP/SSTP) subnet via the accel-ppp container, so
# RADIUS replies and CoA reach tunneled routers. Refreshed in a loop because the
# vpn container's IP changes when it is recreated. Needs NET_ADMIN.
if [ -n "$LEGACY_VPN_HOST" ]; then
  LEGACY_SUBNET="${LEGACY_VPN_SUBNET:-10.78.0.0/24}"
  (
    LAST_GW=""
    while true; do
      GW="$(getent hosts "$LEGACY_VPN_HOST" 2>/dev/null | awk '{print $1; exit}')"
      if [ -n "$GW" ] && [ "$GW" != "$LAST_GW" ]; then
        if ip route replace "$LEGACY_SUBNET" via "$GW" 2>/dev/null; then
          echo "[entrypoint] legacy VPN route $LEGACY_SUBNET via $GW"
          LAST_GW="$GW"
        fi
      fi
      sleep 30
    done
  ) &
fi

exec "$@"

#!/bin/bash
# Perbaikan pasca-start Docker (snap dockerd). Dijalankan oleh
# docker-iptables-fix.service SETELAH snap.docker.dockerd.service aktif.
#
# Dua tugas:
#  1) Re-apply profile AppArmor docker-default yang benar.
#     dockerd me-regenerate docker-default saat start TANPA blok izin
#     snap.docker.dockerd (karena SnapSecurityLabel kosong), sehingga
#     "docker stop/kill" gagal ("permission denied"). Kita muat ulang versi
#     yang sudah diperbaiki dari /etc/apparmor.d/docker-default. Perubahan
#     profile berlaku live ke semua container yang sudah jalan.
#  2) Jaring pengaman iptables: pastikan tiap bridge Docker punya rule FORWARD
#     (auto-discovery, tidak hardcode bridge ID).

set -u
sleep 10

# ── 1. Re-apply AppArmor docker-default yang benar ──────────────────────────
if [ -f /etc/apparmor.d/docker-default ] && command -v apparmor_parser >/dev/null 2>&1; then
    if apparmor_parser -r /etc/apparmor.d/docker-default 2>/dev/null; then
        echo "docker-fix: docker-default AppArmor profile re-applied at $(date)"
    else
        echo "docker-fix: GAGAL apply docker-default at $(date)"
    fi
fi

# ── 2. Jaring pengaman iptables (idempoten, auto-discover bridge) ───────────
ensure_forward() {
    local BR=$1
    iptables -C DOCKER-FORWARD -i "$BR" -j ACCEPT 2>/dev/null || iptables -A DOCKER-FORWARD -i "$BR" -j ACCEPT
    iptables -C DOCKER-FORWARD -o "$BR" -j ACCEPT 2>/dev/null || iptables -A DOCKER-FORWARD -o "$BR" -j ACCEPT
    iptables -C DOCKER-CT -o "$BR" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
        iptables -I DOCKER-CT 1 -o "$BR" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
}

if iptables -L DOCKER-FORWARD -n >/dev/null 2>&1; then
    for BR in $(ip -o link show | awk -F': ' '{print $2}' | cut -d'@' -f1 | grep -E '^(br-|docker0)'); do
        ensure_forward "$BR"
    done
    echo "docker-fix: iptables safety-net applied at $(date)"
else
    echo "docker-fix: DOCKER-FORWARD belum ada, iptables dilewati at $(date)"
fi

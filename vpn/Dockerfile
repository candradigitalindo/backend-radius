# Konsentrator VPN legacy (L2TP + SSTP) untuk router yang tidak bisa WireGuard:
# RouterOS 6 dan/atau router di belakang NAT. accel-ppp memberi IP statis per
# router dari chap-secrets yang ditulis backend (volume /vpn).
FROM debian:bookworm-slim AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git ca-certificates \
    libssl-dev libpcre3-dev \
 && rm -rf /var/lib/apt/lists/*

ARG ACCEL_PPP_VERSION=1.13.0
RUN git clone --depth 1 --branch ${ACCEL_PPP_VERSION} https://github.com/accel-ppp/accel-ppp.git /src

RUN mkdir /src/build && cd /src/build \
 && cmake -DCMAKE_INSTALL_PREFIX=/usr \
          -DCMAKE_BUILD_TYPE=Release \
          -DRADIUS=FALSE -DSHAPER=FALSE -DLUA=FALSE \
          -DBUILD_IPOE_DRIVER=FALSE -DBUILD_VLAN_MON_DRIVER=FALSE .. \
 && make -j"$(nproc)" \
 && make DESTDIR=/install install

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libssl3 libpcre3 iptables iproute2 kmod openssl procps netbase ca-certificates \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build /install/ /

COPY accel-ppp.conf.tpl /etc/accel-ppp.conf.tpl
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# L2TP (udp) + SSTP (tcp, TLS)
EXPOSE 1701/udp 5443/tcp

ENTRYPOINT ["/entrypoint.sh"]

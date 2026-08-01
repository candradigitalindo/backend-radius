[modules]
log_file
l2tp
sstp
auth_pap
auth_chap_md5
auth_mschap_v1
auth_mschap_v2
ippool
chap-secrets

[core]
thread-count=2

[common]
single-session=replace

[ppp]
verbose=1
min-mtu=1280
mtu=1400
mru=1400
ipv4=require
ipv6=deny
lcp-echo-interval=20
lcp-echo-failure=3
mppe=prefer

[l2tp]
verbose=1
bind=0.0.0.0
port=@L2TP_PORT@
host-name=dradius

[sstp]
verbose=1
accept=ssl
ssl-pemfile=/vpn/certs/server.crt
ssl-keyfile=/vpn/certs/server.key
port=@SSTP_PORT@

# ACL IP SUMBER koneksi tunnel (bukan subnet IP tunnel!). Router tenant dial
# dari IP publik/NAT mana pun, jadi harus disable — autentikasi tetap lewat
# chap-secrets.
[client-ip-range]
disable

# TANPA pool cadangan — IP tunnel WAJIB dari kolom ke-4 chap-secrets. Pool akan
# menimpa IP statis (terbukti saat testing) dan memberi IP acak yang merusak
# identifikasi NAS-IP per router.
[ip-pool]
gw-ip-address=@GW_IP@

[chap-secrets]
chap-secrets=/vpn/chap-secrets
gw-ip-address=@GW_IP@

[cli]
tcp=127.0.0.1:2001

[log]
log-file=/var/log/accel-ppp.log
log-emerg=/var/log/accel-ppp-emerg.log
level=3

# Deploy & Disaster Recovery — D Radius

Paket ini membuat server baru bisa berdiri dari OS kosong sampai seluruh stack jalan
dalam ±15 menit: `install.sh` (bootstrap), `backup.sh` (backup harian), `restore.sh`
(restore data), plus salinan script host & unit systemd di `host/`.

## Isi

| File | Fungsi |
|---|---|
| `install.sh` | Bootstrap server baru: Docker, clone 5 repo, unit systemd, restore, up semua |
| `backup.sh` | Dump Postgres/Mongo/MySQL + arsip volume + `.env` & `keys/` → 1 tarball |
| `restore.sh` | Kebalikan `backup.sh`, dengan urutan start yang benar |
| `host/bin/` | Script host: recovery Docker, iptables fix, health-check WA |
| `host/systemd/` | Unit & timer: watchdog, CS bot (followup/eskalasi/health), backup nightly |

## Server baru (migrasi / disaster recovery)

```bash
# Di server LAMA: buat backup terbaru
/home/daniswara/radius-server/backend-radius/deploy/backup.sh

# Kirim installer + backup ke server BARU
scp deploy/install.sh /var/backups/radius/radius-backup-XXXX.tar.gz root@IP_BARU:

# Di server BARU (Ubuntu/Debian kosong, sebagai root)
GITHUB_TOKEN=ghp_xxx ./install.sh --backup radius-backup-XXXX.tar.gz
```

Setelah selesai, kerjakan checklist yang dicetak installer — terutama
`VPN_PUBLIC_IP` di `backend-radius/.env` dan DNS.

## Migrasi tanpa downtime (ringkas)

1. H-1: turunkan TTL DNS ke 60–300 dtk; jalankan `install.sh --backup` di server baru
   (warm standby).
2. Cutover: stop `radius_app`/`radius_worker` di lama → `backup.sh` final → `restore.sh`
   di baru → ganti DNS.
3. Di server lama pasang relay DNAT ke IP baru selama transisi, supaya router yang
   masih menunjuk IP lama tetap terlayani:
   ```bash
   sysctl -w net.ipv4.ip_forward=1
   for P in "51820 udp" "1812 udp" "1813 udp" "80 tcp"; do set -- $P
     iptables -t nat -A PREROUTING -p $2 --dport $1 -j DNAT --to-destination IP_BARU
   done
   iptables -t nat -A POSTROUTING -j MASQUERADE
   ```
4. Pindahkan endpoint WireGuard router ke IP/domain baru (bertahap, via API MikroTik),
   pantau heartbeat, lalu matikan server lama.

Sesi PPPoE yang sudah online **tidak putus** saat RADIUS mati sebentar — hanya login
baru yang gagal — jadi jendela cutover beberapa menit tidak terasa oleh pelanggan.

## Backup

- Nightly 02:00 via `radius-backup.timer` → `/var/backups/radius/`, rotasi 7 hari,
  permission 600 (berisi secret: `.env`, token GitHub, kunci WireGuard).
- Postgres/Mongo/MySQL di-dump konsisten (aman saat live). Volume `redisdata`,
  `n8n_data`, `storage`, `wp_data` dicopy live — untuk backup rutin ini cukup;
  untuk cutover migrasi, stop `radius_app`/`radius_worker` dulu supaya tidak ada
  write yang tertinggal.
- `pgdata`/`mongodata`/`mysql` TIDAK diarsip sebagai volume (sudah lewat dump).
- **PR berikutnya:** kirim tarball ke offsite (rclone → S3/Backblaze/Drive) —
  backup yang menginap di server yang sama bukan disaster recovery.

## Catatan

- Compose project name terikat nama folder: kelima repo **harus** berada di
  `/home/daniswara/radius-server/<nama-repo>` agar nama volume cocok
  (`backend-radius_*`, `landing-page_*`).
- Network `web` bersifat external — dibuat installer (`docker network create web`).
- `host/bin` & `host/systemd` adalah salinan master dari `/usr/local/bin` dan
  `/etc/systemd/system`. Kalau mengubah yang live, salin ke sini juga supaya
  installer tidak menyebarkan versi basi.
- Script CS bot n8n (`cs-*.js`) tinggal di volume `n8n_data` — ikut ter-backup.

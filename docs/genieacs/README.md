# GenieACS — Preset & Provision

Konfigurasi TR-069/CWMP GenieACS **tersimpan di MongoDB**, bukan di file. Saat
pindah server / GenieACS fresh, muat ulang preset + provision di bawah ini via
NBI (REST API) supaya perangkat di-refresh otomatis tiap Inform.

## Komponen

- **Provision `inform_refresh`** — script [`inform_refresh.js`](./inform_refresh.js).
  Menarik parameter penting (SoftwareVersion, UpTime, IP WAN, RX/TX Power PON,
  SSID + sandi WiFi, host LAN, perangkat wireless) tiap sesi.
- **Preset `inform_refresh`** — memicu provision di atas untuk semua Inform
  (precondition kosong = semua perangkat).

> ⚠️ Jangan minta `value` pada path objek (berakhiran titik) — itu bikin channel
> FAULT `Invalid parameter path` dan refresh berhenti total. Pakai wildcard `*`.
> Lihat catatan di atas `inform_refresh.js`.

## Cara muat (jalankan dari host; NBI = service `genieacs-nbi:7557`)

```sh
# 1) Provision (dari file)
docker exec -i radius_genieacs_cwmp node -e '
  const http=require("http"),fs=require("fs");
  const body=fs.readFileSync("/dev/stdin");
  const r=http.request({host:"genieacs-nbi",port:7557,
    path:"/provisions/inform_refresh",method:"PUT",
    headers:{"Content-Type":"application/javascript","Content-Length":body.length}},
    s=>s.on("end",()=>console.log("provision:",s.statusCode)));
  r.end(body);' < inform_refresh.js

# 2) Preset (memicu provision di tiap Inform)
docker exec radius_genieacs_cwmp node -e '
  const http=require("http");
  const body=JSON.stringify({weight:0,precondition:"",
    configurations:[{type:"provision",name:"inform_refresh"}]});
  const r=http.request({host:"genieacs-nbi",port:7557,
    path:"/presets/inform_refresh",method:"PUT",
    headers:{"Content-Type":"application/json","Content-Length":body.length}},
    s=>s.on("end",()=>console.log("preset:",s.statusCode)));
  r.write(body);r.end();'
```

## Sisi perangkat (ONU/router)

- ACS URL: `https://app.dradius.net/cwmp` (inform **tanpa** auth)
- PeriodicInform: ON, interval 300–3600 dtk
- Connection Request username/password: bebas (dipakai ACS utk wake-up)

## Verifikasi

```sh
docker exec radius_acs_mongo mongo genieacs --quiet --eval \
  'print("devices="+db.devices.count()+" faults="+db.faults.count())'
```
`faults` harus 0. Kalau ada device tapi data kosong, sesuaikan path parameter
di `inform_refresh.js` dengan data model merk ONU yang dipakai.

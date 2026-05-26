# WhatsApp & Broadcast API — Panduan Frontend

> Base URL: `https://<domain>/api/v1`  
> Semua endpoint protected memerlukan header `Authorization: Bearer <token>`

---

## Daftar Endpoint

| Endpoint | Method | Keterangan |
|---|---|---|
| `/whatsapp/send` | POST | Kirim WA ke satu nomor |
| `/whatsapp/sessions/status` | GET | Cek status koneksi WA |
| `/whatsapp/sessions/start` | POST | Mulai/sambung sesi WA |
| `/whatsapp/sessions/qr` | GET | Ambil QR code untuk scan |
| `/whatsapp/sessions` | DELETE | Putus sesi WA |
| `/whatsapp/logs` | GET | Log pengiriman WA |
| `/whatsapp/config` | GET/PUT | Konfigurasi WA |
| `/whatsapp/templates` | GET/POST/PUT/DELETE | Template broadcast |
| `/broadcasts` | POST | Kirim broadcast WA massal |
| `/broadcasts` | GET | Riwayat broadcast |
| `/broadcasts/:id` | GET | Detail broadcast |
| `/notifications/send` | POST | Kirim notifikasi ke 1 pelanggan |
| `/notifications/broadcast` | POST | Broadcast notifikasi ke banyak pelanggan |

---

## 1. Kirim WA ke Satu Nomor

**`POST /api/v1/whatsapp/send`**  
Permission: `whatsapp.send`

### Tanpa file — JSON
```json
{
  "phone": "6281234567890",
  "message": "Halo, tagihan Anda sudah jatuh tempo."
}
```

### Dengan file — `multipart/form-data`
| Field | Tipe | Keterangan |
|---|---|---|
| `phone` | string | Nomor WA (format internasional, tanpa `+`) |
| `message` | string | Teks / caption |
| `file` | File | Gambar atau dokumen (opsional) |

> Jika ada file: `message` menjadi **caption** dari file tersebut.  
> Ekstensi gambar: `.jpg .jpeg .png .webp .gif` → dikirim sebagai gambar  
> Ekstensi lain: `.pdf .doc .xlsx` dll → dikirim sebagai dokumen

**Contoh JavaScript (fetch):**
```javascript
// Tanpa file
const res = await fetch('/api/v1/whatsapp/send', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ phone: '6281234567890', message: 'Halo!' }),
})

// Dengan file
const form = new FormData()
form.append('phone', '6281234567890')
form.append('message', 'Ini invoice Anda.')
form.append('file', fileInput.files[0])

const res = await fetch('/api/v1/whatsapp/send', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: form,
})
```

**Response sukses:**
```json
{
  "id": "msg-id",
  "jid": "6281234567890@s.whatsapp.net",
  "status": "sent",
  "timestamp": 1715000000
}
```

---

## 2. Broadcast WA Massal

**`POST /api/v1/broadcasts`**  
Permission: `broadcasts.create`

### Field

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `type` | string | ✅ | Kategori pesan, bebas isi (misal: `"info"`, `"promo"`, `"tagihan"`) |
| `title` | string | ✅ | Judul broadcast (untuk pencatatan) |
| `message` | string | ✅ | Isi pesan / caption |
| `target` | string | | `"active"` `"inactive"` `"isolated"` `"all"` (default: `"all"`) |
| `image_url` | string | | URL gambar/dokumen (JSON only, alternatif upload file) |
| `file` | File | | Upload file langsung (multipart only) |

### Target penerima

| Nilai | Penerima |
|---|---|
| `active` | Pelanggan aktif |
| `inactive` | Pelanggan tidak aktif |
| `isolated` | Pelanggan diisolir |
| `all` | Semua pelanggan |

### Tanpa file — JSON
```json
{
  "type": "info",
  "title": "Pemberitahuan Pemeliharaan",
  "message": "Jaringan akan down 23.00–01.00 WIB.",
  "target": "active"
}
```

### Dengan file — `multipart/form-data`
| Field | Nilai contoh |
|---|---|
| `type` | `info` |
| `title` | `Promo Bulan Ini` |
| `message` | `Diskon 20% untuk upgrade paket.` |
| `target` | `all` |
| `file` | `[binary: promo.jpg]` |

**Contoh JavaScript:**
```javascript
// Tanpa file
const res = await fetch('/api/v1/broadcasts', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    type: 'info',
    title: 'Pemeliharaan',
    message: 'Jaringan down 23.00–01.00.',
    target: 'active',
  }),
})

// Dengan file
const form = new FormData()
form.append('type', 'promo')
form.append('title', 'Promo Bulan Ini')
form.append('message', 'Diskon 20% untuk upgrade paket.')
form.append('target', 'all')
form.append('file', fileInput.files[0])

const res = await fetch('/api/v1/broadcasts', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: form,
})
```

**Response sukses (`201 Created`):**
```json
{
  "id": "uuid",
  "type": "info",
  "title": "Pemeliharaan",
  "message": "Jaringan down 23.00–01.00.",
  "target": "active",
  "status": "sending",
  "total_sent": 150,
  "total_success": 0,
  "total_failed": 0,
  "total_pending": 0,
  "sent_by": "user-uuid",
  "created_at": "2026-05-13T10:00:00Z"
}
```

> Status broadcast: `sending` → `completed` / `failed` / `pending`  
> Cek status dengan `GET /api/v1/broadcasts/:id`

---

## 3. Kirim Notifikasi ke Satu Pelanggan

**`POST /api/v1/notifications/send`**  
Permission: `notifications.send`

### Field

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `customer_id` | string (UUID) | ✅ | ID pelanggan |
| `title` | string | ✅ | Judul notifikasi |
| `body` | string | ✅ | Isi notifikasi / caption |
| `data` | string (JSON) | | Data tambahan (harus JSON valid atau kosong) |
| `file` | File | | Upload file (multipart only) |

### Tanpa file — JSON
```json
{
  "customer_id": "uuid-pelanggan",
  "title": "Tagihan Baru",
  "body": "Tagihan Anda sebesar Rp 200.000 telah diterbitkan.",
  "data": "{\"invoice_id\": \"inv-001\"}"
}
```

### Dengan file — `multipart/form-data`
| Field | Nilai contoh |
|---|---|
| `customer_id` | `uuid-pelanggan` |
| `title` | `Invoice Bulan Ini` |
| `body` | `Silakan cek lampiran invoice berikut.` |
| `data` | *(kosong atau JSON string)* |
| `file` | `[binary: invoice.pdf]` |

**Contoh JavaScript:**
```javascript
// Dengan file
const form = new FormData()
form.append('customer_id', 'uuid-pelanggan')
form.append('title', 'Invoice Bulan Ini')
form.append('body', 'Silakan cek lampiran invoice berikut.')
form.append('file', fileInput.files[0])

const res = await fetch('/api/v1/notifications/send', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: form,
})
```

**Response sukses:**
```json
{ "message": "Notifikasi terkirim" }
```

---

## 4. Broadcast Notifikasi ke Banyak Pelanggan

**`POST /api/v1/notifications/broadcast`**  
Permission: `notifications.send`

### Field

| Field | Tipe | Wajib | Keterangan |
|---|---|---|---|
| `title` | string | ✅ | Judul notifikasi |
| `body` | string | ✅ | Isi notifikasi / caption |
| `target` | string | | `"active"` `"inactive"` `"isolated"` `"all"` (default: `"all"`) |
| `data` | string (JSON) | | Data tambahan push notification |
| `file` | File | | Upload file (multipart only) |

### Tanpa file — JSON
```json
{
  "title": "Promo Spesial",
  "body": "Upgrade paket dan hemat 20%.",
  "target": "active",
  "data": ""
}
```

### Dengan file — `multipart/form-data`
| Field | Nilai contoh |
|---|---|
| `title` | `Promo Spesial` |
| `body` | `Lihat penawaran terbaru kami!` |
| `target` | `active` |
| `data` | *(kosong)* |
| `file` | `[binary: promo.jpg]` |

**Contoh JavaScript:**
```javascript
// Dengan file
const form = new FormData()
form.append('title', 'Promo Spesial')
form.append('body', 'Lihat penawaran terbaru kami!')
form.append('target', 'active')
form.append('file', fileInput.files[0])

const res = await fetch('/api/v1/notifications/broadcast', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: form,
})
```

**Response sukses:**
```json
{ "message": "Broadcast terkirim", "sent": 42 }
```

> `sent` = jumlah push notification FCM yang berhasil dikirim.  
> Pengiriman WA dilakukan secara background (tidak ditunggu).

---

## Status WA Session

**`GET /api/v1/whatsapp/sessions/status`**

```json
{
  "status": "connected",
  "device": {
    "phone": "6281234567890",
    "name": "My Phone",
    "platform": "Android",
    "connectedAt": "2026-05-13T08:00:00Z"
  }
}
```

| `status` | Keterangan |
|---|---|
| `connected` | WA aktif dan siap kirim |
| `disconnected` | WA tidak terhubung |
| `connecting` | Sedang proses koneksi |

> Jika `disconnected`, tampilkan tombol **Start Session** dan **Scan QR**.

---

## Catatan Penting untuk Frontend

1. **Pemilihan Content-Type otomatis:**
   - Tidak ada file → `Content-Type: application/json`
   - Ada file → `Content-Type: multipart/form-data` (jangan set manual, biarkan browser yang set boundary)

2. **File yang didukung:**
   - Gambar (dikirim sebagai caption+gambar): `jpg`, `jpeg`, `png`, `webp`, `gif`
   - Dokumen (dikirim sebagai caption+file): `pdf`, `doc`, `docx`, `xls`, `xlsx`, dll

3. **Ukuran file:** Sesuai limit WA Service (umumnya maks 16MB untuk dokumen, 5MB untuk gambar)

4. **Broadcast bersifat async** — setelah API response `201`, proses kirim WA berjalan di background. Gunakan polling `GET /broadcasts/:id` untuk cek progres.

5. **Nomor telepon** harus format internasional tanpa `+`, contoh: `6281234567890` (Indonesia)

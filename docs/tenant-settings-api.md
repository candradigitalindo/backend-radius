# Tenant Settings — Frontend Developer Guide

> Base URL: `https://backend-radius.binjaidc.com/api/v1`  
> Semua endpoint di bawah ini memerlukan: `Authorization: Bearer <token>`

---

## 1. Informasi Umum Tenant

### GET `/tenant`

Ambil profil tenant yang sedang login. **Respons menyertakan `wa_api_key`, `pg_api_key`, `pg_secret_key`** (khusus endpoint ini — untuk ditampilkan di form settings).

**Response:**
```json
{
  "id": "01KQJ2SEFY5Q3KP2AZ3H5TGMJZ",
  "name": "BinjaiDC",
  "slug": "binjaidc",
  "email": "admin@binjaidc.com",
  "phone": "0812xxxxxxxx",
  "address": "Jl. Contoh No. 1, Binjai",
  "logo_url": "https://...",
  "timezone": "Asia/Jakarta",
  "currency": "IDR",
  "billing_cycle": 1,
  "due_day": 10,
  "isolir_day": 15,
  "grace_period": 3,
  "plan": "pro",
  "plan_expires_at": "2027-01-01T00:00:00Z",
  "max_customers": 500,
  "wa_sender": "628123456789",
  "pg_provider": "tripay",
  "pg_merchant_id": "T12345",
  "is_active": true,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-05-14T07:00:00Z",

  // hanya muncul di endpoint ini (TenantWithSecrets):
  "wa_api_key": "key-wa-xxx",
  "pg_api_key": "key-pg-xxx",
  "pg_secret_key": "secret-pg-xxx"
}
```

> **Catatan frontend:** Field `wa_api_key`, `pg_api_key`, `pg_secret_key` hanya tersedia dari endpoint `GET /tenant`. Endpoint lain (`GET /tenants/:id`, dll.) **tidak** mengembalikan field ini. Gunakan data ini untuk pre-fill form pengaturan integrasi.

---

### PUT `/tenant`

Update profil umum dan konfigurasi billing.

**Body:**
```json
{
  "name": "BinjaiDC",
  "email": "admin@binjaidc.com",
  "phone": "0812xxxxxxxx",
  "address": "Jl. Contoh No. 1, Binjai",
  "logo_url": "https://storage.../logo.png",
  "timezone": "Asia/Jakarta",
  "currency": "IDR",
  "billing_cycle": 1,
  "due_day": 10,
  "isolir_day": 15,
  "grace_period": 3,
  "is_active": true
}
```

**Validasi yang dilakukan server:**
- `name` dan `email` wajib diisi
- `email` harus format valid
- `due_day`, `isolir_day`, `grace_period`, `billing_cycle` harus `>= 0`
- `due_day` maksimal `31`

**Response:** objek `Tenant` yang sudah diperbarui (tanpa field secrets).

---

## 2. Konfigurasi Billing

Field billing ada di `PUT /tenant`, bukan endpoint terpisah.

| Field | Tipe | Keterangan |
|---|---|---|
| `billing_cycle` | `int` | Tanggal generate invoice setiap bulan (1–28) |
| `due_day` | `int` | Hari jatuh tempo pembayaran (1–31) |
| `isolir_day` | `int` | Hari ke-N setelah jatuh tempo untuk isolir otomatis |
| `grace_period` | `int` | Toleransi hari sebelum isolir efektif |

**Contoh alur billing:**
- `billing_cycle = 1` → invoice dibuat tiap tanggal 1
- `due_day = 10` → jatuh tempo tanggal 10
- `isolir_day = 5` → isolir dilakukan 5 hari setelah jatuh tempo (tanggal 15)
- `grace_period = 2` → jika pelanggan bayar dalam 2 hari setelah jatuh tempo, tidak diisolir

**Rekomendasi UI:**
```
[ Tanggal Generate Invoice  ] [1 ▼]   ← billing_cycle
[ Tanggal Jatuh Tempo       ] [10 ▼]  ← due_day
[ Hari Isolir (setelah JT)  ] [5 ▼]   ← isolir_day
[ Grace Period (hari)       ] [2 ▼]   ← grace_period
```

---

## 3. Integrasi Payment Gateway

### PUT `/tenant/settings`

Update kredensial WhatsApp dan payment gateway.

**Body:**
```json
{
  "wa_api_key": "key-wa-xxx",
  "wa_sender": "628123456789",
  "pg_provider": "tripay",
  "pg_api_key": "DEV-xxxxxxxxxxxx",
  "pg_secret_key": "xxxxxxxxxxxxxxxx",
  "pg_merchant_id": "T12345"
}
```

**Field `pg_provider`:**
| Nilai | Keterangan |
|---|---|
| `"tripay"` | Tripay (default) |
| `"midtrans"` | Midtrans Snap |

**Credentials per provider:**

**Tripay:**
| Field | Diisi dengan |
|---|---|
| `pg_api_key` | API Key dari dashboard Tripay |
| `pg_secret_key` | Private Key dari dashboard Tripay |
| `pg_merchant_id` | Merchant Code dari dashboard Tripay |

**Midtrans:**
| Field | Diisi dengan |
|---|---|
| `pg_api_key` | Client Key (untuk frontend Snap.js) |
| `pg_secret_key` | Server Key |
| `pg_merchant_id` | tidak wajib (bisa dikosongkan) |

**Response:**
```json
{ "message": "Pengaturan berhasil diperbarui" }
```

> **Catatan frontend:** Setelah simpan, fetch ulang `GET /tenant` untuk menampilkan nilai terbaru di form. Server tidak mengembalikan data tenant dari endpoint ini.

---

### GET `/tenant/webhook-urls`

Ambil URL webhook siap pakai yang harus didaftarkan di dashboard Tripay / Midtrans.

**Response:**
```json
{
  "success": true,
  "data": {
    "tripay": "https://backend-radius.binjaidc.com/api/v1/webhooks/tripay",
    "tripay_voucher": "https://backend-radius.binjaidc.com/api/v1/webhooks/tripay/voucher",
    "tripay_subscription": "https://backend-radius.binjaidc.com/api/v1/webhooks/subscription/tripay",
    "midtrans": "https://backend-radius.binjaidc.com/api/v1/webhooks/midtrans",
    "midtrans_voucher": "https://backend-radius.binjaidc.com/api/v1/webhooks/midtrans/voucher",
    "midtrans_subscription": "https://backend-radius.binjaidc.com/api/v1/webhooks/subscription/midtrans"
  }
}
```

**Rekomendasi UI — halaman "Cara Setup Webhook":**

Tampilkan per-provider sesuai `pg_provider` yang aktif, dengan tombol copy untuk setiap URL:

```
Provider aktif: Tripay

┌─────────────────────────────────────────────────────────────────┐
│ Invoice Callback URL                                            │
│ https://...com/api/v1/webhooks/tripay             [ Salin ]    │
├─────────────────────────────────────────────────────────────────┤
│ Voucher Callback URL                                            │
│ https://...com/api/v1/webhooks/tripay/voucher     [ Salin ]    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Alur Pembayaran via Gateway (Invoice)

Saat pelanggan membayar tagihan melalui payment gateway:

### POST `/invoices/:id/pay-gateway`

**Body:**
```json
{
  "payment_method": "BRIVA",
  "customer_name": "Budi Santoso",
  "customer_email": "budi@mail.com",
  "customer_phone": "0812xxxxxxxx",
  "return_url": "https://frontend.com/invoices/bayar/selesai"
}
```

**Field `payment_method`** — daftar kode yang didukung:

| Provider | Kode | Keterangan |
|---|---|---|
| Tripay | `BRIVA` | BRI Virtual Account |
| Tripay | `BNIVA` | BNI Virtual Account |
| Tripay | `MANDIRIVA` | Mandiri Virtual Account |
| Tripay | `BCAVA` | BCA Virtual Account |
| Tripay | `PERMATAVA` | Permata Virtual Account |
| Tripay | `QRIS` | QRIS |
| Tripay | `QRISC` | QRIS (Customizable) |
| Midtrans | `credit_card` | Kartu Kredit |
| Midtrans | `bank_transfer` | Transfer Bank (VA otomatis) |
| Midtrans | `gopay` | GoPay |
| Midtrans | `shopeepay` | ShopeePay |
| Midtrans | `qris` | QRIS |

> **Catatan:** Metode yang tersedia tergantung konfigurasi akun Tripay/Midtrans masing-masing tenant. Disarankan fetch daftar metode langsung dari API Tripay/Midtrans menggunakan API Key tenant jika ingin menampilkan daftar dinamis.

**Response (201):**
```json
{
  "data": {
    "payment_id": "01KQJ2SEFY5Q3KP2AZ3H5TGMJZ",
    "gateway_trx_id": "T20240514XXXXXXX",
    "payment_url": "https://tripay.co.id/checkout/T20240514XXXXXXX",
    "expired_at": "2026-05-15T07:55:00Z"
  }
}
```

**Error responses:**

| Status | `error` | Keterangan |
|---|---|---|
| 400 | `"Metode pembayaran wajib diisi"` | `payment_method` kosong |
| 404 | `"Faktur tidak ditemukan"` | Invoice ID tidak valid |
| 409 | `"Faktur sudah dibayar"` | Invoice sudah lunas |
| 500 | pesan dari gateway | API Key salah, dll. |

**Alur frontend:**
1. Tampilkan pilihan metode pembayaran ke user
2. `POST /invoices/:id/pay-gateway` → dapat `payment_url`
3. Redirect user ke `payment_url` (buka tab baru atau iframe)
4. Setelah user kembali ke `return_url`, cek status invoice dengan `GET /invoices/:id`
5. Jika `status === "paid"` → tampilkan konfirmasi sukses

---

## 5. Cek Status Pembayaran

### GET `/invoices/:id/payments`

Riwayat semua percobaan pembayaran untuk sebuah invoice.

**Response:**
```json
[
  {
    "id": "...",
    "invoice_id": "...",
    "amount": 150000,
    "payment_method": "BRIVA",
    "gateway": "tripay",
    "gateway_trx_id": "T20240514XXXXXXX",
    "gateway_status": "PAID",
    "status": "paid",
    "paid_at": "2026-05-14T08:30:00Z",
    "expired_at": "2026-05-15T07:55:00Z",
    "created_at": "2026-05-14T07:55:00Z"
  }
]
```

**Field `status`:** `pending` | `paid` | `failed`  
**Field `gateway_status`:** nilai asli dari gateway (Tripay: `PAID`/`EXPIRED`/`FAILED`, Midtrans: `settlement`/`expire`/`cancel`)

---

## 6. Ringkasan Halaman Settings yang Perlu Dibangun

### Halaman: Profil Tenant
- Form: `name`, `email`, `phone`, `address`, `logo_url`, `timezone`, `currency`
- Fetch: `GET /tenant`
- Simpan: `PUT /tenant`

### Halaman: Konfigurasi Billing
- Form: `billing_cycle`, `due_day`, `isolir_day`, `grace_period`
- Fetch: `GET /tenant`
- Simpan: `PUT /tenant` (field billing digabung dengan profil dalam satu request)

### Halaman: Integrasi Payment Gateway
- Dropdown: `pg_provider` (`tripay` / `midtrans`)
- Form fields muncul kondisional berdasarkan provider:
  - Tripay: `pg_api_key` (API Key), `pg_secret_key` (Private Key), `pg_merchant_id` (Merchant Code)
  - Midtrans: `pg_api_key` (Client Key), `pg_secret_key` (Server Key)
- Fetch: `GET /tenant` (ambil nilai existing termasuk secrets)
- Simpan: `PUT /tenant/settings`
- Section "URL Webhook": `GET /tenant/webhook-urls` → tampilkan URL per provider + tombol copy

### Halaman: Integrasi WhatsApp
- Form: `wa_api_key`, `wa_sender`
- Fetch: `GET /tenant`
- Simpan: `PUT /tenant/settings`

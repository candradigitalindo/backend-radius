# Changelog v1.1 — Ringkasan Perubahan untuk Tim Frontend

> Tanggal: 26 Mei 2026  
> Versi backend: v1.1  
> Status: **Production-ready** (semua perubahan sudah di-compile tanpa error)

---

## 🆕 Fitur Baru

### 1. Integrasi Payment Gateway: **Xendit**

Backend sekarang mendukung **3 payment gateway**: Tripay, Midtrans, dan **Xendit** (baru).

#### Perubahan di halaman **Pengaturan Tenant** (`PUT /tenant/settings`)

Field baru yang perlu ditambahkan ke form settings:

| Field | Tipe | Keterangan |
|---|---|---|
| `pg_provider` | `string` | Sekarang mendukung nilai `"xendit"` (selain `"tripay"` dan `"midtrans"`) |
| `pg_sandbox` | `boolean` | **BARU** — `true` = mode testing, `false` = production. Default: `true` |

**Panduan input per provider:**

| Provider | `pg_api_key` | `pg_secret_key` | `pg_merchant_id` |
|---|---|---|---|
| `tripay` | API Key | Private Key | Merchant Code |
| `midtrans` | Client Key | Server Key | *(kosongkan)* |
| `xendit` | Secret Key (`xnd_...`) | Webhook Verification Token | *(kosongkan)* |

**Contoh body request:**
```json
{
  "pg_provider": "xendit",
  "pg_api_key": "xnd_development_XXXXXXXX",
  "pg_secret_key": "XENDIT_WEBHOOK_TOKEN_HERE",
  "pg_sandbox": true
}
```

#### Perubahan di halaman **Webhook URLs** (`GET /tenant/webhook-urls`)

Response sekarang menyertakan 3 URL baru untuk Xendit:

```json
{
  "data": {
    "tripay": "...",
    "tripay_voucher": "...",
    "tripay_subscription": "...",
    "midtrans": "...",
    "midtrans_voucher": "...",
    "midtrans_subscription": "...",
    "xendit": "https://api.example.com/api/v1/webhooks/xendit",
    "xendit_voucher": "https://api.example.com/api/v1/webhooks/xendit/voucher",
    "xendit_subscription": "https://api.example.com/api/v1/webhooks/subscription/xendit"
  }
}
```

**Tampilkan URL Xendit** di halaman webhook URLs dengan instruksi:
> "Daftarkan URL ini di Xendit Dashboard → Settings → Callbacks → Invoice Paid"

---

### 2. Field `pg_sandbox` di Response Tenant

Response `GET /tenant` sekarang menyertakan field baru:

```json
{
  "pg_provider": "xendit",
  "pg_sandbox": true,
  ...
}
```

Gunakan field ini untuk menampilkan badge **"Mode Testing"** / **"Mode Produksi"** di halaman settings.

---

### 3. Endpoint Webhook Baru (tidak perlu action frontend)

3 endpoint webhook baru ditambahkan untuk Xendit. Ini adalah endpoint server-to-server yang dipanggil langsung oleh Xendit — **tidak ada perubahan yang diperlukan di frontend** untuk ini.

| Endpoint | Keterangan |
|---|---|
| `POST /api/v1/webhooks/xendit` | Callback pembayaran invoice via Xendit |
| `POST /api/v1/webhooks/xendit/voucher` | Callback pembelian voucher via Xendit |
| `POST /api/v1/webhooks/subscription/xendit` | Callback langganan SaaS via Xendit |

---

## 🔧 Perubahan yang Memerlukan Update Frontend

### Halaman: Pengaturan Integrasi (`/settings/integration` atau sejenisnya)

1. **Tambahkan opsi `xendit`** di dropdown `pg_provider`
2. **Tambahkan toggle/checkbox `pg_sandbox`** (label: "Mode Sandbox/Testing")
3. **Update label field** berdasarkan provider yang dipilih:
   - Saat `pg_provider = "xendit"`:
     - Label `pg_api_key` → "Secret Key (xnd_...)"
     - Label `pg_secret_key` → "Webhook Verification Token"
     - Sembunyikan field `pg_merchant_id`
4. **Tambahkan URL Xendit** di bagian webhook URLs

### Halaman: Webhook URLs

Tampilkan 3 URL baru untuk Xendit dengan instruksi setup yang jelas.

---

### 4. Endpoint Uji Koneksi Payment Gateway (BARU)

Endpoint baru untuk memvalidasi kredensial payment gateway **sebelum disimpan ke production**:

```
POST /tenant/settings/test
Authorization: Bearer <token>
Body: (tidak diperlukan)
```

**Response sukses (200):**
```json
{ "success": true, "message": "Koneksi payment gateway berhasil" }
```

**Response gagal (400):**
```json
{ "success": false, "error": "Tripay: Invalid API Key" }
```

**Cara kerja per provider:**
| Provider | Endpoint yang diuji |
|---|---|
| `tripay` | `GET /merchant/profile` (Bearer API Key) |
| `midtrans` | `GET /v2/payment-types` (Basic Auth Server Key) |
| `xendit` | `GET /balance` (Basic Auth Secret Key) |

> **Rekomendasi UX:** Tambahkan tombol **"Test Koneksi"** di halaman pengaturan payment gateway. Panggil endpoint ini setelah user mengisi kredensial, sebelum menekan tombol Simpan.

---

## 📋 Checklist untuk Tim Frontend

- [ ] Update dropdown `pg_provider` → tambahkan opsi `"xendit"`
- [ ] Tambahkan field `pg_sandbox` (boolean toggle) di form settings
- [ ] Update label dinamis untuk `pg_api_key` dan `pg_secret_key` berdasarkan provider
- [ ] Sembunyikan `pg_merchant_id` saat provider = `xendit` atau `midtrans`
- [ ] Tampilkan URL webhook Xendit di halaman webhook URLs
- [ ] Tambahkan instruksi setup Xendit (link ke dashboard Xendit)
- [ ] Tampilkan badge "Mode Testing/Produksi" berdasarkan `pg_sandbox`
- [ ] **[BARU]** Tambahkan tombol "Test Koneksi" yang memanggil `POST /tenant/settings/test`

---

## ℹ️ Tidak Ada Breaking Change di Endpoint Lain

Semua endpoint yang sudah ada **tidak berubah**. Perubahan ini bersifat **additive** (menambah, tidak menghapus atau mengubah yang sudah ada).

---

## 📚 Referensi

- Dokumentasi lengkap: `docs/frontend-api.md`
- Swagger/OpenAPI: `docs/swagger.yaml`
- Xendit API Docs: https://developers.xendit.co/api-reference/

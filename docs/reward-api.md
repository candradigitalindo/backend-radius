# Reward Program API

Dokumentasi lengkap untuk sistem reward, referral, dan klaim reward.  
Semua endpoint bersifat **per-tenant** — data antar tenant terisolasi sepenuhnya.

---

## Autentikasi & Header

Semua endpoint memerlukan JWT Bearer token dan header `X-Tenant-ID`.

```
Authorization: Bearer <token>
X-Tenant-ID: <tenant_id>
Content-Type: application/json
```

**Base URL:**

```
https://backend-radius.binjaidc.com/api/v1
```

Contoh URL lengkap:
```
GET  https://backend-radius.binjaidc.com/api/v1/rewards
GET  https://backend-radius.binjaidc.com/api/v1/rewards/dashboard?months=12
GET  https://backend-radius.binjaidc.com/api/v1/referrals
POST https://backend-radius.binjaidc.com/api/v1/reward-claims/01JCLAIM001/apply
```

> Semua endpoint reward berada di prefix `/api/v1` — tidak ada prefix tambahan seperti `/admin` atau `/v2`.  
> Port default aplikasi: `3000` (dikonfigurasi via env `APP_PORT`).

---

## Konsep Bisnis

```
Customer A (referrer)
  └─ Bagikan referral_code ke Customer B
       └─ Customer B daftar dengan referral_code → Referral dibuat (status: pending)
            └─ Customer B bayar invoice pertama → Referral qualified
                 └─ Sistem otomatis buat RewardClaim untuk Customer A (balance_credit)
                      └─ Admin/sistem apply claim → saldo kredit ditambah ke invoice berikutnya
```

### Tipe Reward

| Tipe | Keterangan |
|---|---|
| `referral` | Diberikan ke referrer ketika referred customer membayar invoice pertama |
| `loyalty` | Diberikan ke customer yang telah membayar minimal N invoice |
| `promo` | Reward manual/promo khusus |

### Value Type

| Value Type | Keterangan |
|---|---|
| `fixed` | Nominal tetap dalam rupiah (satuan terkecil) |
| `percentage` | Persentase dari tagihan |

### Status Referral

| Status | Keterangan |
|---|---|
| `pending` | Referral baru terdaftar, belum ada pembayaran |
| `qualified` | Referred customer sudah bayar invoice → reward claim dibuat |
| `rewarded` | Claim sudah di-apply secara manual oleh admin |
| `expired` | Referral kadaluarsa |

### Status Klaim (RewardClaim)

| Status | Keterangan |
|---|---|
| `pending` | Klaim tersedia, belum diterapkan |
| `applied` | Klaim sudah diterapkan ke invoice |
| `expired` | Klaim kadaluarsa sebelum digunakan |

### Tipe Klaim

| Tipe | Keterangan |
|---|---|
| `balance_credit` | Kredit saldo untuk invoice berikutnya |
| `invoice_discount` | Diskon langsung pada invoice |

---

## Alur Otomatis (Tidak Perlu Tindakan Manual)

1. **Registrasi customer** — Jika field `referral_code_used` diisi pada saat buat customer, sistem otomatis membuat record `Referral` (status `pending`).
2. **Pembayaran invoice** — Setelah invoice lunas (via manual, Tripay, atau Midtrans), sistem:
   - Mencari referral `pending` milik customer tersebut → ubah ke `qualified` → buat `RewardClaim` untuk referrer
   - Mengecek loyalty reward → jika jumlah invoice berbayar mencapai `min_invoices`, buat `RewardClaim`
3. **Expire otomatis** — Worker berjalan setiap hari pukul 02.00 untuk mengexpire klaim yang sudah melewati `expires_at`.

**Satu-satunya aksi manual:** `POST /reward-claims/:claimId/apply` untuk menerapkan klaim ke invoice.

---

## Endpoints

---

### 1. Reward (Program Reward)

#### `GET /rewards`
Daftar semua program reward tenant.

**Query params:**

| Param | Tipe | Default | Keterangan |
|---|---|---|---|
| `active_only` | bool | `false` | Hanya tampilkan yang aktif |
| `page` | int | `1` | Halaman |
| `per_page` | int | `20` | Jumlah per halaman |

**Permission:** `rewards.view`

**Response:**
```json
{
  "data": [
    {
      "id": "01JTEST00001",
      "tenant_id": "tenant-abc",
      "name": "Reward Referral Standar",
      "description": "Bonus Rp 50.000 untuk setiap referral berhasil",
      "type": "referral",
      "value": 50000,
      "value_type": "fixed",
      "min_invoices": 1,
      "is_active": true,
      "start_date": "2026-01-01T00:00:00Z",
      "end_date": null,
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-01T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "per_page": 20
}
```

---

#### `POST /rewards`
Buat program reward baru.

**Permission:** `rewards.create`

**Request body:**
```json
{
  "name": "Reward Referral Standar",
  "description": "Bonus Rp 50.000 untuk setiap referral berhasil",
  "type": "referral",
  "value": 50000,
  "value_type": "fixed",
  "min_invoices": 1,
  "is_active": true,
  "start_date": "2026-01-01T00:00:00Z",
  "end_date": null
}
```

**Wajib:** `name`, `type`, `value_type`

**Response:** `201 Created`
```json
{ "data": { ...reward object... } }
```

---

#### `GET /rewards/stats`
Ringkasan statistik reward tenant (untuk widget/card di dashboard).

**Permission:** `rewards.view`

**Response:**
```json
{
  "data": {
    "total_referrals": 120,
    "qualified_referrals": 85,
    "total_rewarded": 4250000,
    "pending_claims": 12,
    "active_rewards": 3
  }
}
```

---

#### `GET /rewards/dashboard`
Dashboard lengkap program reward — tren bulanan, top referrer, referral & klaim terbaru.

**Permission:** `rewards.view`

**Query params:**

| Param | Tipe | Default | Keterangan |
|---|---|---|---|
| `months` | int | `12` | Rentang bulan untuk tren (1–24) |

**Response:**
```json
{
  "months": 12,
  "data": {
    "stats": {
      "total_referrals": 120,
      "qualified_referrals": 85,
      "total_rewarded": 4250000,
      "pending_claims": 12,
      "active_rewards": 3
    },
    "monthly_trends": [
      {
        "year": 2026,
        "month": 5,
        "referrals": 15,
        "qualified": 10,
        "claims": 12,
        "claimed_amount": 600000
      }
    ],
    "top_referrers": [
      {
        "customer_id": "01JCUST001",
        "customer_name": "Budi Santoso",
        "referral_count": 10,
        "qualified_count": 8,
        "total_rewarded": 400000
      }
    ],
    "recent_referrals": [
      {
        "id": "01JREF001",
        "referrer_id": "01JCUST001",
        "referrer_name": "Budi Santoso",
        "referred_id": "01JCUST002",
        "referred_name": "Andi Pratama",
        "referral_code": "A1B2C3D4",
        "status": "qualified",
        "reward_amount": 50000,
        "qualified_at": "2026-05-10T08:00:00Z",
        "created_at": "2026-05-01T10:00:00Z"
      }
    ],
    "recent_claims": [
      {
        "id": "01JCLAIM001",
        "customer_id": "01JCUST001",
        "customer_name": "Budi Santoso",
        "reward_id": "01JREWARD001",
        "reward_name": "Reward Referral Standar",
        "amount": 50000,
        "type": "balance_credit",
        "status": "pending",
        "expires_at": "2026-06-01T00:00:00Z",
        "created_at": "2026-05-10T08:00:00Z"
      }
    ],
    "claim_breakdown": [
      { "type": "balance_credit",   "status": "applied",  "count": 70, "total_amount": 3500000 },
      { "type": "balance_credit",   "status": "pending",  "count": 12, "total_amount": 600000  },
      { "type": "invoice_discount", "status": "applied",  "count": 15, "total_amount": 750000  },
      { "type": "balance_credit",   "status": "expired",  "count": 5,  "total_amount": 250000  }
    ]
  }
}
```

---

#### `GET /rewards/:id`
Detail satu program reward.

**Permission:** `rewards.view`

**Response:** `{ "data": { ...reward object... } }`

---

#### `PUT /rewards/:id`
Update program reward.

**Permission:** `rewards.edit`

**Request body:** sama dengan `POST /rewards`

**Response:** `{ "message": "Reward diperbarui" }`

---

#### `DELETE /rewards/:id`
Hapus program reward.

**Permission:** `rewards.delete`

**Response:** `{ "message": "Reward dihapus" }`

---

### 2. Referral

#### `GET /referrals`
Daftar referral.

**Permission:** `rewards.view`

**Query params:**

| Param | Tipe | Keterangan |
|---|---|---|
| `customer_id` | string | Filter referral milik customer (sebagai referrer atau referred) |
| `status` | string | Filter status: `pending`, `qualified`, `rewarded`, `expired` |
| `page` | int | Halaman |
| `per_page` | int | Jumlah per halaman |

**Response:**
```json
{
  "data": [ { ...referral object... } ],
  "total": 50,
  "page": 1,
  "per_page": 20
}
```

**Referral object:**
```json
{
  "id": "01JREF001",
  "tenant_id": "tenant-abc",
  "referrer_id": "01JCUST001",
  "referrer_name": "Budi Santoso",
  "referred_id": "01JCUST002",
  "referred_name": "Andi Pratama",
  "reward_id": "01JREWARD001",
  "referral_code": "A1B2C3D4",
  "status": "qualified",
  "reward_amount": 50000,
  "qualified_at": "2026-05-10T08:00:00Z",
  "rewarded_at": null,
  "created_at": "2026-05-01T10:00:00Z"
}
```

---

#### `GET /referrals/:id`
Detail satu referral.

**Permission:** `rewards.view`

---

#### `POST /referrals/:id/reward`
Tandai referral sebagai sudah diberi reward (manual oleh admin).

**Permission:** `rewards.edit`

**Response:** `{ "message": "Referral ditandai diberi reward" }`

---

### 3. Reward Claims

#### `GET /reward-claims`
Daftar klaim reward.

**Permission:** `rewards.view`

**Query params:**

| Param | Tipe | Keterangan |
|---|---|---|
| `customer_id` | string | Filter klaim milik customer tertentu |
| `status` | string | Filter status: `pending`, `applied`, `expired` |
| `page` | int | Halaman |
| `per_page` | int | Jumlah per halaman |

**Response:**
```json
{
  "data": [ { ...claim object... } ],
  "total": 30,
  "page": 1,
  "per_page": 20
}
```

**Claim object:**
```json
{
  "id": "01JCLAIM001",
  "tenant_id": "tenant-abc",
  "customer_id": "01JCUST001",
  "customer_name": "Budi Santoso",
  "reward_id": "01JREWARD001",
  "reward_name": "Reward Referral Standar",
  "referral_id": "01JREF001",
  "amount": 50000,
  "type": "balance_credit",
  "status": "pending",
  "applied_at": null,
  "expires_at": "2026-06-01T00:00:00Z",
  "created_at": "2026-05-10T08:00:00Z"
}
```

---

#### `POST /reward-claims/:claimId/apply`
Terapkan klaim ke customer (satu-satunya aksi manual dalam alur reward).

**Permission:** `rewards.edit`

Hanya bisa apply klaim dengan status `pending`. Klaim yang sudah `applied` atau `expired` akan ditolak.

**Response:**
```json
{ "message": "Klaim diterapkan" }
```

**Error (400):**
```json
{ "error": "claim not found or already applied" }
```

---

#### `GET /reward-claims/balance/:id`
Cek total saldo klaim `pending` milik customer (jumlah yang bisa digunakan).

**Permission:** `rewards.view`

**Params:** `:id` = customer ID

**Response:**
```json
{ "balance": 150000 }
```

---

## Integrasi dengan Modul Lain

### Buat Customer dengan Referral Code

Saat membuat customer baru, tambahkan field `referral_code_used` untuk mendaftarkan referral secara otomatis:

```json
POST /customers
{
  "name": "Andi Pratama",
  "phone": "081234567890",
  ...
  "referral_code_used": "A1B2C3D4"
}
```

Jika `referral_code_used` diisi dan kode valid, sistem akan:
1. Mencari customer pemilik kode tersebut (referrer)
2. Mencari program reward `referral` yang aktif
3. Membuat record `Referral` (status `pending`) secara otomatis di background

### Referral Code Customer

Setiap customer yang dibuat memiliki `referral_code` unik (8 karakter uppercase) yang bisa dilihat di response `GET /customers/:id`.

---

## Permission yang Diperlukan

| Permission | Endpoint |
|---|---|
| `rewards.view` | Semua GET |
| `rewards.create` | `POST /rewards` |
| `rewards.edit` | `PUT /rewards/:id`, `POST /referrals/:id/reward`, `POST /reward-claims/:claimId/apply` |
| `rewards.delete` | `DELETE /rewards/:id` |

---

## Error Responses

| HTTP Code | Kondisi |
|---|---|
| `400` | Request body tidak valid / klaim sudah applied |
| `404` | Data tidak ditemukan |
| `500` | Error server internal |

Format error:
```json
{ "error": "Pesan error dalam bahasa Indonesia" }
```

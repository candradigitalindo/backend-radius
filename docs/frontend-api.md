# API Documentation — BinjaiDC Radius Backend

> Base URL: `https://backend-radius.binjaidc.com/api/v1`  
> All protected endpoints require: `Authorization: Bearer <token>`  
> All IDs use ULID format (26-char string), e.g. `01KQJ2SEFY5Q3KP2AZ3H5TGMJZ`

---

## Authentication

### Standard Response Envelopes

**Success (list):**
```json
{ "data": [...], "total": 100, "page": 1, "per_page": 20 }
```

**Success (single):**
```json
{ "data": { ... } }
```

**Error:**
```json
{ "error": "pesan error" }
```

---

### `POST /auth/register`
Register new tenant (creates tenant + owner user).

**Body:**
```json
{ "name": "Nama ISP", "email": "admin@example.com", "password": "secret", "phone": "0812..." }
```

**Response:** Same as login response.

### `POST /auth/login`
Login user admin/staff.

**Body:**
```json
{ "email": "admin@example.com", "password": "secret", "tenant_id": "..." }
```

> `tenant_id` opsional. Wajib jika email terdaftar di lebih dari satu tenant (response akan return `409 MULTIPLE_TENANTS`).

**Response:**
```json
{
  "token": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ..."
  },
  "user": {
    "id": "...", "tenant_id": "...", "name": "Admin", "email": "admin@example.com",
    "role": "admin", "phone": "...", "plan": "pro", "plan_expires_at": "...",
    "permissions": ["customers.view", "invoices.pay", ...]
  }
}
```

### `GET /auth/me`
Get current authenticated user profile.

**Response:**
```json
{
  "id": "...", "tenant_id": "...", "name": "Admin", "email": "...",
  "role": "admin", "phone": "...", "plan": "pro", "plan_expires_at": "...",
  "permissions": ["customers.view", ...]
}
```

### `PUT /auth/me`
Update current user profile (name, phone, email).

**Body:**
```json
{ "name": "...", "phone": "...", "email": "..." }
```

### `PUT /auth/change-password`
Change own password.

**Body:**
```json
{ "current_password": "...", "new_password": "..." }
```

### `POST /auth/logout`
Invalidates the current token (requires Authorization header).

### `POST /auth/refresh`
Refresh access token.

**Body:**
```json
{ "refresh_token": "eyJ..." }
```

**Response:**
```json
{ "access_token": "eyJ...", "refresh_token": "eyJ..." }
```

### `POST /auth/reset-pin`
Request password reset PIN via WhatsApp.

**Body:**
```json
{ "email": "admin@example.com", "phone": "0812..." }
```

**Response:**
```json
{ "message": "PIN telah dikirim melalui WhatsApp", "phone": "0812****1234" }
```

### `POST /auth/reset-password`
Verify PIN and reset password.

**Body:**
```json
{ "email": "...", "phone": "...", "pin": "123456", "new_password": "newsecret" }
```

---

## Tenant

### `GET /tenant`
Get current tenant info.

**Response:**
```json
{
  "id": "...", "name": "BinjaiDC", "slug": "binjaidc",
  "email": "...", "phone": "...", "address": "...",
  "logo_url": "...", "timezone": "Asia/Jakarta", "currency": "IDR",
  "billing_cycle": 1, "due_day": 10, "isolir_day": 15, "grace_period": 3,
  "plan": "pro", "plan_expires_at": "...", "max_customers": 500,
  "wa_sender": "628xxx", "pg_provider": "tripay", "pg_api_key": "...", "pg_secret_key": "...",
  "pg_merchant_id": "...", "pg_sandbox": true,
  "is_active": true, "created_at": "...", "updated_at": "..."
}
```

### `PUT /tenant`
Update tenant profile.

**Body:**
```json
{
  "name": "...", "email": "...", "phone": "...", "address": "...",
  "timezone": "Asia/Jakarta", "currency": "IDR",
  "billing_cycle": 1, "due_day": 10, "isolir_day": 15, "grace_period": 3
}
```

### `PUT /tenant/settings`
Update payment gateway & WhatsApp credentials.

> ⚠️ **BREAKING CHANGE (v1.1):** Field `pg_sandbox` ditambahkan. Field `pg_provider` sekarang mendukung nilai `"xendit"`.

**Body:**
```json
{
  "pg_provider": "tripay|midtrans|xendit",
  "pg_api_key": "...",
  "pg_secret_key": "...",
  "pg_merchant_id": "...",
  "pg_sandbox": true,
  "wa_api_key": "...",
  "wa_sender": "628xxx"
}
```

**Panduan per provider:**

| Provider | `pg_api_key` | `pg_secret_key` | `pg_merchant_id` |
|---|---|---|---|
| `tripay` | API Key | Private Key | Merchant Code |
| `midtrans` | Client Key | Server Key | *(tidak dipakai)* |
| `xendit` | Secret Key (`xnd_...`) | Webhook Verification Token | *(tidak dipakai)* |

**`pg_sandbox`:** `true` = mode testing, `false` = production. Default: `true`.

### `POST /tenant/settings/test`
Uji koneksi ke payment gateway menggunakan kredensial yang sudah dikonfigurasi.
Berguna untuk memvalidasi API Key sebelum digunakan di production.

**Body:** *(tidak diperlukan)*

**Response sukses (200):**
```json
{ "success": true, "message": "Koneksi payment gateway berhasil" }
```

**Response gagal (400):**
```json
{ "success": false, "error": "Tripay: Invalid API Key" }
```

**Cara kerja per provider:**
| Provider | Endpoint yang diuji | Autentikasi |
|---|---|---|
| `tripay` | `GET /merchant/profile` | Bearer API Key |
| `midtrans` | `GET /v2/payment-types` | Basic Auth (Server Key) |
| `xendit` | `GET /balance` | Basic Auth (Secret Key) |

> 💡 **Tips UX:** Tampilkan tombol "Test Koneksi" di halaman pengaturan payment gateway. Panggil endpoint ini setelah user mengisi kredensial, sebelum menyimpan ke production.

### `GET /tenant/webhook-urls`
Get ready-to-use webhook URLs to register in payment gateway dashboards.

> ⚠️ **UPDATE (v1.1):** Response sekarang juga menyertakan URL untuk Xendit.

**Response:**
```json
{
  "success": true,
  "data": {
    "tripay": "https://yourdomain.com/api/v1/webhooks/tripay",
    "tripay_voucher": "...",
    "tripay_subscription": "...",
    "midtrans": "...",
    "midtrans_voucher": "...",
    "midtrans_subscription": "...",
    "xendit": "https://yourdomain.com/api/v1/webhooks/xendit",
    "xendit_voucher": "...",
    "xendit_subscription": "..."
  }
}
```

**Cara setup Xendit:**
1. Login ke [Xendit Dashboard](https://dashboard.xendit.co)
2. Buka **Settings → Callbacks**
3. Masukkan URL dari `xendit` di atas ke field **Invoice Paid**
4. Copy **Webhook Verification Token** dari halaman yang sama
5. Masukkan token tersebut ke field `pg_secret_key` di `PUT /tenant/settings`

---

## Users

### `GET /users?page=1&per_page=20&search=`
Permission: `users.view`

### `POST /users`
Permission: `users.create`

**Body:**
```json
{
  "name": "...", "email": "...", "password": "...",
  "role": "admin|staff", "permissions": ["customers.view", ...]
}
```

### `GET /users/:id`
### `PUT /users/:id`
### `DELETE /users/:id`
### `PUT /users/:id/password` — Change specific user password (superadmin)
### `PUT /auth/change-password` — Change own password

---

## Roles

### `GET /roles`
Permission: `roles.view`

### `POST /roles`
Permission: `roles.create`

**Body:**
```json
{
  "name": "Staff Lapangan",
  "permissions": ["customers.view", "tickets.view", "tickets.edit"]
}
```

### `GET /roles/:id`
### `PUT /roles/:id`
### `DELETE /roles/:id`
### `GET /roles/permissions` — Returns all available permission keys grouped by feature

---

## Customers

All customer IDs are ULIDs. Requires `customers.view`.

### `GET /customers?page=1&per_page=20&search=&status=&router_id=&package_id=`

**Status values:** `active` | `inactive` | `isolated` | `deleted`  
**Connection types:** `pppoe` | `static` | `hotspot`

### `GET /customers/next-code`
Returns next auto-generated customer code.

### `POST /customers`
Permission: `customers.create`

**Body:**
```json
{
  "name": "Budi Santoso", "nik": "1234...", "phone": "0812...",
  "email": "budi@mail.com", "address": "Jl. ...",
  "latitude": 3.571, "longitude": 98.718,
  "connection_type": "pppoe",
  "pppoe_username": "budi", "pppoe_password": "secret",
  "ip_address": "10.0.0.5",
  "package_id": "...", "router_id": "...", "odp_port_id": "...",
  "join_date": "2026-01-01",
  "billing_date": 1, "billing_type": "monthly", "billing_deadline": 10,
  "custom_price": null, "discount": 0, "additional_fee": 0,
  "fee_description": "", "notes": ""
}
```

### `GET /customers/:id`
Returns full customer detail response:

```json
{
  "id": "...", "tenant_id": "...", "customer_code": "C0001",
  "name": "...", "nik": "...", "phone": "...", "email": "...",
  "address": "...", "latitude": 3.571, "longitude": 98.718,
  "package_id": "...", "router_id": "...", "odp_port_id": "...",
  "status": "active", "notes": "...", "referral_code": "A1B2C3D4",
  "created_at": "...", "updated_at": "...",
  "access": {
    "pppoe_username": "budi", "pppoe_password": "secret", "acs_url": "..."
  },
  "billing": {
    "join_date": "...", "billing_date": 1, "invoice_date": "...",
    "billing_type": "monthly", "billing_deadline": 10,
    "billing_due_date": "...", "custom_price": null,
    "discount": 0, "additional_fee": 0, "fee_description": "",
    "current_invoice": { "id": "...", "invoice_number": "INV-...", "status": "unpaid", "total_amount": 150000, "due_date": "..." }
  },
  "connection": {
    "type": "pppoe", "status": "online|offline",
    "configured_ip": "10.0.0.5", "current_ip": "10.0.0.5",
    "download": 10240, "upload": 5120,
    "realtime_download_mbps": 2.5, "realtime_upload_mbps": 1.0,
    "realtime_sampled_at": "...",
    "active_session": { "ip": "...", "started_at": "...", "uptime_seconds": 3600 }
  },
  "package": { "id": "...", "name": "Paket 10Mbps", "bandwidth_up": 10, "bandwidth_down": 10, "price": 150000 },
  "router": { "id": "...", "name": "Router-01", "host": "192.168.1.1" },
  "ont": { "id": "...", "serial": "ZTEG12345678", "status": "online" }
}
```

### `PUT /customers/:id` — Full update (permission: `customers.edit`)
### `PUT /customers/:id/profile` — Update name/phone/email/address/nik/notes
### `PUT /customers/:id/access` — Update PPPoE credentials / IP
### `PUT /customers/:id/service` — Update package/router/odp assignment
### `DELETE /customers/:id` — Soft delete (permission: `customers.delete`)
### `POST /customers/:id/isolate` — Isolir customer (permission: `customers.isolate`)
### `POST /customers/:id/activate` — Aktifkan customer (permission: `customers.isolate`)
### `GET /customers/:id/logs` — Customer activity logs
### `GET /customers/:id/ont` — ONT attached to customer

---

## Packages

### `GET /packages?page=1&per_page=20&search=`
### `POST /packages`
**Body:**
```json
{
  "name": "Paket 10Mbps", "description": "...",
  "bandwidth_up": 10, "bandwidth_down": 10,
  "price": 150000, "burst_limit": "20M/20M",
  "address_list": "pelanggan-aktif"
}
```
### `GET /packages/:id`
### `PUT /packages/:id`
### `DELETE /packages/:id`

---

## Invoices

### `GET /invoices?page=1&per_page=20&status=&customer_id=&month=&year=`
**Status values:** `unpaid` | `paid` | `overdue` | `cancelled`

### `POST /invoices`
**Body:**
```json
{
  "customer_id": "...", "period_month": 5, "period_year": 2026,
  "package_price": 150000, "discount": 0, "additional_fee": 0,
  "fee_description": "", "due_date": "2026-05-10", "notes": ""
}
```

### `POST /invoices/generate`
Bulk generate invoices for all active customers.
**Body:** `{ "period_month": 5, "period_year": 2026 }`

### `GET /invoices/:id`
### `PUT /invoices/:id`
### `DELETE /invoices/:id`

### `POST /invoices/:id/pay`
Manual payment record. Permission: `invoices.pay`
**Body:**
```json
{
  "amount": 150000, "payment_method": "cash|transfer|qris",
  "notes": "lunas via transfer BCA"
}
```

### `POST /invoices/:id/pay-gateway`
Create payment via Tripay/Midtrans. Permission: `invoices.pay`
**Body:**
```json
{
  "payment_method": "BRIVA|BNIVA|MANDIRIVA|QRIS|...",
  "customer_name": "...", "customer_email": "...", "customer_phone": "...",
  "return_url": "https://frontend.com/invoices/pay/result"
}
```
**Response:**
```json
{
  "payment_id": "...", "gateway_trx_id": "T12345...",
  "payment_url": "https://tripay.co.id/...",
  "expired_at": "..."
}
```

### `GET /invoices/:id/payments`
### `GET /payments?page=1&per_page=20&status=&gateway=`
All payments across invoices.

---

## Tickets

### `GET /tickets?page=1&per_page=20&status=&priority=&customer_id=`
**Status values:** `open` | `in_progress` | `resolved` | `closed`  
**Priority values:** `low` | `medium` | `high` | `urgent`

### `POST /tickets`
**Body:**
```json
{
  "customer_id": "...", "title": "Internet mati",
  "description": "...", "priority": "high", "category": "..."
}
```

### `GET /tickets/:id`
### `PUT /tickets/:id`
### `DELETE /tickets/:id`
### `PUT /tickets/:id/status` — `{ "status": "resolved" }`
### `PUT /tickets/:id/assign` — `{ "user_id": "..." }` (permission: `tickets.assign`)
### `GET /tickets/:id/messages`
### `POST /tickets/:id/messages` — `{ "message": "...", "attachment_url": "..." }`

---

## Routers (MikroTik)

### `GET /routers?page=1&per_page=20&search=`
### `POST /routers`
**Body:**
```json
{
  "name": "Router-01", "host": "192.168.1.1", "port": 8728,
  "username": "admin", "password": "secret",
  "description": "...", "is_active": true
}
```
### `GET /routers/:id`
### `PUT /routers/:id`
### `DELETE /routers/:id`
### `GET /routers/:id/sessions` — Active PPPoE sessions
### `GET /routers/:id/traffic` — Current bandwidth usage
### `POST /routers/:id/test` — Test MikroTik connection
### `POST /routers/:id/sync` — Sync sessions from MikroTik
### `POST /routers/:id/regenerate-token` — Generate new API token
### `GET /routers/:id/mikrotik-config` — Get MikroTik script config
### `GET /routers/:id/connection-logs`
### `POST /routers/:id/vpn-key` — Register WireGuard VPN key
### `GET /routers/vpn/peers` — All VPN peers
### `POST /routers/heartbeat` — Public (no auth). MikroTik heartbeat endpoint.

---

## OLT (Optical Line Terminal)

### `GET /olts?page=1&per_page=20&search=`
### `POST /olts`
**Body:**
```json
{
  "name": "OLT-01", "brand": "ZTE|Huawei|...",
  "host": "192.168.0.1", "port": 161,
  "snmp_community": "public", "description": "..."
}
```
### `GET /olts/:id`
### `PUT /olts/:id`
### `DELETE /olts/:id`
### `GET /olts/:id/snmp/monitor` — SNMP real-time monitoring
### `GET /olts/:id/pon-ports`
### `POST /olts/:id/pon-ports`
### `PUT /olts/:id/pon-ports/:portId`
### `DELETE /olts/:id/pon-ports/:portId`

---

## ODP (Optical Distribution Point)

### `GET /odps?page=1&per_page=20&search=&olt_id=`
### `POST /odps`
**Body:**
```json
{
  "name": "ODP-A01", "address": "...", "latitude": 3.571, "longitude": 98.718,
  "olt_id": "...", "pon_port_id": "...", "capacity": 8, "description": "..."
}
```
### `GET /odps/:id`
### `PUT /odps/:id`
### `DELETE /odps/:id`
### `GET /odps/:id/ports`
### `POST /odps/:id/ports`
### `PUT /odps/:id/ports/:portId`
### `DELETE /odps/:id/ports/:portId`

**Splitters:**
### `GET /splitters`
### `POST /splitters`
### `GET /splitters/:id`
### `PUT /splitters/:id`
### `DELETE /splitters/:id`

---

## ONT (Optical Network Terminal)

### `GET /onts?page=1&per_page=20&search=&olt_id=`
### `POST /onts`
**Body:**
```json
{
  "serial": "ZTEG12345678", "mac": "...", "brand": "ZTE|Huawei",
  "model": "F670L", "olt_id": "...", "pon_port_id": "...",
  "customer_id": "...", "description": "..."
}
```
### `GET /onts/:id`
### `PUT /onts/:id`
### `DELETE /onts/:id`

**GenieACS ONT actions (require `genieacs.manage`):**
### `POST /onts/:id/sync` — Sync ONT data from GenieACS
### `POST /onts/:id/reboot` — Reboot device via TR-069
### `POST /onts/:id/wifi` — Set WiFi SSID/password
**Body:** `{ "ssid": "MyWiFi", "password": "secret123", "band": "2.4|5|both" }`
### `POST /onts/:id/provision` — Provision ONT on GenieACS
### `GET /onts/:id/diagnostics` — Diagnostics (ping, traceroute, connection test)
### `POST /onts/:id/pppoe` — Set PPPoE username/password on device

---

## GenieACS (TR-069 Device Management)

Permission: `genieacs.view` / `genieacs.manage`

### `GET /genieacs/devices?page=1&per_page=20&search=`
### `GET /genieacs/devices/:id`
### `GET /genieacs/devices/serial/:serial`
### `GET /genieacs/devices/serial/:serial/status`
### `POST /genieacs/devices/serial/:serial/reboot`
### `POST /genieacs/devices/serial/:serial/wifi`
**Body:** `{ "ssid": "...", "password": "..." }`
### `GET /genieacs/tasks`
### `GET /genieacs/ont-dashboard` — Overview of all ONT statuses
### `POST /genieacs/bulk-provision`
### `POST /genieacs/discover` — Discover unregistered ONTs
### `POST /genieacs/auto-match` — Auto-match discovered ONTs to customers

---

## Bandwidth

Permission: `bandwidth.view`

### `GET /bandwidth/summary`
### `GET /bandwidth/top-users?limit=10&period=day|week|month`
### `GET /bandwidth/saturation`
### `GET /bandwidth/customers/:id`
### `GET /bandwidth/customers/:id/history?period=day|week|month`
### `GET /bandwidth/customers/:id/connections`

---

## IPAM (IP Address Management)

### IP Pools

### `GET /ip-pools?page=1&per_page=20`
### `POST /ip-pools`
**Body:** `{ "name": "Pool-A", "subnet": "10.10.0.0/24", "gateway": "10.10.0.1", "description": "..." }`
### `GET /ip-pools/:id`
### `PUT /ip-pools/:id`
### `DELETE /ip-pools/:id`
### `GET /ip-pools/:poolId/stats`
### `GET /ip-pools/:poolId/available`
### `GET /ip-pools/:poolId/addresses?page=1&per_page=20&status=`

**Status values:** `available` | `assigned` | `reserved`

### `POST /ip-pools/:poolId/addresses`
**Body:** `{ "address": "10.10.0.5", "description": "..." }`

### `POST /ip-pools/:poolId/addresses/batch`
**Body:** `{ "start": "10.10.0.10", "end": "10.10.0.50" }`

### `POST /ip-pools/:poolId/addresses/:addrId/assign`
**Body:** `{ "customer_id": "..." }`

### `POST /ip-pools/:poolId/addresses/:addrId/release`

---

## FTTH

### `GET /ftth/stats`
### `GET /ftth/map` — GeoJSON map of all ODPs and customers

---

## Expenses

### `GET /expense-categories`
### `POST /expense-categories` — `{ "name": "...", "description": "..." }`
### `GET /expense-categories/:id`
### `PUT /expense-categories/:id`
### `DELETE /expense-categories/:id`

### `GET /expenses?page=1&per_page=20&category_id=&month=&year=`
### `GET /expenses/summary?month=5&year=2026`
### `POST /expenses`
**Body:**
```json
{
  "category_id": "...", "amount": 500000,
  "description": "Beli kabel fiber 100m",
  "date": "2026-05-01", "notes": ""
}
```
### `GET /expenses/:id`
### `PUT /expenses/:id`
### `DELETE /expenses/:id`

---

## Reports

Permission: `reports.view` (export requires `reports.export`)

### `GET /reports/revenue?month=5&year=2026`
### `GET /reports/customers?year=2026`
### `GET /reports/customer-growth?year=2026`
### `GET /reports/payments?month=5&year=2026`
### `GET /reports/payment-breakdown?month=5&year=2026`
### `GET /reports/collection-rate?month=5&year=2026`
### `GET /reports/profit-loss?month=5&year=2026`
### `GET /reports/vouchers?month=5&year=2026`
### `GET /reports/voucher-sales?month=5&year=2026`

**Exports (returns file download):**
### `GET /reports/export/revenue/excel?month=5&year=2026`
### `GET /reports/export/revenue/pdf?month=5&year=2026`
### `GET /reports/export/customer-growth/excel?year=2026`
### `GET /reports/export/customer-growth/pdf?year=2026`
### `GET /reports/export/profit-loss/excel?month=5&year=2026`
### `GET /reports/export/profit-loss/pdf?month=5&year=2026`

---

## Dashboard

Permission: `dashboard.view`

### `GET /dashboard/stats`
**Response:**
```json
{
  "total_customers": 450, "active_customers": 420,
  "isolated_customers": 10, "inactive_customers": 20,
  "total_revenue_month": 63000000,
  "unpaid_invoices": 35, "overdue_invoices": 8,
  "open_tickets": 12, "total_routers": 5, "online_routers": 5
}
```

### `GET /dashboard/revenue?year=2026`

---

## Vouchers

### Voucher Products
### `GET /voucher-products?page=1&per_page=20`
### `POST /voucher-products`
**Body:**
```json
{
  "name": "Voucher 1 Hari", "description": "...",
  "duration_hours": 24, "bandwidth_up": 5, "bandwidth_down": 5,
  "price": 5000, "quota_mb": 0
}
```
### `GET /voucher-products/:id`
### `PUT /voucher-products/:id`
### `DELETE /voucher-products/:id`

### Vouchers
### `GET /vouchers?page=1&per_page=20&product_id=&status=`
**Status:** `unused` | `used` | `expired`
### `POST /vouchers/generate`
**Body:** `{ "product_id": "...", "quantity": 50, "prefix": "DC" }`
### `GET /vouchers/:id`
### `DELETE /vouchers/:id`

---

## Broadcasts

Permission: `broadcasts.view`

### `GET /broadcasts?page=1&per_page=20&status=`
### `POST /broadcasts`
**Body:**
```json
{
  "title": "Pengumuman pemeliharaan",
  "message": "Halo {nama}, ada pemeliharaan...",
  "template_id": "...",
  "filter": { "status": "active", "package_id": "..." },
  "scheduled_at": "2026-05-15T09:00:00Z"
}
```
### `GET /broadcasts/:id`
### `DELETE /broadcasts/:id`

### WhatsApp Broadcast Templates
### `GET /whatsapp/templates`
### `POST /whatsapp/templates`
**Body:** `{ "name": "Reminder Tagihan", "content": "Halo {nama}, tagihan bulan {bulan} sebesar Rp {nominal}..." }`
### `GET /whatsapp/templates/:id`
### `PUT /whatsapp/templates/:id`
### `DELETE /whatsapp/templates/:id`

**Template Variables:** `{nama}`, `{kode_pelanggan}`, `{nominal}`, `{bulan}`, `{tahun}`, `{jatuh_tempo}`, `{paket}`, `{router}`

---

## Reminders

Permission: `reminders.view`

### `GET /reminders?page=1&per_page=20`
### `POST /reminders`
**Body:**
```json
{
  "name": "Reminder H-3",
  "type": "before_due|on_due|after_due",
  "days_offset": 3,
  "agenda": "Pengingat jatuh tempo",
  "message_template": "Tagihan {nama} jatuh tempo {jatuh_tempo}",
  "is_active": true
}
```

**Field descriptions:**
| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | ✅ | Nama pengingat |
| `type` | enum | ✅ | `before_due` (sebelum JT), `on_due` (saat JT), `after_due` (setelah JT) |
| `days_offset` | number | ⚠️ | Jumlah hari offset. **Tidak dipakai** jika `type = "on_due"` |
| `agenda` | string | ❌ | Subjek/agenda pesan WhatsApp |
| `message_template` | string | ✅ | Template pesan. Mendukung format WhatsApp (`*tebal*`, `_miring_`) |
| `is_active` | boolean | ❌ | Default `true` |

**Template Variables:** `{salam}`, `{nama}`, `{kode_pelanggan}`, `{nomor_invoice}`, `{periode}`, `{paket}`, `{jumlah}`, `{jatuh_tempo}`

### `GET /reminders/:id`
### `PUT /reminders/:id`
### `DELETE /reminders/:id`
### `POST /reminders/trigger` — Manually trigger reminder sending. Response: `{ "sent": 5 }`

---

## Notifications (Push/Admin)

Permission: `notifications.view`

### `GET /notifications?page=1&per_page=20&read=false`
### `PUT /notifications/:id/read`
### `POST /notifications/send` — Send FCM push to specific user
### `POST /notifications/broadcast` — Send FCM to all users

---

## Resellers

Permission: `resellers.view`

### `GET /resellers?page=1&per_page=20&search=`
### `POST /resellers`
**Body:**
```json
{
  "name": "Reseller A", "phone": "0812...", "email": "...",
  "address": "...", "commission_rate": 5.0, "notes": ""
}
```
### `GET /resellers/:id`
### `PUT /resellers/:id`
### `DELETE /resellers/:id`
### `GET /resellers/:id/customers`
### `GET /resellers/:id/commissions?page=1&per_page=20&status=`
**Status:** `pending` | `paid`
### `POST /resellers/:id/commissions`
### `POST /resellers/:id/commissions/pay-all` — Mark all pending commissions paid
### `POST /resellers/:id/commission-summary`
### `POST /resellers/commissions/:commissionId/pay`

---

## Rewards & Referrals

Permission: `rewards.view`

### `GET /rewards?page=1&per_page=20`
### `POST /rewards`
**Body:**
```json
{
  "name": "Referral Bonus", "description": "...",
  "type": "referral|loyalty|promo",
  "value": 15000, "value_type": "fixed|percentage",
  "min_invoices": 3, "is_active": true,
  "start_date": "2026-05-01T00:00:00Z",
  "end_date": "2026-12-31T23:59:59Z"
}
```
### `GET /rewards/stats`
### `GET /rewards/:id`
### `PUT /rewards/:id`
### `DELETE /rewards/:id`

### `GET /referrals?page=1&per_page=20&status=`
**Status:** `pending` | `qualified` | `rewarded` | `expired`

### `GET /reward-claims?page=1&per_page=20&status=`
**Status:** `pending` | `applied` | `expired`

### `GET /reward-claims/dashboard`

---

## WhatsApp

Permission: `whatsapp.view`

### `GET /whatsapp/config`
### `PUT /whatsapp/config` — Permission: `whatsapp.configure`
### `GET /whatsapp/logs`
### `POST /whatsapp/sessions/start`
### `GET /whatsapp/sessions/status`
### `GET /whatsapp/sessions/qr` — Returns QR code image/base64
### `DELETE /whatsapp/sessions` — Stop session
### `POST /whatsapp/send`
**Body:** `{ "phone": "628xxx", "message": "Halo..." }`

---

## Settings

### `GET /settings` — All settings as key-value
### `GET /settings/:key`
### `POST /settings` — `{ "key": "...", "value": "..." }`
### `PUT /settings/bulk` — `{ "settings": { "key1": "val1", "key2": "val2" } }`
### `DELETE /settings/:key`
### `GET /settings/theme` — Get active tenant theme
### `PUT /settings/theme` — Update theme (permission: `settings.edit`)

### User Preferences
### `GET /user/preferences`
### `PUT /user/preferences`
**Body:** `{ "theme": "light|dark", "language": "id|en", "sidebar_collapsed": false }`

---

## Audit Logs

Role: `superadmin` only

### `GET /audit-logs?page=1&per_page=20&user_id=&action=&resource=`

---

## Tenant Subscription

### `GET /subscription/plans`
### `GET /subscription/plans/:id`
### `POST /subscription/subscribe` — `{ "plan_id": "...", "payment_method": "..." }`
### `GET /subscription/orders`
### `GET /subscription/orders/:id`
### `POST /subscription/orders/:id/pay`
### `POST /subscription/orders/:id/confirm` — Role: `superadmin`

---

## Customer Portal (Self-service)

Auth: Customer JWT token from portal login.

### Public (no auth):
- `GET /public/portal/:slug` — Get tenant info
- `POST /public/portal/:slug/login` — `{ "phone": "...", "password": "..." }`
- `POST /public/portal/:slug/reset-pin` — Request PIN reset
- `POST /public/portal/:slug/reset-password` — `{ "pin": "...", "password": "..." }`

### Authenticated customer:
- `GET /portal/profile`
- `GET /portal/customer`
- `GET /portal/invoices`
- `GET /portal/invoices/:id`
- `GET /portal/invoices/:id/payments`
- `GET /portal/tickets`
- `POST /portal/tickets` — `{ "title": "...", "description": "..." }`
- `GET /portal/tickets/:id`
- `GET /portal/tickets/:id/messages`
- `POST /portal/tickets/:id/messages`
- `PUT /portal/change-password` — `{ "old_password": "...", "new_password": "..." }`

---

## Public Payment Page

No auth required. Base: `/api/v1/public/pay`

- `GET /public/pay/:tenant_id/:customer_code` — Get customer info for payment
- `POST /public/pay/:tenant_id/:customer_code` — Create payment
- `GET /public/pay/check/:trx_id` — Check payment status

---

## Public Voucher Store

No auth required. Base: `/api/v1/public/store`

- `GET /public/store/:tenant_slug` — List available voucher products
- `POST /public/store/:tenant_slug/buy` — Purchase voucher

---

## Mobile App

- `POST /mobile/auth/login` — `{ "phone": "...", "password": "..." }`

---

## Isolir Page

- `GET /isolir/:tenant_slug/:customer_code` — Redirect page shown to isolated customers

---

## Webhook Endpoints (Payment Gateways)

All are public POST endpoints — no auth, signature-verified internally.

> ⚠️ **UPDATE (v1.1):** Ditambahkan 3 endpoint baru untuk Xendit.

| Endpoint | Gateway | Flow |
|---|---|---|
| `POST /webhooks/tripay` | Tripay | Invoice payment |
| `POST /webhooks/midtrans` | Midtrans | Invoice payment |
| `POST /webhooks/xendit` | **Xendit** *(baru)* | Invoice payment |
| `POST /webhooks/tripay/voucher` | Tripay | Voucher purchase |
| `POST /webhooks/midtrans/voucher` | Midtrans | Voucher purchase |
| `POST /webhooks/xendit/voucher` | **Xendit** *(baru)* | Voucher purchase |
| `POST /webhooks/subscription/tripay` | Tripay | SaaS subscription |
| `POST /webhooks/subscription/midtrans` | Midtrans | SaaS subscription |
| `POST /webhooks/subscription/xendit` | **Xendit** *(baru)* | SaaS subscription |

**Xendit Webhook Verification:**  
Xendit mengirim header `x-callback-token` di setiap request. Backend memverifikasi token ini dengan `pg_secret_key` yang tersimpan di tenant settings. Tidak perlu tindakan khusus dari frontend.

---

## SuperAdmin (Pengelola)

> **Role:** `superadmin` only. Cross-tenant management endpoints.

### `GET /admin/dashboard`
SuperAdmin dashboard overview — all stats across all tenants.

**Response:**
```json
{
  "data": {
    "total_tenants": 15,
    "active_tenants": 12,
    "free_plan_tenants": 8,
    "pro_plan_tenants": 5,
    "enterprise_tenants": 2,
    "total_customers": 4500,
    "active_customers": 3800,
    "inactive_customers": 700,
    "total_routers": 25,
    "online_routers": 22,
    "total_revenue": 125000000,
    "subscriber_count": 7,
    "tenant_stats": [
      {
        "tenant_id": "...", "tenant_name": "ISP A", "slug": "isp-a",
        "plan": "pro", "is_active": true,
        "total_customers": 450, "active_customers": 420, "inactive_customers": 30,
        "total_routers": 5, "online_routers": 5
      }
    ]
  }
}
```

### `GET /admin/tenants`
Per-tenant statistics (customer counts, router counts, plan info).

**Response:**
```json
{
  "data": [
    {
      "tenant_id": "...", "tenant_name": "ISP A", "slug": "isp-a",
      "plan": "pro", "is_active": true,
      "total_customers": 450, "active_customers": 420, "inactive_customers": 30,
      "total_routers": 5, "online_routers": 5
    }
  ]
}
```

### `GET /admin/routers?page=1&per_page=20`
All routers across all tenants with monitoring info.

**Response:**
```json
{
  "data": [
    {
      "tenant_id": "...", "tenant_name": "ISP A",
      "router_id": "...", "router_name": "Router-01",
      "host": "192.168.1.1", "is_active": true, "last_seen": "2026-05-27T09:00:00Z"
    }
  ],
  "total": 25, "page": 1, "per_page": 20
}
```

### `GET /admin/customers`
Customer counts per tenant (active/inactive breakdown).

**Response:** Same format as `/admin/tenants` — array of tenant stats with customer counts.

---

## WebSocket

### `GET /ws`
Real-time notifications. After JWT auth, server pushes events:

```json
{ "type": "notification", "data": { "id": "...", "message": "...", "created_at": "..." } }
{ "type": "customer_status", "data": { "customer_id": "...", "status": "online" } }
```

---

## i18n

- `GET /languages` — Available languages
- `GET /translations?lang=id` — Translation strings

---

## Permissions Reference

All permission keys used in `permissions[]` array:

| Group | Keys |
|---|---|
| Dashboard | `dashboard.view` |
| Pelanggan | `customers.view`, `customers.create`, `customers.edit`, `customers.delete`, `customers.isolate` |
| Paket | `packages.view`, `packages.create`, `packages.edit`, `packages.delete` |
| Tagihan | `invoices.view`, `invoices.create`, `invoices.edit`, `invoices.delete`, `invoices.pay` |
| Tiket | `tickets.view`, `tickets.create`, `tickets.edit`, `tickets.delete`, `tickets.assign` |
| Router | `routers.view`, `routers.create`, `routers.edit`, `routers.delete` |
| OLT | `olts.view`, `olts.create`, `olts.edit`, `olts.delete` |
| ODP | `odps.view`, `odps.create`, `odps.edit`, `odps.delete` |
| ONT | `onts.view`, `onts.create`, `onts.edit`, `onts.delete` |
| FTTH | `ftth.view` |
| Voucher | `vouchers.view`, `vouchers.create`, `vouchers.delete` |
| Pengeluaran | `expenses.view`, `expenses.create`, `expenses.edit`, `expenses.delete` |
| Broadcast | `broadcasts.view`, `broadcasts.create`, `broadcasts.delete` |
| Laporan | `reports.view`, `reports.export` |
| Bandwidth | `bandwidth.view` |
| Reward | `rewards.view`, `rewards.create`, `rewards.edit`, `rewards.delete` |
| Reseller | `resellers.view`, `resellers.create`, `resellers.edit`, `resellers.delete` |
| IPAM | `ip_pools.view`, `ip_pools.create`, `ip_pools.edit`, `ip_pools.delete` |
| WhatsApp | `whatsapp.view`, `whatsapp.configure`, `whatsapp.send` |
| Pengaturan | `settings.view`, `settings.edit` |
| User | `users.view`, `users.create`, `users.edit`, `users.delete` |
| Role | `roles.view`, `roles.create`, `roles.edit`, `roles.delete` |
| Notifikasi | `notifications.view`, `notifications.send` |
| Reminder | `reminders.view`, `reminders.create`, `reminders.edit`, `reminders.delete` |
| GenieACS | `genieacs.view`, `genieacs.manage` |

---

## Common Query Parameters

| Parameter | Description |
|---|---|
| `page` | Page number (default: 1) |
| `per_page` | Items per page (default: 20) |
| `search` | Full-text search |
| `sort` | Field to sort by |
| `order` | `asc` or `desc` |

## HTTP Status Codes

| Code | Meaning |
|---|---|
| 200 | OK |
| 201 | Created |
| 400 | Bad request / validation error |
| 401 | Unauthorized (missing/invalid token) |
| 403 | Forbidden (insufficient permission) |
| 404 | Not found |
| 422 | Unprocessable entity |
| 500 | Internal server error |

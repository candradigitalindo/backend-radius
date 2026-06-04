# Setup Web Push (PWA) — Firebase Cloud Messaging

Notifikasi push portal pelanggan memakai **Firebase Cloud Messaging (FCM) HTTP v1**.
API legacy (server key) sudah dimatikan Google sejak Juni 2024, jadi wajib pakai
service account (v1). Dokumen ini langkah lengkap dari nol.

## Ringkasan arsitektur

- **Backend** mengirim push via Firebase Admin SDK (butuh *service account JSON*).
- **Frontend** (portal) minta izin notifikasi, ambil *FCM token*, daftarkan ke
  `POST /api/v1/portal/push/register`. Butuh *web app config* + *VAPID key*.
- Token disimpan di tabel `push_devices`. Saat ada notifikasi (mis. tagihan),
  backend kirim ke semua token milik pelanggan tsb.

---

## 1. Buat / siapkan project Firebase

1. Buka <https://console.firebase.google.com> → **Add project** (atau pakai yang ada).
2. Aktifkan **Cloud Messaging** (otomatis aktif).

## 2. Kredensial BACKEND (service account)

1. Firebase Console → ⚙️ **Project settings** → tab **Service accounts**.
2. Klik **Generate new private key** → unduh file JSON.
3. Simpan sebagai:
   ```
   backend-radius/keys/fcm-service-account.json
   ```
   (folder `keys/` sudah ter-mount ke `/app/keys` di container).
4. Edit `backend-radius/.env`:
   ```env
   FCM_PROJECT_ID=<project_id-dari-json>
   FCM_CREDENTIALS_FILE=./keys/fcm-service-account.json
   FCM_ENABLED=true
   ```
5. Rebuild + restart `app` dan `worker`. Saat start, log harus muncul:
   ```
   [notification] FCM HTTP v1 aktif (project: <project_id>)
   ```
   Kalau gagal, log `[notification] FCM dinonaktifkan: ...` — push mati, tapi
   in-app + WhatsApp tetap jalan.

## 3. Kredensial FRONTEND (web config + VAPID)

1. Project settings → tab **General** → bagian **Your apps** → **Add app** → **Web** (`</>`).
2. Salin objek `firebaseConfig` (apiKey, authDomain, projectId, messagingSenderId, appId).
3. Project settings → **Cloud Messaging** → **Web Push certificates** →
   **Generate key pair** → salin **Key pair** (ini VAPID key).
4. Edit `frontend-radius/.env`:
   ```env
   VITE_FIREBASE_API_KEY=...
   VITE_FIREBASE_AUTH_DOMAIN=...
   VITE_FIREBASE_PROJECT_ID=...
   VITE_FIREBASE_MESSAGING_SENDER_ID=...
   VITE_FIREBASE_APP_ID=...
   VITE_FIREBASE_VAPID_KEY=...
   ```
5. Rebuild frontend (image nginx) dan deploy.

> Selama `VITE_FIREBASE_*` kosong, tombol **Aktifkan Notifikasi** di halaman
> Profil portal otomatis **tersembunyi** — jadi aman di-deploy sebelum diisi.

## 4. Cara kerja di sisi pelanggan

1. Pelanggan buka portal → menu **Profil** → klik **Aktifkan Notifikasi**.
2. Browser minta izin → pelanggan **Allow**.
3. Service worker `firebase-messaging-sw.js` terdaftar, token dikirim ke backend.
4. Mulai sekarang notifikasi tampil walau tab portal ditutup (selama browser hidup).

## Catatan teknis

- **HTTPS wajib.** Web push hanya jalan di origin aman (production sudah HTTPS).
- **iOS Safari** mendukung web push hanya jika portal di-**Add to Home Screen**
  (iOS 16.4+). Android Chrome jalan langsung.
- Token mati (uninstall/clear data) otomatis dinonaktifkan backend saat kirim
  (`messaging.IsUnregistered`).
- Service worker mengambil konfigurasi via query string saat registrasi
  (`/firebase-messaging-sw.js?apiKey=...`), jadi tidak ada kunci yang di-hardcode.

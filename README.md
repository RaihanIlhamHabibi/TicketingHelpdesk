# Helpdesk Internal

Aplikasi helpdesk internal untuk mengelola tiket karyawan. Monorepo ini berisi REST API Go, frontend Vue 3, dan PostgreSQL. Implementasi memakai asumsi bahwa akun publik selalu berperan `requester`; akun `agent` dan `admin` disediakan melalui seed untuk kebutuhan operasional/demo. Login memakai JWT bearer token, dan catatan internal hanya dapat dibaca oleh agent/admin.

## Menjalankan dengan Docker

Pastikan Docker dan Docker Compose tersedia, lalu dari root project jalankan:

```sh
docker compose up --build
```

- UI: http://localhost:8081 by default (`FRONTEND_PORT`)
- API: http://localhost:8080/api by default (`API_PORT`)
- PostgreSQL: port internal `5432`

Sebelum menjalankan Compose, salin `.env.example` menjadi `.env` (`Copy-Item .env.example .env` di PowerShell atau `cp .env.example .env` di shell POSIX) dan ganti **semua** nilai placeholder dengan nilai unik. Compose akan menolak startup bila variabel wajib belum diisi; file `.env` tidak masuk Git. Password database harus URL-safe (huruf, angka, underscore) karena digunakan dalam `DATABASE_URL`. Gunakan secret dan password berbeda di setiap environment. Migrasi dan seed dijalankan oleh backend saat startup. Password seed hanya dipakai ketika akun seed pertama kali dibuat.
Jika port `8080` atau `8081` sudah digunakan, ubah `API_PORT` atau `FRONTEND_PORT` di `.env`.

### Akun seed lokal

| Peran | Email | Password lokal |
|---|---|---|
| Requester | `requester@example.com` | Nilai `SEED_REQUESTER_PASSWORD` di `.env` |
| Agent | `agent@example.com` | Nilai `SEED_AGENT_PASSWORD` di `.env` |
| Admin | `admin@example.com` | Nilai `SEED_ADMIN_PASSWORD` di `.env` |

Semua password seed dapat diubah lewat variabel `SEED_*_PASSWORD`.

## Menjalankan untuk pengembangan

Backend memerlukan Go 1.22+ dan PostgreSQL. Frontend memerlukan Node.js 20+.

1. Jalankan PostgreSQL dan set `DATABASE_URL`, `JWT_SECRET`, dan password seed. Contoh URL lokal:
   `postgres://helpdesk:helpdesk@localhost:5432/helpdesk?sslmode=disable`
2. Dari `backend/`, jalankan `go run ./cmd/api`.
3. Dari `frontend/`, jalankan `npm install` lalu `npm run dev`. Vite meneruskan `/api` ke backend lokal di `http://localhost:8080`.

Detail variabel konfigurasi dan perilaku startup tercantum di masing-masing folder.

## Endpoint API

Semua endpoint bertanda login memerlukan header `Authorization: Bearer <token>`. Error berbentuk JSON `{ "error": "pesan" }`.

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| POST | `/api/auth/register` | Publik | Daftar akun requester |
| POST | `/api/auth/login` | Publik | Login dan memperoleh JWT |
| GET | `/api/me` | Login | Profil akun aktif |
| GET | `/api/tickets` | Login | Daftar tiket, filter, pencarian, pagination, sorting |
| POST | `/api/tickets` | Login | Buat tiket |
| GET | `/api/tickets/{id}` | Login | Detail dan komentar; requester hanya tiket miliknya |
| PATCH | `/api/tickets/{id}` | Login | Ubah status/prioritas/assignee sesuai peran dan transisi |
| DELETE | `/api/tickets/{id}` | Admin; requester pemilik tiket `open` | Hapus tiket dan komentarnya; agent tidak diizinkan |
| POST | `/api/tickets/{id}/comments` | Login | Komentar atau catatan internal |
| GET | `/api/users` | Admin | Daftar pengguna |
| GET | `/api/agents` | Agent/Admin | Daftar agent/admin yang dapat ditugaskan |
| PATCH | `/api/users/{id}/role` | Admin | Ubah peran pengguna |

Query tiket mendukung `page`, `pageSize`, `status`, `priority`, `assignee` (ID atau `unassigned`), `search` (pencarian judul), `sortBy` (`createdAt`, `updatedAt`, `title`, `priority`, `status`) dan `sortOrder` (`asc`, `desc`). Status: `open`, `in_progress`, `resolved`, `closed`; prioritas: `low`, `medium`, `high`, `urgent`.

Dokumentasi API tersedia di [`docs/openapi.yaml`](docs/openapi.yaml).

## Test

Jalankan unit test backend:

```sh
cd backend
go test ./...
```

Frontend:

```sh
cd frontend
npm install
npm run build
```

CI tersedia melalui GitHub Actions dan menjalankan `go test ./...` serta `npm run build` pada push dan pull request.

## Arsitektur dan catatan

- Backend memisahkan transport HTTP, aturan bisnis, dan akses PostgreSQL agar aturan akses/status dapat diuji.
- Frontend menggunakan Vue Router untuk guard rute, Pinia untuk sesi pengguna, dan satu HTTP client untuk autentikasi/penanganan 401.
- Migrasi idempotent diterapkan saat backend mulai; Docker Compose menunggu database sehat sebelum menjalankan API.
- Upload lampiran, notifikasi real-time, SLA, dan statistik tidak termasuk implementasi. Penghapusan tiket dibatasi untuk admin dan requester pemilik tiket yang masih `open`; komentar terhapus melalui foreign-key cascade. Jika dilanjutkan, prioritas berikutnya adalah integrasi test berbasis database, rate limiting untuk autentikasi, dan observabilitas.

## Catatan pengumpulan

Implementasi awal dipublikasikan dalam satu commit besar (`dec9c14`), lalu perbaikan berikutnya dicatat pada commit terpisah (`5b3aa8e`). Riwayat sebelum implementasi bukan catatan tahapan pengembangan aplikasi, dan tidak dibuat ulang atau direkayasa. Mohon tinjau source code, test, dan commit perbaikan yang tersedia sebagai hasil akhir.

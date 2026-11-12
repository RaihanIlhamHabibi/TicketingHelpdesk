# Helpdesk Internal

Aplikasi helpdesk internal untuk mengelola tiket karyawan. Monorepo ini berisi REST API Go, frontend Vue 3, dan PostgreSQL. Implementasi memakai asumsi bahwa akun publik selalu berperan `requester`; akun `agent` dan `admin` disediakan melalui seed untuk kebutuhan operasional/demo. Login memakai JWT bearer token, dan catatan internal hanya dapat dibaca oleh agent/admin.

## Menjalankan dengan Docker

Pastikan Docker dan Docker Compose tersedia, lalu dari root project jalankan:

```sh
docker compose up --build
```

- UI: http://localhost:8081
- API: http://localhost:8080/api
- PostgreSQL: port internal `5432`

Compose menyediakan kredensial seed dan secret **khusus untuk pengembangan lokal** agar aplikasi dapat langsung dicoba. Untuk mengubahnya, salin `.env.example` menjadi `.env` lalu ganti nilainya sebelum menjalankan `docker compose up --build`. Jangan gunakan default lokal pada deployment. Migrasi dan seed dijalankan oleh backend saat startup.

### Akun seed lokal

| Peran | Email | Password lokal |
|---|---|---|
| Requester | `requester@example.com` | `Requester123!` |
| Agent | `agent@example.com` | `Agent123!` |
| Admin | `admin@example.com` | `Admin123!` |

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

## Arsitektur dan catatan

- Backend memisahkan transport HTTP, aturan bisnis, dan akses PostgreSQL agar aturan akses/status dapat diuji.
- Frontend menggunakan Vue Router untuk guard rute, Pinia untuk sesi pengguna, dan satu HTTP client untuk autentikasi/penanganan 401.
- Migrasi idempotent diterapkan saat backend mulai; Docker Compose menunggu database sehat sebelum menjalankan API.
- Upload lampiran, notifikasi real-time, SLA, statistik, dan pipeline CI tidak termasuk implementasi inti. Jika dilanjutkan, prioritas berikutnya adalah integrasi test berbasis database, rate limiting untuk autentikasi, dan observabilitas.

# boilerplate

Template Go untuk **microservice baru**: REST API dengan Gin, PostgreSQL + GORM,
autentikasi JWT, versioned migration, health check, Swagger, structured logging,
test, Docker, dan CI/CD.

## Fitur

- **Clean architecture**: `delivery → usecase → domain ← infrastructure`, semua
  koneksi antar-layer lewat interface.
- **Health check** dengan probe runtime dan PostgreSQL.
- **Autentikasi aman**: password bcrypt (cost 12), access token JWT HS256,
  validasi issuer/audience/expiry, dan endpoint profil yang dilindungi Bearer
  token.
- **Response konsisten** untuk semua endpoint: envelope `success`, `data`/`error`,
  dan `request_id`.
- **Tracing log**: setiap request punya `request_id` unik yang muncul di log
  `request → process → response` dan di response header/body.
- **PostgreSQL**: kredensial dari environment, query lewat GORM, migration
  versioned berbasis golang-migrate.
- **Swagger UI**, test (unit + integration), Docker multi-stage, dan CI/CD.

## Quick start

Prasyarat: Go 1.24+, PostgreSQL 14+.

```bash
cp .env.example .env

# isi JWT_SECRET dengan secret acak minimal 32 karakter
openssl rand -hex 32

# sesuaikan DB_USER / DB_PASSWORD / DB_NAME di .env, lalu buat database-nya
createdb boilerplate

make migrate-up   # buat tabel users
make run          # default port 1111
```

Atau langsung `go run ./cmd/api`. Port dapat diubah lewat `PORT`:

```bash
PORT=9090 make run
```

| URL | Keterangan |
| --- | --- |
| <http://localhost:1111/api/v1/health> | Health check (JSON) |
| <http://localhost:1111/swagger/index.html> | Swagger UI |
| <http://localhost:1111/> | Redirect ke Swagger UI |

Service **wajib** bisa terhubung ke PostgreSQL saat startup. Jalankan
`make migrate-up` sebelum service pertama kali dijalankan.

## Konfigurasi

Nilai dibaca dari environment. Jika file `.env` ada, file itu dimuat lebih dulu;
environment yang sudah ter-set tidak pernah ditimpa. `.env` sudah di-ignore git,
sedangkan `.env.example` ikut di-commit.

| Variabel | Default | Keterangan |
| --- | --- | --- |
| `PORT` | `1111` | Port HTTP |
| `SERVICE_NAME` | `boilerplate` | Nama service di health response dan log |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_DIR` | `logs` | Direktori log, dibuat otomatis |
| `DB_HOST` | `localhost` | Host PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_USER` | `postgres` | User PostgreSQL |
| `DB_PASSWORD` | `postgres` | Password PostgreSQL |
| `DB_NAME` | `boilerplate` | Nama database |
| `DB_SSLMODE` | `disable` | Mode SSL (`require` untuk produksi) |
| `DB_MAX_CONNS` | `10` | Maksimum koneksi pool |
| `DB_CONNECT_TIMEOUT` | `5s` | Timeout koneksi awal |
| `DB_AUTO_MIGRATE` | `false` | Jalankan migration saat startup |
| `DATABASE_URL` | — | Override penuh DSN PostgreSQL; menang atas `DB_*` |
| `JWT_SECRET` | — | **Wajib**, minimal 32 karakter |
| `JWT_ISSUER` | `boilerplate` | Claim `iss` |
| `JWT_AUDIENCE` | `boilerplate-api` | Claim `aud` |
| `JWT_TTL` | `1h` | Masa berlaku access token |

Contoh `DATABASE_URL`:

```text
postgres://user:password@localhost:5432/boilerplate?sslmode=disable
```

Version binary di-inject saat build:

```bash
make build VERSION=1.2.3
```

## Format response

Semua endpoint memakai envelope yang sama.

Sukses:

```json
{
  "success": true,
  "data": { "...": "..." },
  "request_id": "7f3a9c1d2e4b5a6f"
}
```

Gagal:

```json
{
  "success": false,
  "error": {
    "code": "invalid_credentials",
    "message": "invalid email or password"
  },
  "request_id": "7f3a9c1d2e4b5a6f"
}
```

`request_id` juga dikirim di header `X-Request-ID`. Client boleh mengirim header
tersebut; service akan memakainya kembali. ID yang terlalu panjang (>128
karakter) akan diganti dengan ID baru.

Error code yang dipakai:

| Code | HTTP | Arti |
| --- | --- | --- |
| `invalid_request` | 400 | Body bukan JSON yang valid |
| `request_too_large` | 413 | Body melebihi 1 MiB |
| `invalid_name` | 400 | Nama kosong/terlalu panjang |
| `invalid_email` | 400 | Format email tidak valid |
| `weak_password` | 400 | Password di luar 8–72 karakter |
| `email_taken` | 409 | Email sudah terdaftar |
| `unauthorized` | 401 | Bearer token tidak ada/tidak valid |
| `invalid_credentials` | 401 | Email/password salah |
| `route_not_found` | 404 | Route tidak dikenal |
| `method_not_allowed` | 405 | Method tidak didukung |
| `internal_server_error` | 500 | Kesalahan internal (detail hanya di log) |

## Endpoint

| Method | Path | Auth | Deskripsi |
| --- | --- | --- | --- |
| GET | `/api/v1/health` | — | Status service + dependency |
| POST | `/api/v1/auth/register` | — | Registrasi user |
| POST | `/api/v1/auth/token` | — | Tukar email/password menjadi JWT |
| GET | `/api/v1/auth/me` | Bearer | Profil user yang sedang login |
| GET | `/swagger/*` | — | Swagger UI & OpenAPI spec |
| GET | `/` | — | Redirect ke Swagger UI |

### Registrasi

```bash
curl -s -X POST http://localhost:1111/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Jane Doe","email":"jane@example.com","password":"correct-horse-battery"}'
```

```json
{
  "success": true,
  "data": {
    "id": "6f1c3a2b-6a2f-4f7c-9d0d-3b8c0f0f0f0f",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "created_at": "2026-10-27T22:00:00Z",
    "updated_at": "2026-10-27T22:00:00Z"
  },
  "request_id": "7f3a9c1d2e4b5a6f"
}
```

### Generate token

```bash
curl -s -X POST http://localhost:1111/api/v1/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"correct-horse-battery"}'
```

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "expires_at": "2026-10-27T23:00:00Z",
    "user": { "id": "...", "name": "Jane Doe", "email": "jane@example.com" }
  },
  "request_id": "7f3a9c1d2e4b5a6f"
}
```

### Profil (protected)

```bash
TOKEN='<access_token>'

curl -s http://localhost:1111/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

Tanpa token atau dengan token yang tidak valid, response adalah `401` dengan
header `WWW-Authenticate: Bearer realm="api"`.

### Health

```bash
curl -s http://localhost:1111/api/v1/health
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "service": "boilerplate",
    "version": "1.0.0",
    "timestamp": "2026-10-27T15:19:04Z",
    "checks": [
      { "name": "runtime", "status": "up", "detail": "uptime=7s" },
      { "name": "postgres", "status": "up", "detail": "connected" }
    ]
  },
  "request_id": "7f3a9c1d2e4b5a6f"
}
```

Jika dependency gagal, `data.status` menjadi `degraded` dan HTTP code `503`.
Detail error asli hanya masuk log; response memakai `"detail": "unavailable"`.

## Autentikasi

- Password di-hash dengan **bcrypt** cost 12 dan tidak pernah diserialisasi
  (`json:"-"`).
- Token dibuat dengan **golang-jwt/jwt/v5** memakai HMAC-SHA256.
- Verifikasi memeriksa signature, algoritma, `iss`, `aud`, dan wajib ada `exp`.
- Email dinormalisasi (trim + lowercase) sebelum disimpan/dicari.
- Email tidak dikenal dan password salah sama-sama mengembalikan
  `invalid_credentials`; untuk email tidak dikenal tetap dilakukan bcrypt
  comparison agar timing tidak membocorkan apakah akun ada.
- `JWT_SECRET` minimal 32 karakter dan tidak pernah dicatat ke log.

## Database & migration

- Semua operasi CRUD aplikasi memakai **GORM** (`db.WithContext(ctx)`).
- File SQL versioned ada di `migrations/` dan di-embed ke binary.
- Runner migration memakai **golang-migrate** melalui `cmd/migrate`.

```bash
make migrate-up        # terapkan semua migration
make migrate-down      # rollback satu migration
make migrate-version   # lihat versi & status dirty
```

Migration pertama membuat tabel `users`:

```sql
CREATE TABLE users (
    id uuid PRIMARY KEY,
    name varchar(100) NOT NULL,
    email varchar(254) NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
```

Untuk container/produksi, jalankan migration sebagai langkah deploy terpisah
(lihat `docker-compose.yml`). Alternatif untuk development: set
`DB_AUTO_MIGRATE=true`.

## Logging & tracing

Hanya **logrus** yang dipakai (Gin, `net/http`, dan GORM diarahkan ke logrus).
Format JSON, output ke stdout dan `LOG_DIR/app.log`.

Setiap request menghasilkan tepat tiga tahap dengan `request_id` yang sama:

1. `event=request` — saat request diterima
2. `event=process` — dari handler/usecase (mis. health, register, token)
3. `event=response` — setelah response dikirim, berisi status dan latency

Contoh:

```json
{"level":"info","msg":"request received","event":"request","request_id":"7f3a9c1d2e4b5a6f","method":"POST","path":"/api/v1/auth/token"}
{"level":"info","msg":"token generation completed","event":"process","request_id":"7f3a9c1d2e4b5a6f","action":"token"}
{"level":"info","msg":"response sent","event":"response","request_id":"7f3a9c1d2e4b5a6f","status":200,"latency_ms":42.1}
```

Status `4xx` dicatat sebagai `warning`, `5xx` sebagai `error`, dan panic dicatat
sebagai `error` lengkap dengan stack trace. Password, hash, JWT, dan DSN tidak
pernah dilog.

Catatan: belum ada rotasi file. Untuk produksi gunakan log collector/container
runtime atau tambahkan rotasi (misalnya `lumberjack`).

## Struktur

```
.
├── cmd/api/main.go                     # composition root: wiring + graceful shutdown
├── cmd/migrate/main.go                 # CLI migration (up/down/version)
├── internal
│   ├── config/                         # loader + validasi environment
│   ├── constant/                       # route, header, default, error code
│   ├── helper/                         # env, request ID, durasi
│   ├── domain/
│   │   ├── health.go                   # entity + port Checker
│   │   └── user.go                     # entity + port repository/hasher/token
│   ├── usecase/
│   │   ├── health.go                   # application service health
│   │   └── auth.go                     # register, token, profile
│   ├── infrastructure/
│   │   ├── checker/                    # runtime + PostgreSQL probe
│   │   ├── logger/                     # logrus + file writer
│   │   ├── postgres/                   # GORM repo, pool, migration runner
│   │   └── security/                   # bcrypt + JWT
│   └── delivery/httpapi/               # handler, middleware, router, envelope
├── migrations/                         # SQL versioned (di-embed)
├── docs/                               # hasil generate swag
├── docker-compose.yml                  # PostgreSQL + migrate + app
├── Dockerfile
└── Makefile
```

Arah dependensi:

```
delivery ──▶ usecase ──▶ domain ◀── infrastructure
                              ▲
                         cmd/api (wiring)
```

`domain` tidak mengimpor layer lain. Semua konstruksi objek hanya terjadi di
`cmd/api/main.go`.

## Testing & quality

```bash
make test          # semua test + race detector + coverage
make lint          # go vet
make fmt           # gofmt
make ci            # fmt-check + vet + test + build
```

Test mencakup domain, usecase, security (bcrypt/JWT), config, helper, middleware
(request/process/response + request ID), handler/router, health checker, dan
integration test PostgreSQL.

Integration test GORM otomatis di-skip jika tidak ada database:

```bash
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/boilerplate?sslmode=disable' \
  go test ./internal/infrastructure/postgres -race -count=1
```

Test tersebut menjalankan migration dan hanya menghapus data yang dibuatnya
sendiri.

## Docker

Cara termudah, memakai Compose (PostgreSQL + migration + app):

```bash
cp .env.example .env
# set JWT_SECRET dan kredensial DB di .env

make compose-up      # docker compose up --build
make compose-down
```

`docker-compose.yml` menjalankan migration sebagai service sekali jalan, lalu
menyalakan app setelah migration selesai.

Build/manual:

```bash
make docker VERSION=1.2.3
make docker-run

docker build --build-arg VERSION=1.2.3 -t boilerplate:1.2.3 .
docker run --rm -p 1111:1111 \
  -e SERVICE_NAME=my-service \
  -e DB_HOST=host.docker.internal \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=boilerplate \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -v boilerplate-logs:/app/logs \
  boilerplate:1.2.3
```

Image runtime memakai Alpine, berjalan sebagai non-root, membawa binary
`/app/migrate`, dan punya `HEALTHCHECK` ke endpoint health.

## CI/CD

`.github/workflows/ci.yml` berjalan pada push ke `main` dan pull request:

1. cek `gofmt`
2. `go vet`
3. test + race + coverage, dengan service PostgreSQL (termasuk integration test)
4. verifikasi `cmd/migrate`
5. `go build` dan `docker build`

`.github/workflows/release.yml` berjalan saat tag `v*` dibuat: build binary
linux/darwin (amd64/arm64) dengan version dari tag, lalu publish GitHub Release.

## Memakai template untuk service baru

1. Buat repo dari template ini (**Use this template** di GitHub, atau clone lalu
   hapus `.git`).
2. Ganti module path:

   ```bash
   OLD=github.com/sirojulanam/boilerplate
   NEW=github.com/<org>/<repo>

   go mod edit -module "$NEW"
   grep -rl "$OLD" --exclude-dir=.git --exclude-dir=logs . \
     | xargs perl -pi -e "s|\Q$OLD\E|$NEW|g"
   ```

3. Ganti nama service:
   - runtime: `SERVICE_NAME=nama-service`
   - default kode: `DefaultServiceName` di `internal/constant/constant.go`
   - binary: `APP` di `Makefile`
4. Ubah judul/deskripsi Swagger di `cmd/api/main.go`, lalu `make docs`.
5. Ganti migration pertama sesuai domain baru, lalu `make ci`.
6. Aktifkan **Settings → Template repository** di GitHub.

## Perintah Make

| Target | Fungsi |
| --- | --- |
| `make run` | Jalankan service |
| `make build` | Build binary ke `bin/$(APP)` |
| `make test` | Test + race + coverage |
| `make docs` | Regenerate Swagger |
| `make tidy` | `go mod tidy` |
| `make fmt` / `make fmt-check` | Format / cek format |
| `make lint` | `go vet` |
| `make ci` | Pipeline lokal lengkap |
| `make migrate-up` | Terapkan semua migration |
| `make migrate-down` | Rollback satu migration |
| `make migrate-version` | Lihat versi migration |
| `make docker` / `make docker-run` | Build / jalankan image |
| `make compose-up` / `make compose-down` | Jalankan / hentikan Compose |

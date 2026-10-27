# boilerplate

Template Go untuk **microservice / repository baru**: health-check endpoint,
Swagger UI, structured logging, test, CI/CD, dan Docker — semuanya disusun dengan
clean architecture berbasis interface.

## Quick start

```bash
make run          # default port 1111
PORT=9090 make run

# atau
go run ./cmd/api
```

Buka:

| URL | Keterangan |
| --- | --- |
| <http://localhost:1111/api/v1/health> | Health check (JSON) |
| <http://localhost:1111/swagger/index.html> | Swagger UI |
| <http://localhost:1111/> | Redirect ke Swagger UI |

Contoh respons:

```json
{
  "status": "ok",
  "service": "boilerplate",
  "version": "1.0.0",
  "timestamp": "2026-10-27T15:19:04Z",
  "checks": [
    { "name": "runtime", "status": "up", "detail": "uptime=7s" }
  ]
}
```

Jika salah satu dependency gagal, `status` menjadi `degraded` dan HTTP code `503`.
Detail error asli **hanya masuk log**, sedangkan response memakai
`"detail": "unavailable"` supaya informasi internal tidak bocor ke client.

## Struktur

```
.
├── cmd/api/main.go                        # composition root: wiring, server, graceful shutdown
├── internal
│   ├── constant/constant.go               # konstanta global: route, header, default config
│   ├── helper
│   │   ├── duration.go                    # konversi durasi ke milidetik
│   │   ├── env.go                         # pembacaan environment variable
│   │   └── requestid.go                   # generator request ID
│   ├── domain/health.go                   # entity, value object, invariant, port Checker
│   ├── usecase/health.go                  # application service (interface Health)
│   ├── infrastructure
│   │   ├── checker/runtime.go             # implementasi Checker
│   │   └── logger/logger.go               # logrus logger + file writer
│   └── delivery/httpapi
│       ├── health_handler.go              # handler HTTP
│       ├── router.go                      # routing, middleware, Swagger
│       ├── middleware.go                  # request ID, access log, recovery
│       └── response.go                    # error envelope
├── docs/                                  # hasil generate swag (jangan diedit manual)
├── logs/                                  # output log (app.log, di-ignore git)
├── .github/workflows/                     # CI dan release pipeline
├── Dockerfile
├── Makefile
└── go.mod
```

Arah dependensi:

```
delivery ──▶ usecase ──▶ domain ◀── infrastructure
                              ▲
                         cmd/api (wiring)
```

- `domain` tidak mengimpor layer lain.
- `usecase` bergantung pada interface `domain.Checker`.
- `delivery` bergantung pada interface `usecase.Health`.
- `infrastructure` mengimplementasikan port dari layer dalam.
- Semua konstruksi objek hanya terjadi di `cmd/api/main.go`.

## Konstanta & helper global

- `internal/constant/constant.go` menyimpan nilai yang dipakai lintas layer:
  route HTTP, nama header, default port/service/log, dan pesan error.
- `internal/helper` menyimpan fungsi murni yang tidak terikat layer:
  `Env`, `NewRequestID`, dan `DurationMS`.

Keduanya sengaja ditempatkan di `internal/` supaya tidak menjadi API publik
yang tidak sengaja dipakai proyek lain. Saat butuh nilai atau fungsi baru yang
dipakai lebih dari satu layer, tambahkan di sini agar tidak terjadi duplikasi.

## Konfigurasi

| Variabel | Default | Keterangan |
| --- | --- | --- |
| `PORT` | `1111` | Port HTTP |
| `SERVICE_NAME` | `boilerplate` | Nama service di response health dan log |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_DIR` | `logs` | Direktori log, dibuat otomatis |
| `GIN_MODE` | `release` | Mode Gin; otomatis `release` jika tidak di-set |

Version di-inject saat build:

```bash
make build VERSION=1.2.3
# atau
go build -ldflags "-X main.version=1.2.3" -o bin/app ./cmd/api
```

## Endpoint

| Method | Path | Deskripsi |
| --- | --- | --- |
| GET | `/api/v1/health` | Status kesehatan service |
| GET | `/swagger/*` | Swagger UI & OpenAPI spec |
| GET | `/` | Redirect ke Swagger UI |

Route tidak dikenal mengembalikan JSON `404`, method salah mengembalikan JSON
`405`, dan panic ditangkap recovery middleware menjadi `500`.

## Logging

Semua log aplikasi memakai **logrus** dalam format JSON, ditulis ke **stdout**
dan file `LOG_DIR/app.log`. Output Gin dan `net/http` juga diarahkan ke logrus;
tidak ada logger lain.

Setiap request menghasilkan satu access log berisi:

- `request_id` — diambil dari header `X-Request-ID`, atau dibuat otomatis
- `method`, `path`, `status`, `latency_ms`
- `client_ip`, `user_agent`

Status `4xx` dicatat sebagai `warning`, `5xx` sebagai `error`, dan panic dicatat
sebagai `error` lengkap dengan stack trace. Response juga mengembalikan header
`X-Request-ID` supaya client bisa mencocokkan request dengan log.

Contoh entry:

```json
{"level":"info","msg":"http request","request_id":"7f3a9c1d2e4b5a6f",
 "method":"GET","path":"/api/v1/health","status":200,"latency_ms":0.268,
 "client_ip":"::1","user_agent":"curl/8.7.1","time":"2026-10-27T22:38:09+07:00"}
```

Catatan: belum ada rotasi file. Untuk produksi, tambahkan rotasi (misalnya
`lumberjack`) atau serahkan ke log collector/container runtime.

## Testing & quality

```bash
make test     # semua test + race detector + coverage
make lint     # go vet
make fmt      # gofmt
make ci       # fmt-check + vet + test + build
```

Test mencakup:

- invariant domain (`NewHealth`, status agregat, error tidak ikut ter-serialisasi)
- implementasi `Checker`
- helper global (`Env`, `NewRequestID`, `DurationMS`)
- usecase (100% coverage)
- handler, router, 404/405, Swagger
- middleware: access log, request ID, panic recovery + stack trace
- integration smoke test lewat HTTP server sungguhan (`httptest.NewServer`)

## Docker

```bash
make docker VERSION=1.2.3
make docker-run

# atau manual
docker build --build-arg VERSION=1.2.3 -t boilerplate:1.2.3 .
docker run --rm -p 1111:1111 \
  -e SERVICE_NAME=my-service \
  -v boilerplate-logs:/app/logs \
  boilerplate:1.2.3
```

Image runtime memakai Alpine, berjalan sebagai user non-root, dan sudah punya
`HEALTHCHECK` ke endpoint health.

## CI/CD

`.github/workflows/ci.yml` berjalan pada push ke `main` dan pull request:

1. cek `gofmt`
2. `go vet`
3. `go test -race -cover`
4. `go build`
5. `docker build` (validasi Dockerfile)

`.github/workflows/release.yml` berjalan saat tag `v*` dibuat: build binary
linux/darwin (amd64/arm64) dengan version dari tag, lalu publish GitHub Release.

## Memakai template ini untuk repo/microservice baru

1. Buat repo dari template ini (tombol **Use this template** di GitHub, atau
   clone lalu hapus `.git`).
2. Ganti module path:

   ```bash
   OLD=github.com/sirojulanam/boilerplate
   NEW=github.com/<org>/<repo>

   go mod edit -module "$NEW"
   grep -rl "$OLD" --exclude-dir=.git --exclude-dir=logs . \
     | xargs perl -pi -e "s|\Q$OLD\E|$NEW|g"
   go mod tidy
   ```

3. Sesuaikan nama service:
   - runtime: set `SERVICE_NAME=nama-service`
   - default di kode: ubah `DefaultServiceName` di `internal/constant/constant.go`
   - binary: ubah `APP` di `Makefile` atau jalankan `make build APP=nama-service`
4. Ubah judul/deskripsi Swagger di anotasi `cmd/api/main.go`, lalu:
   ```bash
   make docs
   ```
5. Jalankan `make ci` dan `make docker` untuk memastikan semuanya hijau.
6. Di GitHub, aktifkan **Settings → Template repository** agar repo ini bisa
   dipakai langsung oleh repo baru berikutnya.

## Menambah dependency check baru

Cukup implementasikan `domain.Checker`:

```go
type Checker interface {
	Name() string
	Check(ctx context.Context) (detail string, err error)
}
```

Lalu daftarkan di `cmd/api/main.go`:

```go
healthUsecase := usecase.NewHealth(
	serviceName,
	version,
	checker.NewRuntime(startedAt),
	checker.NewDatabase(db), // contoh
)
```

Tidak ada perubahan di handler maupun router — semuanya lewat interface.

## Perintah Make

| Target | Fungsi |
| --- | --- |
| `make run` | Jalankan service |
| `make build` | Build binary ke `bin/$(APP)` |
| `make test` | Test + race + coverage |
| `make docs` | Regenerate Swagger |
| `make tidy` | `go mod tidy` |
| `make fmt` | Format kode |
| `make fmt-check` | Cek format tanpa mengubah file |
| `make lint` | `go vet` |
| `make ci` | Pipeline lokal lengkap |
| `make docker` | Build image |
| `make docker-run` | Jalankan image dengan volume log |

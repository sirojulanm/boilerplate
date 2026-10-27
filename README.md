# opencodeproject

Project Go sederhana dengan **health check endpoint** dan **Swagger UI**, disusun
memakai clean architecture berbasis interface.

## Menjalankan

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
  "service": "opencodeproject",
  "version": "1.0.0",
  "timestamp": "2026-10-27T15:19:04Z",
  "checks": [
    { "name": "runtime", "status": "up", "detail": "uptime=7s" }
  ]
}
```

Jika salah satu dependency gagal, `status` menjadi `degraded` dan HTTP code `503`.

## Struktur

```
.
├── cmd/api/main.go                        # entrypoint: wiring + HTTP server + graceful shutdown
├── internal
│   ├── domain/health.go                   # entity + port (interface Checker)
│   ├── usecase/health.go                  # business logic (interface Health)
│   ├── infrastructure
│   │   ├── checker/runtime.go             # implementasi Checker
│   │   └── logger/logger.go               # logrus logger + file writer
│   └── delivery/httpapi
│       ├── health_handler.go              # handler HTTP
│       ├── router.go                      # routing + Swagger
│       ├── middleware.go                  # access log + recovery
│       └── response.go                    # error envelope
├── docs/                                  # hasil generate swag (jangan diedit manual)
├── logs/                                  # output log (app.log, di-ignore git)
├── Makefile
└── go.mod
```

Arah dependensi: `delivery` → `usecase` → `domain` ← `infrastructure`.
Lapisan dalam tidak tahu apa pun tentang lapisan luar.

## Endpoint

| Method | Path | Deskripsi |
| --- | --- | --- |
| GET | `/api/v1/health` | Status kesehatan service |
| GET | `/swagger/*` | Swagger UI & OpenAPI spec |

## Logging

Semua log aplikasi memakai **logrus** (JSON), ditulis ke **stdout** dan file
`logs/app.log`. Tidak ada logger lain: output Gin dan `net/http` juga
diteruskan ke logrus.

Setiap request menghasilkan satu access log berisi `method`, `path`, `status`,
`latency_ms`, `client_ip`, dan `user_agent`. Status `4xx` dicatat sebagai
`warning`, `5xx` sebagai `error`, panic ditangkap recovery middleware dan
dicatat sebagai `error` tanpa menghentikan server.

Konfigurasi lewat environment:

| Variabel | Default | Keterangan |
| --- | --- | --- |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_DIR` | `logs` | direktori output, dibuat otomatis |

Contoh entry:

```json
{"level":"info","msg":"http request","method":"GET","path":"/api/v1/health",
 "status":200,"latency_ms":0.268,"client_ip":"::1","time":"2026-10-27T22:38:09+07:00"}
```

Catatan: belum ada rotasi file. Untuk produksi, tambahkan rotasi (misalnya
`lumberjack`) atau serahkan ke log collector/container runtime.

## Perintah

```bash
make docs     # regenerate docs/ dari anotasi (swag v1.16.6)
make test     # jalankan semua unit test + coverage
make build    # build binary ke bin/opencodeproject
make tidy     # go mod tidy
```

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

Tidak ada perubahan di handler maupun router — semua lewat interface.

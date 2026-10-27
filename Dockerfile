# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

ARG VERSION=1.0.0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/boilerplate ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/migrate ./cmd/migrate

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/boilerplate /app/boilerplate
COPY --from=build /out/migrate /app/migrate

RUN mkdir -p /app/logs && chown -R app:app /app

USER app

ENV PORT=1111 \
    SERVICE_NAME=boilerplate \
    LOG_DIR=/app/logs

EXPOSE 1111
VOLUME ["/app/logs"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/api/v1/health" >/dev/null || exit 1

ENTRYPOINT ["/app/boilerplate"]

# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Use native build platform so Go cross-compiles without QEMU
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETARCH
ARG TARGETOS=linux

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" \
    -o /out/auth-api ./cmd/api/main.go

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" \
    -o /out/migrate ./cmd/migrate/main.go

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/auth-api /auth-api
COPY --from=builder /out/migrate  /migrate
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/auth-api"]

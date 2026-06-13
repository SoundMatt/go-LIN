# Multi-stage build for go-LIN quickstart and lintool binaries.

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /bin/quickstart ./examples/quickstart
RUN CGO_ENABLED=0 go build -trimpath -o /bin/lintool    ./cmd/lintool

# ── quickstart image ──────────────────────────────────────────────────────────
FROM alpine:3.21 AS quickstart
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/quickstart /usr/local/bin/quickstart
LABEL org.opencontainers.image.title="go-LIN quickstart" \
      org.opencontainers.image.source="https://github.com/SoundMatt/go-LIN" \
      org.opencontainers.image.licenses="MPL-2.0"
ENTRYPOINT ["quickstart"]

# ── lintool image ─────────────────────────────────────────────────────────────
FROM alpine:3.21 AS lintool
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/lintool /usr/local/bin/lintool
LABEL org.opencontainers.image.title="go-LIN lintool" \
      org.opencontainers.image.source="https://github.com/SoundMatt/go-LIN" \
      org.opencontainers.image.licenses="MPL-2.0"
ENTRYPOINT ["lintool"]

# Multi-stage build for go-LIN images: the RELAY-conformant go-lin CLI
# (published as ghcr.io/soundmatt/go-lin per spec §13.5), plus the quickstart
# and lintool example/legacy binaries.

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /bin/go-lin     ./cmd/go-lin
RUN CGO_ENABLED=0 go build -trimpath -o /bin/quickstart ./examples/quickstart
RUN CGO_ENABLED=0 go build -trimpath -o /bin/lintool    ./cmd/lintool

# ── go-lin image (RELAY-conformant CLI, spec §13.5) ───────────────────────────
FROM alpine:3.21 AS go-lin
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/go-lin /usr/local/bin/go-lin
LABEL org.opencontainers.image.title="go-lin" \
      org.opencontainers.image.source="https://github.com/SoundMatt/go-LIN" \
      org.opencontainers.image.licenses="MPL-2.0" \
      io.relay.tool="go-lin" \
      io.relay.language="go" \
      io.relay.binary="go-lin" \
      io.relay.spec-version="1.11"
ENTRYPOINT ["go-lin"]

# ── quickstart image ──────────────────────────────────────────────────────────
FROM alpine:3.21 AS quickstart
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/quickstart /usr/local/bin/quickstart
LABEL org.opencontainers.image.title="go-LIN quickstart" \
      org.opencontainers.image.source="https://github.com/SoundMatt/go-LIN" \
      org.opencontainers.image.licenses="MPL-2.0"
ENTRYPOINT ["quickstart"]

# ── lintool image (legacy, non-RELAY-conformant example CLI) ─────────────────
FROM alpine:3.21 AS lintool
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/lintool /usr/local/bin/lintool
LABEL org.opencontainers.image.title="go-LIN lintool" \
      org.opencontainers.image.source="https://github.com/SoundMatt/go-LIN" \
      org.opencontainers.image.licenses="MPL-2.0"
ENTRYPOINT ["lintool"]

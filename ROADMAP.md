# go-LIN Roadmap

## Vision

go-LIN is a modern, Go-native LIN bus library for automotive, industrial, and embedded domains.

The project focuses on:

- A clean, stable `lin.Bus` / `lin.MasterBus` interface with swappable transports
- Pure Go — no CGo, no native dependencies beyond optional serial I/O
- Safety-oriented design with go-FuSa annotations and E2E protection
- Standards compliance: LIN 2.x, LDF parsing, PID and checksum algorithms
- Testability by default via the in-process virtual bus

---

## Guiding Principles

1. Pure Go first
2. Standards where they provide value (LIN 2.x, LDF, E2E)
3. Simplicity over completeness
4. Testability by default — virtual bus works everywhere
5. Safety as a first-class concern
6. Interface-first API — transports are always swappable
7. Optional bridges — protocol adapters carry their own dependencies; core remains zero-dependency

---

## Release History

See [CHANGELOG.md](CHANGELOG.md) for the detailed per-version changelog. Summary:

| Version | Theme |
|---|---|
| v0.1.0 | Core `lin.Bus`/`MasterBus` interfaces, virtual bus, LDF parser, master/slave nodes, E2E safety, `cmd/lintool` CLI, Docker quickstart |
| v0.2.0 | 100 atomic ASIL-B SEOOC requirements |
| v0.3.0 | RELAY spec v0.2 conformance — `Subscribe` slice signature, optional interfaces (`HealthProvider`, `MetricsProvider`, `Drainer`) |
| v0.4.0 | RELAY spec v0.3 conformance |
| v1.0.0 | RELAY spec v1.0 (stable) conformance |
| v1.1.0 | RELAY spec v1.10 conformance — §13.7 cross-language library architecture, §20 continuous conformance |
| v1.2.0 | Full ISO/IEC/DO compliance evidence pack, max coverage |
| Unreleased | RELAY spec v1.11 conformance; `cmd/go-lin` RELAY CLI docs/Docker image; diagnostic frames, sporadic frames, LDF write-direction encoding, `stats` package; see CHANGELOG |

---

## Planned / Open Work

Re-baselined against the current issue tracker (2026-07) — this table replaces
the original pre-RELAY plan, which described work later superseded by the
RELAY spec-conformance releases above.

| Issue | Theme | Status |
|---|---|---|
| [#1](https://github.com/SoundMatt/go-LIN/issues/1) | Serial/UART transport (`transport/`) — physical LIN on Linux via `/dev/ttyS*` | open — hardware-facing; needs a real-hardware or hardware-in-the-loop validation plan before landing in a safety library |
| [#3](https://github.com/SoundMatt/go-LIN/issues/3) | Sleep/wakeup frame sequences — go-to-sleep command, wakeup pulse generation/detection | open — wakeup pulse handling is a physical-transport concern, blocked on #1 |
| [#5](https://github.com/SoundMatt/go-LIN/issues/5) | Event-triggered frames — multi-slave collision resolution | open — needs a dedicated protocol-timing design pass |
| [#7](https://github.com/SoundMatt/go-LIN/issues/7) | **Bridge — CAN** (`bridge/can/`) — LIN-over-CAN gateway (works with go-CAN) | open — cross-repo dependency on go-CAN |
| [#8](https://github.com/SoundMatt/go-LIN/issues/8) | **Bridge — MQTT** (`bridge/mqtt/`) — publish/subscribe LIN frames over MQTT | open — needs an MQTT client dependency, not yet vendored |

Delivered since the original plan was written (see CHANGELOG for details):
diagnostic frames (#2), sporadic frames (#4), LDF write-direction encoding (#6),
statistics (#10), godoc examples (#11), go-FuSa/coverage (#9).

---

## Milestones

### 1 — Core Type Abstraction
- `lin.Frame` with ID (6-bit), Data (1–8 bytes), Checksum, ChecksumType
- `lin.Filter` with exact-ID and all-frames matching
- `lin.Bus` interface (Publish, Subscribe, Close)
- `lin.MasterBus` extension (SendHeader)
- `lin.ProtectID`, `lin.VerifyPID`, `lin.CalcChecksum`
- `lin.ValidateFrame`

### 2 — Virtual In-Process Bus
- Zero-dependency broadcast bus
- Simulates master/slave frame exchanges in-process
- Multiple subscribers with independent filter sets
- Drop-on-full-channel semantics
- Fuzz target for `SendHeader`

### 3 — LDF Parser
- Protocol version, language version, baud rate
- Node declarations (master + slaves)
- Signal definitions (bit width, init value, publisher/subscribers)
- Frame definitions (ID, publisher, length, signal-to-bit-offset mappings)
- Schedule table parsing (frame name + delay)
- Signal decoder: `db.Decode(id, data) map[string]uint64`
- Signal encoder (write direction): `db.Encode(id, signals) []byte`
- Fuzz target for `Parse`

### 4 — Master Node
- Schedule table execution (frame-ID + slot delay)
- `SendHeader` driving `MasterBus`
- `OnFrame` and `OnError` callbacks
- Context-cancellation support
- `Diagnostics` — LIN 2.x §4.2.3 master-request/slave-response exchange
  (`MasterRequestFrame`/`SlaveResponseFrame`)
- Sporadic frame slots — `SetSporadicGroup`/`SetPending`, priority-ordered
  candidate selection per LIN 2.x §2.3.2.4

### 5 — Slave Node
- Response registration per frame ID
- Multiple IDs per slave
- Direct pass-through to `bus.Publish`

### 6 — Safety E2E
- 10-byte protection header: DataID, SourceID, SequenceCounter, CRC-16/CCITT-FALSE
- `Protector` and `Receiver` wrappers
- Detects CRC mismatch, sequence gaps, and short headers
- Fuzz target for `ProtectUnwrap`

### 7 — CLI
- `cmd/go-lin` — the RELAY-conformant CLI: `version`/`capabilities`/`status`
  (spec §11.1), `convert` (spec §11.2 interop driver), `send`/`subscribe`
  (spec §11.2 optional crossbar-spoke commands, including the
  `--id`/`--data`/`--count` flag forms), plus `dump`/`pid`/`cs`
- `cmd/lintool` — legacy pre-RELAY example CLI (`send`, `dump`, `pid`, `cs`),
  kept for backward compatibility

### 8 — Docker
- Multi-stage Dockerfile (builder → go-lin, quickstart, lintool)
- `go-lin` image carries the spec §13.5 `io.relay.*` labels, published as
  `ghcr.io/soundmatt/go-lin`
- docker-compose.yml for zero-config demo
- Multi-arch images (linux/amd64, linux/arm64) published to GHCR

### 9 — Observability
- `stats.Collector` — frames/sec, per-frame-ID counters, estimated bus load
  percentage

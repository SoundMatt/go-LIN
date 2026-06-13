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

## Release Plan

| Version | Theme | Status |
|---|---|---|
| v0.1.0 | Core `lin.Bus`/`MasterBus` interfaces, virtual bus, LDF parser, master/slave nodes, E2E safety, CLI, Docker quickstart | **next** |
| v0.2.0 | Serial/UART transport (`transport/`) — physical LIN on Linux via `/dev/ttyS*` | planned |
| v0.3.0 | Diagnostic frames — master request (0x3C) and slave response (0x3D) handling | planned |
| v0.4.0 | Sleep/wakeup frame sequences — go-to-sleep command, wakeup pulse | planned |
| v0.5.0 | Sporadic frames — master selects which frame to transmit based on flags | planned |
| v0.6.0 | Event-triggered frames — multi-slave collision resolution | planned |
| v0.7.0 | LDF signal encoding (write direction) and value table support | planned |
| v0.8.0 | go-FuSa v0.30.0 → latest; coverage 80% across all packages | planned |
| v0.9.0 | Statistics — bus load, frame error counters, per-ID metrics | planned |
| v1.0.0 | API stability, full serial transport, documentation complete | planned |
| v1.1.0 | **Bridge — CAN** (`bridge/can/`) — LIN-over-CAN gateway (works with go-CAN) | planned |
| v1.2.0 | **Bridge — MQTT** (`bridge/mqtt/`) — publish/subscribe LIN frames over MQTT | planned |
| v1.3.0 | **Bridge — DDS** (`bridge/dds/`) — LIN frame distribution over DDS topics | planned |
| v1.4.0 | **Bridge — SOME/IP** (`bridge/someip/`) — LIN frames as SOME/IP service events | planned |
| v1.5.0 | **Bridge — gRPC** (`bridge/grpc/`) — stream LIN frames over gRPC | planned |

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
- Fuzz target for `Parse`

### 4 — Master Node
- Schedule table execution (frame-ID + slot delay)
- `SendHeader` driving `MasterBus`
- `OnFrame` and `OnError` callbacks
- Context-cancellation support

### 5 — Slave Node
- Response registration per frame ID
- Multiple IDs per slave
- Direct pass-through to `bus.Publish`

### 6 — Safety E2E
- 10-byte protection header: DataID, SourceID, SequenceCounter, CRC-16/CCITT-FALSE
- `Protector` and `Receiver` wrappers
- Detects CRC mismatch, sequence gaps, and short headers
- Fuzz target for `ProtectUnwrap`

### 7 — CLI (lintool)
- `send <id> <hex-data>` — publish response and trigger one frame exchange
- `dump` — print all received frames to stdout
- `pid <id>` — compute and display Protected Identifier
- `cs <id> <hex-data>` — compute and display enhanced checksum

### 8 — Docker Quickstart
- Multi-stage Dockerfile (builder → quickstart, builder → lintool)
- docker-compose.yml for zero-config demo
- Multi-arch images (linux/amd64, linux/arm64) published to GHCR

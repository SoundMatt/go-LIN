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
| v1.3.0 | RELAY spec v1.11 conformance; `cmd/go-lin` RELAY CLI docs/Docker image; diagnostic frames, sporadic frames, LDF write-direction encoding, `stats` package; see CHANGELOG |
| v1.4.0 | Audit fix pass: diagnostic-frame (0x3C/0x3D) classic-checksum bug, oversize-payload rejection, HARA ASIL recomputation, deprecated-alias cleanup, doc/traceability corrections; see CHANGELOG |
| v1.5.0 | RELAY dependency bump v1.11.0 → `github.com/SoundMatt/RELAY/v2` v2.0.4 (spec v2.0 conformance); no LIN-relevant behavioral changes, verified against the real CLI; see CHANGELOG |

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

---

## Future — LIN Bus Simulator

Not scheduled against a milestone or issue yet. Captured here as a scoped plan
because LIN has a testing gap none of the other x-Net members have: go-CAN can
validate against Linux's real `vcan` kernel interface, and go-DDS/cpp-DDS/
rust-DDS can validate against CycloneDDS as a third-party peer (or, per
rust-DDS's current interop work, a second process of themselves). LIN has
neither a kernel-native virtual bus nor a widely-used third-party stack in
this ecosystem to test against. A real simulator is the only way to close
that gap.

### What already exists (read from the current codebase, not assumed)

- `virtual.Bus` (`virtual/bus.go`) is **not** a slave-node simulator. It is a
  single shared `map[uint8]responseEntry` guarded by one `sync.RWMutex`.
  `Publish`/`PublishClassic` write an entry; `SendHeader` reads the entry back,
  computes `lin.ProtectID` and `lin.CalcChecksum` *correctly* every time, and
  broadcasts the resulting `lin.Frame`. There is no path by which a registered
  response can come back with a wrong checksum, a wrong PID, or no response at
  all (short of not registering one, which only exercises `ErrNoResponse`,
  not "slave went silent mid-session").
- `slave.Node` (`slave/slave.go`) is a thin wrapper: `SetResponse` just calls
  `bus.Publish` and tracks `RegisteredIDs()`. It has no goroutine, no
  internal state machine, and no independent timing — it is a registration
  call, not a simulated ECU. Two slaves registering the same ID silently
  overwrite each other in the map; nothing detects or reports the conflict.
- `master.Node` (`master/master.go`), by contrast, is already a real
  schedule-table engine: `Run(ctx)` walks `[]lin.ScheduleEntry{ID, DelayMs}`
  in a loop, calls `bus.SendHeader` per slot, sleeps `DelayMs`, and drives
  `OnFrame`/`OnError`. It already supports LIN 2.x §2.3.2.4 sporadic frame
  slots via `SetSporadicGroup`/`SetPending`. This does not need to be
  reinvented for a simulator — it needs a transport underneath it that behaves
  like a bus with independently-acting slaves instead of a static lookup
  table.
- `ldf.Database` already parses node declarations, frame-to-publisher
  assignments, and schedule tables (`db.Schedule(name)`) from real `.ldf`
  files. A simulator can and should be drivable directly from an LDF, not
  just from hand-built `ScheduleEntry` slices.
- Everything above lives in one Go process. There is no serialization format
  for `lin.Frame` beyond the JSON tags already on the struct, and no IPC of
  any kind — `virtual.Bus` cannot be shared across two `go-LIN` binaries, let
  alone between a go-LIN process and a cpp-LIN or rust-LIN process.

### What a real simulator needs that doesn't exist today

1. **Independent slave behavior**, not a shared map. Each simulated slave
   needs its own goroutine (or at least its own decision point) so that
   "does this slave answer, and with what" is a per-slave, per-cycle
   decision — not a value looked up out of one struct shared with every
   other slave on the bus.
2. **Configurable response behavior, including fault injection.** Concretely:
   respond with a correct frame, respond with a corrupted checksum, respond
   with a mismatched PID, don't respond at all (timeout), or respond after an
   injected delay. This is what actually exercises `master.Node.OnError`,
   `MasterBus.SendHeader`'s `ErrNoResponse` path, and (once E2E is layered on
   top) `safety.Receiver`'s CRC/sequence-gap detection under conditions a
   correct-by-construction bus like `virtual.Bus` can never produce.
3. **Multi-slave conflict detection.** Two simulated slaves claiming the same
   frame ID — or a simulated slave answering an ID the LDF assigns to a
   different publisher — should be observable/reportable by the simulator,
   not silently resolved by map overwrite.
4. **A transport decision**: does the simulator stay in-process (extend
   `virtual`-style semantics, just with real goroutines per slave and a
   fault-injection layer), or does it also get a cross-process transport
   (Unix domain socket, or a shared-memory ring buffer for lower latency) so
   two separate OS processes can be the master and the slaves? The two are
   not the same amount of work, and the phasing below treats them as
   separate deliverables.

### Proposed shape

A new top-level `simulator/` package, additive alongside `virtual` rather
than a replacement for it — `virtual` keeps its documented job (zero-dependency,
deterministic, single-struct mock used by `mock.New()` and the quickstart) per
Guiding Principle 4. `simulator` targets a different job: a bus with
independently-behaving, individually-faultable slave nodes.

Sketch, in this repo's idiom (interfaces + goroutines/channels, matching
`virtual.Bus`'s use of `sync.RWMutex` + `atomic` counters and `lin.Err*`
sentinel wrapping):

```go
// SlaveBehavior decides how a simulated slave answers a header for id.
// ok=false means the slave stays silent this cycle (unpowered, busy,
// deliberately non-responsive) — distinct from "no response ever
// registered," which virtual.Bus already models via ErrNoResponse.
type SlaveBehavior interface {
    Respond(id uint8) (data []byte, ok bool)
}

// FaultBehavior wraps a SlaveBehavior to corrupt its output, mirroring the
// go-DDS record.FaultPublisher wrapper pattern (a decorator around a
// well-behaved implementation, not a fork of it).
type FaultBehavior struct {
    Inner         SlaveBehavior
    WrongChecksum bool
    WrongPID      bool
    Silent        bool          // overrides Inner entirely: never answer
    ResponseDelay time.Duration // injected latency before the header is answered
}

// Slave runs one simulated slave node in its own goroutine, reacting to
// header-request events from a Bus.
type Slave struct {
    ID       uint8   // node address, for logging/conflict reports
    Frames   []uint8 // frame IDs this slave claims
    Behavior SlaveBehavior
}

// Bus is the simulator transport. Unlike virtual.Bus, header requests are
// *events* that registered slaves race to answer, not a map lookup — this
// is what makes independent per-slave goroutines and fault injection
// possible. Bus still implements lin.MasterBus, so master.Node.Run works
// against it unmodified.
type Bus struct { /* ... */ }

func (b *Bus) AddSlave(s *Slave) error
func (b *Bus) Conflicts() []Conflict // frame IDs claimed by >1 slave
```

Cross-process reach, if pursued (see Phase 4 below), extends the same
`lin.MasterBus`/`lin.Bus` interfaces over a Unix domain socket — e.g.
`simulator/ipc.Dial(path)` / `simulator/ipc.Listen(path)` returning something
satisfying `lin.MasterBus` on the master side and `lin.Bus` on each slave-side
connection, framing `lin.Frame` (already JSON-tagged) plus a header-request
event type over the socket. No interface changes are required anywhere else
in the repo — this is exactly what Guiding Principle 6 ("interface-first API
— transports are always swappable") is for.

### Phasing

- **Phase 1 — minimal useful first cut (in-process, well-behaved slaves).**
  `simulator.Bus` + `simulator.Slave` with event-driven (not map-driven)
  header handling, each slave in its own goroutine, default `SlaveBehavior`
  that answers correctly. Reuse `master.Node.Run` unmodified against
  `simulator.Bus`. This alone is already more realistic than `virtual.Bus`
  for schedule-table testing: N independently-scheduled slaves, "slave
  present but silent this cycle" as a real per-cycle decision, and multi-slave
  conflict detection. Optionally drivable straight from a parsed
  `ldf.Database` (nodes → `Slave`s, `db.Schedule(name)` → `master.Node.SetSchedule`).
- **Phase 2 — fault injection.** `FaultBehavior` wrapper: wrong checksum,
  wrong PID, dropped response, injected response delay — deterministic or
  probabilistic. This is what actually lets `master.Node.OnError`,
  `ErrNoResponse`, and `safety.Receiver`'s CRC/sequence-gap detection be
  exercised against realistic bus faults instead of only hand-constructed
  byte slices in unit tests.
- **Phase 3 — multi-slave scheduling conflicts.** `Bus.Conflicts()` and
  LDF cross-validation: flag frame IDs claimed by more than one simulated
  slave, and flag a simulated slave answering an ID the LDF assigns to a
  different node's publisher. This is a correctness-of-the-simulation
  concern (catching *test setup* bugs), not fault injection of the bus
  itself.
- **Phase 4 — cross-process transport (stretch goal).** `simulator/ipc` over
  a Unix domain socket (shared memory only if latency numbers from the
  socket prove insufficient — no evidence of that need yet). This is what
  would let two separate `go-LIN` processes run master and slaves as real
  OS processes instead of goroutines in one binary, and — because the wire
  framing is just `lin.Frame` plus a header-request event — is a plausible
  basis for real go-LIN/cpp-LIN/rust-LIN interop testing, the same role
  `vcan`+`can-utils` plays for go-CAN and CycloneDDS (or rust-DDS's
  two-process self-interop harness) plays for the DDS family. This phase is
  explicitly **not a prerequisite** for the simulator's core value — Phases
  1–3 deliver a fully useful fault-injecting dev/test simulator entirely
  in-process, with zero IPC. Phase 4 is only worth doing once Phases 1–3 are
  proven useful, and only if cross-implementation LIN interop testing
  becomes an active goal rather than a hypothetical one.

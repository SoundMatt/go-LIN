# go-LIN

A Go library for [LIN bus](https://en.wikipedia.org/wiki/LIN_bus) (Local Interconnect Network) communication.
Targets automotive, industrial, and embedded domains where LIN 2.x is used for low-bandwidth subsystems.

The `lin.Bus` interface is stable. Implementations are swappable without changing application code.

[![CI](https://github.com/SoundMatt/go-LIN/actions/workflows/ci.yml/badge.svg)](https://github.com/SoundMatt/go-LIN/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SoundMatt/go-LIN.svg)](https://pkg.go.dev/github.com/SoundMatt/go-LIN)

## Packages

| Package | Description | Requires |
|---|---|---|
| `lin` | Core `Bus` / `MasterBus` interface, `Frame`, `Filter`, `ScheduleEntry`, PID and checksum functions. | Nothing |
| `virtual` | In-process virtual LIN bus. Zero dependencies. Default for development and testing. | Nothing |
| `ldf` | LIN Description File (LDF) parser — nodes, signals, frames, schedule tables. | Nothing |
| `master` | LIN master node — schedule execution and header transmission. | Nothing |
| `slave` | LIN slave node — response registration and frame subscription. | Nothing |
| `safety` | E2E protection header — DataID, SourceID, SequenceCounter, CRC-16. | Nothing |
| `stats` | Observability: per-second frame rate, per-ID counters, estimated bus load. | Nothing |
| `cmd/go-lin` | RELAY-conformant CLI: `version`, `capabilities`, `status`, `convert`, `send`, `subscribe`, plus `dump`/`pid`/`cs`. Built and gated by CI (`relay conform --strict`, `relay interop`). | Nothing |
| `cmd/lintool` | Legacy example CLI (pre-RELAY): `send`, `dump`, `pid`, `cs` subcommands. Not RELAY-conformant — use `cmd/go-lin` instead. | Nothing |

## Install

```bash
go get github.com/SoundMatt/go-LIN
```

## Quick start

```go
import (
    lin "github.com/SoundMatt/go-LIN"
    "github.com/SoundMatt/go-LIN/virtual"
    "github.com/SoundMatt/go-LIN/master"
    "github.com/SoundMatt/go-LIN/slave"
)

bus, _ := virtual.New()
defer bus.Close()

// slave registers a response
slaveNode := slave.New(bus)
slaveNode.SetResponse(0x10, []byte{0x42, 0x00, 0x00, 0x00})

// master drives the schedule
m := master.New(bus)
m.SetSchedule([]lin.ScheduleEntry{{ID: 0x10, DelayMs: 10}})
m.OnFrame(func(f lin.Frame) {
    fmt.Printf("%02X#%X\n", f.ID, f.Data) // 10#42000000
})

ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()
m.Run(ctx)
```

## Docker quickstart

```bash
docker compose -f docker/docker-compose.yml up --build
```

Runs a single container with a slave goroutine publishing synthetic window-position frames
and a master goroutine driving the schedule and printing each received frame.

The RELAY-conformant CLI is also published as an image:

```bash
docker run --rm ghcr.io/soundmatt/go-lin version
```

## LDF parser

```go
import "github.com/SoundMatt/go-LIN/ldf"

db, _ := ldf.Parse(strings.NewReader(ldfContent))
frame := db.Frame(0x10)           // frame descriptor
sig := db.Signal("EngineSpeed")   // signal descriptor
vals := db.Decode(0x10, rawData)  // map[string]uint64 of raw signal values
raw := db.Encode(0x10, vals)      // write direction: pack signal values into a frame payload
sched := db.Schedule("NormalSchedule")
```

## Diagnostic frames

```go
import lin "github.com/SoundMatt/go-LIN"

// master.Node.Diagnostics drives the LIN 2.x §4.2.3 master-request (0x3C) /
// slave-response (0x3D) exchange.
req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2, Data: []byte{0x01, 0x02}}
resp, err := masterNode.Diagnostics(ctx, req)
```

## Sporadic frames

```go
// master.Node.SetSporadicGroup declares a schedule slot as sporadic
// (LIN 2.x §2.3.2.4): on each pass, the master transmits the header of the
// highest-priority candidate the application has flagged as pending, or
// skips the slot entirely if nothing is pending.
m.SetSporadicGroup(0x30, []uint8{0x11, 0x12}) // 0x11 has priority over 0x12
m.SetSchedule([]lin.ScheduleEntry{{ID: 0x30, DelayMs: 10}})
m.SetPending(0x11) // application marks 0x11's data as changed
```

## Statistics

```go
import "github.com/SoundMatt/go-LIN/stats"

c := stats.New(19.2) // bus speed in kbit/s, for BusLoad
ch, _ := bus.Subscribe(nil)
go c.Watch(ch)
// ... later:
c.FrameRate() // frames/sec
c.BusLoad()   // estimated % of bus bandwidth used
c.PerID()     // map[uint8]uint64 of per-frame-ID counts
```

## Protected Identifier (PID)

```go
pid := lin.ProtectID(0x10)         // 0x50 — adds parity bits P0, P1
id, err := lin.VerifyPID(pid)      // verify parity on receive
```

## Checksum

```go
pid := lin.ProtectID(0x10)
cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum) // LIN 2.x
cs  = lin.CalcChecksum(0,   data, lin.ClassicChecksum)  // LIN 1.x
```

## Safety E2E protection

```go
import "github.com/SoundMatt/go-LIN/safety"

cfg := safety.Config{DataID: 0x0001, SourceID: 0x0010}
p := safety.NewProtector(cfg)
r := safety.NewReceiver(cfg)

protected := p.Protect(payload)   // prepends 10-byte E2E header
original, err := r.Unwrap(protected) // checks CRC, sequence counter
```

The 10-byte E2E header exceeds the standard LIN payload (1–8 bytes). Use with diagnostic
frames (0x3C/0x3D) or a multi-slot transport layer.

## RELAY CLI

`cmd/go-lin` is the RELAY-conformant CLI: the binary CI builds and gates with
`relay conform --strict` and `relay interop` (spec §11.1/§11.2), and the one
published as `ghcr.io/soundmatt/go-lin` (spec §13.5).

```bash
# RELAY mandatory commands (spec §11.1)
go run ./cmd/go-lin version                    # tool + spec version, JSON or text
go run ./cmd/go-lin capabilities                # capabilities document (JSON)
go run ./cmd/go-lin status                      # self-assessed health

# RELAY interop driver (spec §11.2)
go run ./cmd/go-lin convert --protocol LIN      # lin.Frame JSON (stdin) -> relay.Message JSON (stdout)

# Crossbar spokes (spec §11.2 optional commands)
go run ./cmd/go-lin send --id 0x10 --data 01020304   # publish response + trigger exchange
go run ./cmd/go-lin send --format json               # NDJSON relay.Message sink (stdin)
go run ./cmd/go-lin subscribe --format json --count 5 # NDJSON relay.Message source (stdout), exit after 5

# Convenience / debugging
go run ./cmd/go-lin dump                        # subscribe to all frames until SIGINT
go run ./cmd/go-lin pid  0x10                    # compute PID
go run ./cmd/go-lin cs   0x10 01020304           # compute enhanced checksum
```

`cmd/lintool` is a legacy pre-RELAY example CLI (`send`, `dump`, `pid`, `cs`
only — no `version`/`capabilities`/`status`/`convert`) kept for backward
compatibility. New integrations should use `cmd/go-lin`.

## Safety & compliance

go-LIN is developed as a Safety Element Out Of Context (SEOOC) targeting
**ASIL-B / SIL 2**, with a complete, continuously-verified evidence pack. Every
CI run gates on the full go-FuSa lifecycle (`check`, 100% requirement
traceability **and** test coverage, `cyber`, `vuln`, `qualify`), RELAY
conformance (`relay conform --strict`, `relay interop`), a per-package coverage
floor, and **zero-GAP compliance gap reports** across seven standards.

| Standard | Report | Evidence |
|---|---|---|
| ISO 26262 (Part 6) | `gofusa iso26262` | [HARA.md](HARA.md), `.fusa-hara.json`, [SEOOC.md](SEOOC.md), `fmea.*` |
| IEC 61508 (Parts 1-3) | `gofusa iec61508` | [SAFETY_PLAN.md](SAFETY_PLAN.md), [SVP.md](SVP.md), `coverage-report.json` |
| ISO/SAE 21434 | `gofusa iso21434` | [tara.md](tara.md), [SECURITY.md](SECURITY.md), `vuln.json` |
| IEC 62443-4-2 | `gofusa iec62443` | `.fusa-iec62443.json`, [INCIDENT-RESPONSE.md](INCIDENT-RESPONSE.md) |
| DO-178C (Annex A) | `gofusa do178` | [SVP.md](SVP.md), [SCMP.md](SCMP.md), [SQAP.md](SQAP.md), `sas.md` |
| UN R.155 (CSMS) | `gofusa unece` | [SECURITY.md](SECURITY.md), `tara.json` |
| SLSA v1.0 | `gofusa slsa` | `provenance.json`, `sbom.json` |

Start with the **[Safety Manual](SAFETY_MANUAL.md)** — it is the integration-facing
document covering safety goals, the safety mechanisms go-LIN provides, the
conditions of use the integrator must satisfy, and the error-handling contract.
The architecture trust boundary is in `boundary.mermaid`; the assembled argument
is in [safety-case.md](safety-case.md). 112 atomic requirements
(`.fusa-reqs.json`) are each traced to code and to a test.

## Philosophy

- **Interface-first** — one stable `lin.Bus` interface; transports are swappable.
- **Safety-oriented** — go-FuSa annotations throughout; E2E protection built-in.
- **Testable by default** — the virtual bus needs no OS or hardware support.
- **Extensible** — bridge packages to CAN, DDS, MQTT, SOME/IP belong here or in consuming projects.

## License

Mozilla Public License v2.0. Copyright (c) 2026 Matt Jones.

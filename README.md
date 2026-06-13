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
| `cmd/lintool` | CLI tool: `send`, `dump`, `pid`, `cs` subcommands. | Nothing |

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

## LDF parser

```go
import "github.com/SoundMatt/go-LIN/ldf"

db, _ := ldf.Parse(strings.NewReader(ldfContent))
frame := db.Frame(0x10)           // frame descriptor
sig := db.Signal("EngineSpeed")   // signal descriptor
vals := db.Decode(0x10, rawData)  // map[string]uint64 of raw signal values
sched := db.Schedule("NormalSchedule")
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

## CLI tool

```bash
go run ./cmd/lintool pid  0x10          # compute PID
go run ./cmd/lintool cs   0x10 01020304 # compute enhanced checksum
go run ./cmd/lintool send 0x10 01020304 # publish response + trigger exchange
go run ./cmd/lintool dump               # subscribe to all frames
```

## Philosophy

- **Interface-first** — one stable `lin.Bus` interface; transports are swappable.
- **Safety-oriented** — go-FuSa annotations throughout; E2E protection built-in.
- **Testable by default** — the virtual bus needs no OS or hardware support.
- **Extensible** — bridge packages to CAN, DDS, MQTT, SOME/IP belong here or in consuming projects.

## License

Mozilla Public License v2.0. Copyright (c) 2026 Matt Jones.

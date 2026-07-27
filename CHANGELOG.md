# Changelog

All notable changes to go-LIN are documented here. Versions correspond to
git tags; see `gh release list` / `git tag --sort=-creatordate` for the
canonical list. Dates are release dates (UTC-7, matching tag creation).

## [Unreleased]

- fix: `virtual.Bus.CloseWithDrain` now returns `lin.ErrTimeout` (and counts
  undelivered frames into `DropCount`) when the context expires before
  draining completes, instead of silently returning `nil` (#37)
- fix: `lin.FromMessage` now wraps `lin.ErrInvalidFrame` for an out-of-range
  or unparseable message ID, so `errors.Is(err, lin.ErrInvalidFrame)` works (#38)
- fix: `master.Node.SetSchedule` now accepts an empty schedule, matching the
  documented `lin.MasterBus.SetSchedule` contract; `master.Node.Run` still
  fails fast on an empty schedule (#43)
- feat: `cmd/go-lin`'s optional `send`/`subscribe` commands now implement the
  spec §11.2 flag signatures — `send --id <uint> --data <hex>` and
  `subscribe --format json --count N` (#41)
- feat: `ldf` package gains write-direction signal encoding, `db.Encode(id,
  signals) ([]byte, error)` (#6)
- feat: new `stats` package — a `Collector` tracking frames/sec, per-ID
  counters, and bus load percentage for observability (#10)
- feat: `lin` package gains typed diagnostic-frame helpers,
  `MasterRequestFrame`/`SlaveResponseFrame`, and a `master.Node.Diagnostics`
  handler for the master request (0x3C) / slave response (0x3D) exchange (#2)
- feat: `master.Node` gains sporadic-frame slot support — a schedule slot can
  now select from a group of candidate frame IDs based on pending-update
  flags set by the application (#4)
- docs: add runnable `Example*` functions for the core `lin`, `virtual`, and
  `master` API surface (#11)
- docs: README documents `cmd/go-lin` (the RELAY-conformant CLI) as the
  primary CLI, with `cmd/lintool` clearly marked legacy (#40)
- docker: add a `go-lin` image target/stage with the spec §13.5 `io.relay.*`
  labels, published as `ghcr.io/soundmatt/go-lin` (#39)
- chore: bump `github.com/SoundMatt/RELAY` to v1.11.0 (go.mod, CI) — closes
  the `relay versions` ALIGNED=false drift (#42)
- chore: bump the pinned `go-FuSa` CLI from v0.30.0 to v0.33.3 in CI and the
  release-artifact-regeneration workflow (#9)
- docs: replace the stale pre-RELAY roadmap with an up-to-date release
  history and re-baseline "planned" work against the current issue tracker;
  add this CHANGELOG (#44)

## [1.2.0] — 2026-06-19

Full ISO/IEC/DO compliance evidence pack + max coverage (#35).

## [1.1.0] — 2026-06-19

Adopt RELAY spec v1.10: §13.7 cross-language library architecture convention,
§20 continuous conformance (#34).

## [1.0.0] — 2026-06-17

Adopt RELAY spec v1.0 (stable) conformance (#27, #28).

## [0.4.0] — 2026-06-17

Adopt RELAY spec v0.3 conformance (#25, #26); CI fix making short fuzz runs
iteration-based to stop flaky failures (#24).

## [0.3.0] — 2026-06-16

RELAY spec v0.2 conformance: `Subscribe` slice signature and LIN-prefixed
constants (#16, #20); full RELAY v0.2 conformance (#17, #21); optional
interfaces — `HealthProvider`, `MetricsProvider`, `Drainer` (#18, #23).

## [0.2.0] — 2026-06-13

Expand to 100 atomic ASIL-B SEOOC requirements (#15).

## [0.1.0] — 2026-06-13

Initial release: core `lin.Bus`/`MasterBus` interfaces, virtual bus, LDF
parser, master/slave nodes, E2E safety, `cmd/lintool` CLI, Docker quickstart.

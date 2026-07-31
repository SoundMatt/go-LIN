# Changelog

All notable changes to go-LIN are documented here. Versions correspond to
git tags; see `gh release list` / `git tag --sort=-creatordate` for the
canonical list. Dates are release dates (UTC-7, matching tag creation).

## [Unreleased]

- fix: `ldf.parseFrameHeader` now rejects (rather than silently corrupting)
  a frame whose ID is outside 0x00–0x3F or whose declared length is
  outside 0–8 bytes — previously a negative length (e.g. from
  `f: 0x10, MASTER, -4;`) reached `DB.Encode`'s `make([]byte, f.Length)`
  and panicked (`runtime error: makeslice: len out of range`, CWE-789 DoS
  on untrusted LDF input), and an out-of-range ID (e.g. `300`) was
  silently truncated via a bare `uint8()` cast, corrupting whatever frame
  already lived at the truncated ID. A frame rejected this way is now
  properly skipped rather than swallowing every subsequent frame in the
  same `Frames` section (the previous single-`continue` mistook the
  rejected frame's own closing brace for the section's closing brace).
  Signal-ref bit offsets with a negative value (also previously discarded
  the parse error) are now rejected the same way rather than relying on
  incidental Go shift/comparison semantics to avoid a panic (#76)
- fix(safety): `safety.Receiver.Unwrap` now compares the wire-transmitted
  `DataID`/`SourceID` against the receiver's configured `Config` and
  returns a new `ErrIDMismatch` on mismatch — previously the CRC check
  alone was (incorrectly, per the code's own now-corrected comment)
  treated as sufficient masquerade protection, so a frame protected under
  a different `DataID`/`SourceID` than the receiver's was accepted
  without error (#76)
- fix(virtual): `Bus.Publish`/`Bus.PublishClassic` now reject a non-nil,
  zero-length payload the same way they already reject an over-length
  one, so the virtual bus can never broadcast a 0-data-byte frame that
  `lin.ValidateFrame` itself would consider malformed (LIN Specification
  Package 2.2A: the data field carries 1–8 bytes); `PublishClassic` also
  gained the `LINMaxDataLen` over-length guard `Publish` already had (#76)
- chore(ci): pinned all third-party and first-party GitHub Actions in
  `.github/workflows/` to immutable commit SHAs (with a `# vX` comment for
  readability) instead of mutable version tags, matching the repo's own
  SLSA/supply-chain evidence posture (#76)

## [1.5.0] — 2026-07-30

- chore: bump `github.com/SoundMatt/RELAY` v1.11.0 → `github.com/SoundMatt/RELAY/v2`
  v2.0.4, tracking RELAY spec v1.12 ("c" as a valid CLI `language` value,
  N/A — go-lin already reports `"go"`), v1.13 (deep-audit CLI/tooling fix
  pass — RELAY-side `relay conform`/`relay interop`/`relay crossbar`
  bugfixes, none of which go-LIN's CI invocation was exposed to: `--strict`
  is already passed before the binary path in `.github/workflows/ci.yml`,
  not after), v1.14 (§13.7.2 module-name registry expansion — RCP/DDS-only,
  N/A to LIN), and v2.0 (MAJOR — RCP canonical-type replacement, RCP-only,
  N/A to LIN; the only change relevant to every RELAY consumer was v2.0.4's
  own fix, the `go.mod` `/v2` module-path suffix required for the v2 tag to
  be `go install`/`go get`-able at all). Updated all six `relay
  "github.com/SoundMatt/RELAY"` imports (`adapt.go`, `adapt_test.go`,
  `lin.go`, `relay_vectors_test.go`, `cmd/go-lin/main.go`,
  `cmd/go-lin/main_test.go`) to `github.com/SoundMatt/RELAY/v2`, and both
  `.github/workflows/ci.yml` `relay` CLI install pins to
  `go install github.com/SoundMatt/RELAY/v2/cmd/relay@v2.0.4`. Verified
  against the real v2.0.4 CLI built from source: `relay conform --strict`
  and `relay interop --strict --protocol LIN` both PASS with no new
  findings; `lin.SpecVersion` (aliased from `relay.SpecVersion`) now
  reports `"2.0"`. The three checked-in `testdata/relay-vectors/` LIN
  golden vectors are byte-identical to RELAY v2.0.4's published
  `spec/vectors/` — no vector drift. `go build ./...`, `go vet ./...`,
  `go test ./...`, `go test ./... -race`, `gofusa check`, `gofusa trace
  -req-coverage 100`, `gofusa trace -sec-tested 100`, `gofusa cyber`, and
  `gofusa vuln` all pass with no new findings (#74)

## [1.4.0] — 2026-07-30

- fix: `master.Node.Diagnostics` now transmits LIN diagnostic frames
  (0x3C/0x3D) with the classic checksum required by ISO 17987 / LIN 2.x
  §4.2.3, instead of the enhanced checksum every other frame uses; added
  a checksum-type-aware `Bus.PublishFrame` and routed `Diagnostics`
  through it (#72)
- fix: `linNode.Send`/`virtual.Bus.Publish` now return
  `lin.ErrPayloadTooLarge` for a payload exceeding `LINMaxDataLen`
  instead of silently accepting it (#72)
- fix: `linNode.Send` now returns `lin.ErrInvalidFrame` (previously
  `ErrNotConnected`) for a malformed/out-of-range message ID, matching
  `FromMessage` (#72)
- fix(safety): recomputed every HARA hazard's `asil` in `.fusa-hara.json`
  from its severity/exposure/controllability fields — all six were one
  band too high (#72)
- test: `TestDiagnostics_requestResponseRoundTrip` now asserts checksum
  type/value on both the 0x3C request and 0x3D response frames (#72)
- chore: replaced internal uses of the deprecated `lin.MaxID`/
  `lin.MaxDataLen` aliases with `LINMaxID`/`LINMaxDataLen` (#72)
- docs: `HARA.md` SG-02 now traces to the checksum requirements
  (`REQ-LIN-008/009/010`) instead of the PID-parity requirement (#72)
- docs: `sas.md` no longer reports `SVP.md`/`SCMP.md`/`SQAP.md` as
  missing — all three exist at the repo root (#72)
- docs: removed a `CONTRIBUTING.md` project-structure row for a
  nonexistent `transport/` directory (#72)

## [1.3.0] — 2026-07-27

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

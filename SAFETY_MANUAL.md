# go-LIN Safety Manual

**Document ID:** SM-001
**Component:** `github.com/SoundMatt/go-LIN`
**Classification:** Safety Element Out Of Context (SEOOC), ASIL-B / SIL 2
**Standards:** ISO 26262:2018 · IEC 61508:2010 · ISO/SAE 21434:2021 · IEC 62443-4-2

This Safety Manual is the integration-facing safety document for go-LIN. It
tells a system integrator what go-LIN guarantees, what it assumes of the
surrounding system, and how to use it so that the assumptions of the safety
case hold. It consolidates and points into the detailed work products:

| Work product | File |
|---|---|
| Safety plan | [SAFETY_PLAN.md](SAFETY_PLAN.md) |
| Hazard & risk analysis (HARA) | [HARA.md](HARA.md), `.fusa-hara.json` |
| Threat analysis (TARA) | [tara.md](tara.md), `tara.json` |
| SEOOC assumptions of use | [SEOOC.md](SEOOC.md) |
| Requirements registry | `.fusa-reqs.json` |
| dFMEA | `fmea.csv`, `fmea.json` |
| Safety case | [safety-case.md](safety-case.md) |
| Security policy / incident response | [SECURITY.md](SECURITY.md), [INCIDENT-RESPONSE.md](INCIDENT-RESPONSE.md) |
| Architecture / trust boundaries | `boundary.mermaid` |

## 1. Scope and intended use

go-LIN is a software library implementing the LIN 2.x protocol (frame
construction/validation, PID parity, checksums), an in-process virtual bus,
an LDF parser, master/slave node logic, and an end-to-end (E2E) safety
protection layer. It is intended for use as a building block within an
automotive or industrial ECU. It is **not** a complete safety function on its
own; the integrator is responsible for the surrounding safety architecture.

The assumed usage context (protocol version, topology, baud rate, payload
size, application domain) is defined in [SEOOC.md §2](SEOOC.md).

## 2. Safety goals

go-LIN contributes to five safety goals derived from the HARA (SG-01 … SG-05).
Full S/E/C classification and hazard linkage are in [HARA.md](HARA.md) and
`.fusa-hara.json`.

| Goal | Statement | ASIL |
|---|---|---|
| SG-01 | Correctly identify frame IDs via PID parity computation and verification. | ASIL-B |
| SG-02 | Detect frame payload corruption using the LIN checksum algorithm. | ASIL-B |
| SG-03 | Correctly parse LDF definitions and decode payloads without offset errors. | ASIL-A |
| SG-04 | Detect E2E sequence gaps and CRC mismatches. | ASIL-A |
| SG-05 | Validate all frame IDs and data lengths at API boundaries. | ASIL-B |

## 3. Safety mechanisms provided

| Mechanism | API | Detects | Requirements |
|---|---|---|---|
| Frame validation | `lin.ValidateFrame` | out-of-range ID (>0x3F), empty / oversized payload (>8 B), non-classic checksum on diagnostic frames | REQ-LIN-001..003, 015..017 |
| PID parity | `lin.ProtectID`, `lin.VerifyPID` | corrupted / wrong frame ID | REQ-LIN-004..007, 018 |
| LIN checksum | `lin.CalcChecksum` (classic + enhanced) | payload corruption | REQ-LIN-008..010 |
| E2E protection | `safety.Protect` / `safety.Unwrap` | CRC mismatch, sequence gap/replay, length error | REQ-SAFETY-001..015 |
| LDF parse validation | `ldf.Parse` | malformed signal/frame/schedule definitions | REQ-LDF-001..015 |

## 4. Conditions of use (assumptions the integrator MUST satisfy)

These are normative. Violating them invalidates the safety case. The full list
with rationale is in [SEOOC.md §3](SEOOC.md); the safety-critical subset:

1. **Provide a correct LIN physical layer** (transceiver, break detection, bit
   timing). go-LIN operates above the physical layer (IA-01).
2. **Call `lin.ValidateFrame` on every externally received frame** before
   acting on it (IA-03, SG-05).
3. **Enforce frame-ID → actuator semantics in application logic.** go-LIN
   delivers frames as received; it does not know which actuator an ID drives
   (IA-02).
4. **Route safety-relevant multi-slot payloads through the `safety` package**
   and check the returned `E2EError` (IA-04, SG-04).
5. **Treat `lin.ErrNoResponse` as a system condition** (no slave answered), not
   a library fault (IA-05, REQ-SEOOC-007).
6. **Source LDF files from a trusted, version-controlled origin** (IA-06).
7. **Provide a monotonic clock with ≥1 ms resolution** for schedule timing
   (REQ-SEOOC-009).

## 5. Error handling contract

go-LIN reports faults synchronously as typed sentinel errors (RELAY §5); it
never panics on untrusted input and never silently drops a detected fault.
The integrator MUST check returned errors and place the system in its defined
safe state on:

| Error | Meaning | Required integrator action |
|---|---|---|
| `lin.ErrInvalidFrame` | structural frame violation | reject frame; do not transmit/process |
| `safety.E2EError{ErrCRCMismatch}` | payload corruption detected | discard payload; enter safe state |
| `safety.E2EError{ErrSequenceGap}` | lost/replayed frame | discard payload; enter safe state |
| `lin.ErrNoResponse` | no slave response | system-level handling (timeout/retry) |
| `lin.ErrTimeout`, `lin.ErrClosed`, `lin.ErrNotConnected` | transport state | per integrator transport policy |

## 6. Verification evidence

Every change is gated in CI (`.github/workflows/ci.yml`):

- Cross-platform unit + race tests, fuzz tests on all untrusted-input parsers.
- Full go-FuSa lifecycle: `check` (0 ERROR), `trace -req-coverage 100`
  (100% requirement traceability **and** function-annotation density),
  `cyber`, `vuln`, `qualify`.
- RELAY conformance: `relay conform --strict`, `relay interop --protocol LIN`.

Compliance gap reports (`gofusa iso26262 | do178 | iec61508 | iso21434 |
iec62443 | unece | slsa`) run in CI with **zero GAP and zero FAIL**; remaining
items are inherent human-review (MANUAL) attestations or N/A for a software
component.

## 7. Limitations / out of scope

See [SEOOC.md §5](SEOOC.md). Notably: hardware FMEDA, transceiver bit-error
analysis, multi-master topologies, and AUTOSAR LIN-stack integration are out
of scope and are the integrator's responsibility.

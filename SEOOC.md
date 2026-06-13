# Safety Element Out Of Context (SEOOC)
## go-LIN

**Document ID:** SEOOC-001
**Version:** 1.0
**Date:** 2026-06-13
**Author:** Matt Jones
**Standard:** ISO 26262:2018 Part 10 §9

---

## 1. Purpose

go-LIN is developed as a SEOOC targeting ASIL-B / SIL 2. This document defines
the integration assumptions that the integrating system must satisfy.

---

## 2. Assumed usage context

| Attribute | Value |
|---|---|
| Protocol | LIN 2.x |
| Network topology | Single master, one or more slaves |
| Baud rate range | 1 kbps – 20 kbps |
| Frame payload | 1–8 bytes |
| Checksum | Enhanced (LIN 2.x default) |
| Application domain | Automotive body electronics, HVAC, seat/mirror/window control |

---

## 3. Integration assumptions

| ID | Assumption | Rationale |
|---|---|---|
| IA-01 | The integrating system provides a correct LIN physical layer (transceiver, break detection, bit timing). | go-LIN operates above the physical layer. |
| IA-02 | The integrating system validates that frame IDs match the intended actuator before acting on received data. | go-LIN delivers frames as received; application logic must enforce semantic validity. |
| IA-03 | The integrating system uses `lin.ValidateFrame` before processing any externally received frame. | go-LIN validates frames at API entry points; raw hardware-layer data must be validated before calling go-LIN. |
| IA-04 | The integrating system uses the `safety` package for safety-relevant payloads spanning multiple LIN slots. | The 10-byte E2E header exceeds the standard 8-byte LIN payload limit. |
| IA-05 | The integrating system treats `lin.ErrNoResponse` as an indication that no slave is responding, not as an error in go-LIN itself. | Absence of a slave response is a system-level condition. |
| IA-06 | LDF files parsed by the `ldf` package originate from a trusted, version-controlled source. | go-LIN parses LDF syntactically; semantic correctness depends on LDF content. |

---

## 4. Safety measures provided by go-LIN

| Requirement | Measure |
|---|---|
| REQ-LIN-001 | `ValidateFrame` rejects frames with invalid ID (>0x3F) or data length (>8 bytes). |
| REQ-LIN-003 | `ProtectID` computes correct P0, P1 parity; `VerifyPID` detects parity errors. |
| REQ-LIN-004 | `CalcChecksum` computes LIN 1.x classic and LIN 2.x enhanced checksums. |
| REQ-SAFETY-005 | `Receiver.Unwrap` returns `ErrCRCMismatch` on any single-bit or multi-bit corruption. |
| REQ-SAFETY-004 | `Receiver.Unwrap` returns `ErrSequenceGap` on missed or replayed frames. |

---

## 5. Out of scope

- Hardware FMEDA (physical-layer fault model)
- Bit-error rate analysis of the LIN transceiver
- Airworthiness (DO-178C), rail (EN 50128), or machinery (IEC 62061)
- AUTOSAR LIN stack integration
- Multi-master LIN topologies (non-standard, not addressed by LIN 2.x)

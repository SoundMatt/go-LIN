# Hazard and Risk Analysis (HARA)
## go-LIN

**Document ID:** HARA-001
**Version:** 1.0
**Date:** 2026-06-13
**Author:** Matt Jones
**Standard:** ISO 26262:2018 Part 3

---

## 1. Scope

This HARA covers the software library `github.com/SoundMatt/go-LIN` used as a
SEOOC (Safety Element Out Of Context) in automotive and industrial LIN bus systems.

---

## 2. Hazardous events

| ID | Hazardous Event | Severity (S) | Exposure (E) | Controllability (C) | ASIL |
|---|---|---|---|---|---|
| H-01 | Master transmits header for wrong frame ID — slave actuates incorrect actuator | S2 | E3 | C2 | ASIL-B |
| H-02 | Corrupted LIN frame payload received without error detection — incorrect command sent to actuator | S2 | E3 | C2 | ASIL-B |
| H-03 | Incorrect PID parity allows wrong frame ID to be accepted | S2 | E3 | C2 | ASIL-B |
| H-04 | Checksum error not detected — corrupted data passed to application | S2 | E3 | C2 | ASIL-B |
| H-05 | LDF signal decoded with wrong bit offsets — actuator set to out-of-range value | S1 | E3 | C2 | ASIL-A |
| H-06 | E2E sequence counter gap not detected — replay attack or lost frame unnoticed | S2 | E2 | C2 | ASIL-A |

---

## 3. Safety goals (derived from H-01 to H-06)

| SG | Goal | ASIL |
|---|---|---|
| SG-01 | go-LIN shall correctly identify frame IDs using PID parity computation and verification. | ASIL-B |
| SG-02 | go-LIN shall detect frame payload corruption using the LIN checksum algorithm. | ASIL-B |
| SG-03 | go-LIN shall correctly parse LDF signal definitions and decode frame payloads without offset errors. | ASIL-A |
| SG-04 | go-LIN shall detect E2E sequence gaps and CRC mismatches. | ASIL-A |
| SG-05 | go-LIN shall validate all frame IDs and data lengths at API boundaries. | ASIL-B |

---

## 4. Requirement allocation

| SG | Requirement IDs |
|---|---|
| SG-01 | REQ-LIN-003, REQ-LIN-004 |
| SG-02 | REQ-LIN-004 |
| SG-03 | REQ-LDF-002, REQ-LDF-003 |
| SG-04 | REQ-SAFETY-004, REQ-SAFETY-005 |
| SG-05 | REQ-LIN-001 |

# Software Safety Plan
## go-LIN — ISO 26262 ASIL-B / IEC 61508 SIL 2

**Document ID:** SSP-001
**Version:** 1.0
**Date:** 2026-06-13
**Status:** Draft
**Author:** Matt Jones (matt@jellybaby.com)
**Standards:** ISO 26262:2018 Part 8 §7, IEC 61508-3:2010 §5

---

## 1. Purpose and scope

This Software Safety Plan (SSP) defines the lifecycle, activities, methods, and
responsibilities for the development and verification of go-LIN
(`github.com/SoundMatt/go-LIN`) in accordance with:

- ISO 26262:2018 — Road vehicles — Functional Safety (Parts 3, 4, 6, 8)
- IEC 61508:2010 — Functional Safety of E/E/PE Safety-related Systems (Part 3)

go-LIN is developed as a **Safety Element Out Of Context (SEOOC)** targeting
ASIL-B (ISO 26262) / SIL 2 (IEC 61508). A HARA.md will derive these levels
for each applicable use case.

**Out of scope:** System-level HARA, hardware fault model (FMEDA), airworthiness
(DO-178C), AUTOSAR integration. These are the integrating system's responsibility.

---

## 2. Applicable documents

| ID | Document | Location |
|---|---|---|
| SC | Safety Case | `safety-case.md` (generated) |
| CS | Coding Standard | `CODING_STANDARD.md` |
| FMEA | Differential FMEA table | `fmea.csv` (generated) |
| PROV | Build provenance | `provenance.json` (generated) |
| HARA | Hazard and Risk Analysis | `HARA.md` |
| SEOOC | SEOOC integration assumptions | `SEOOC.md` |

---

## 3. Organisation and responsibilities

| Role | Responsibility | Person / entity |
|---|---|---|
| Software developer | Implements requirements; runs unit tests | Matt Jones |
| Safety manager | Maintains this plan; approves releases | Matt Jones |
| Configuration manager | Controls baselines, tags, provenance | Automated (GitHub Actions + `gofusa release`) |

---

## 4. Safety requirements

Safety requirements are tracked as `//fusa:req REQ-*` annotations in source code
and verified by `//fusa:test REQ-*` annotations in test files. go-FuSa generates
traceability reports from these annotations.

Key requirement groups:

| Prefix | Description |
|---|---|
| REQ-LIN-* | Core Bus interface, Frame validation, PID, checksum |
| REQ-VIRT-* | Virtual bus correctness and isolation |
| REQ-LDF-* | LDF parser correctness and signal extraction |
| REQ-MASTER-* | Master node schedule execution |
| REQ-SLAVE-* | Slave node response registration and delivery |
| REQ-SAFETY-* | E2E protection header (CRC, sequence counter) |
| REQ-SEOOC-* | SEOOC integration assumptions |

---

## 5. Development process

- Language: Go 1.25+
- Version control: Git (GitHub)
- Review: All changes via pull request; CI must pass before merge
- Testing: `go test -race -count=1 ./...`; fuzz targets in `virtual/`, `ldf/`, `safety/`
- Safety gate: `gofusa check ./...` must pass (no ERROR-level findings)
- Release: Tagged `vX.Y.Z`; safety artifacts regenerated automatically on tag

---

## 6. Verification

| Method | Tool | Scope |
|---|---|---|
| Static analysis | `go vet` | All packages |
| Race detection | `go test -race` | All packages |
| Fuzz testing | `go test -fuzz` | virtual, ldf, safety |
| Safety gate | `gofusa check` | All packages |
| Requirement traceability | `gofusa fmea` | `//fusa:req` annotations |

---

## 7. Configuration management

Releases are tagged `vX.Y.Z`. The GitHub Actions release workflow regenerates
safety artifacts (`fmea.csv`, `fmea.json`, `safety-case.md`, `safety-case.json`,
`safety-case.mermaid`, `sbom.json`, `provenance.json`, `artifact-manifest.json`)
and commits them to main on every tag.

---

## 8. Independence

go-LIN targets ASIL-B / SIL 2 with independence policy equivalent to go-CAN:
the developer and safety verifier are the same person (Matt Jones) for initial
releases. For ASIL-C/D or SIL 3/4 integrations, an independent verifier must
review each release.

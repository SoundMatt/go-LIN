# Cyber-Incident Response Plan

**Component:** go-LIN (LIN protocol library, SEOOC)
**Standard:** IEC 62443-4-2 CR 6.2.1 · ISO/SAE 21434 clause 13 (incident response)

This plan defines how a confirmed cybersecurity incident affecting go-LIN is
handled. Vulnerability **reporting and disclosure** are covered in
[SECURITY.md](SECURITY.md); this document covers the **response process** once
an incident is confirmed.

## Roles

| Role | Responsibility |
|------|----------------|
| Maintainer (Matt Jones) | Incident lead: triage, decisions, disclosure |
| Reporter | Provides details, validates the fix |
| Integrators | Apply the patched release in their ECU/system context |

## Severity classification

Severity is assigned with CVSS v3.1 and cross-referenced to the
[TARA](tara.md) threat that materialised:

| Severity | CVSS | Initial response target |
|----------|------|-------------------------|
| Critical | 9.0–10.0 | interim mitigation within 48 h |
| High | 7.0–8.9 | interim mitigation within 5 business days |
| Medium | 4.0–6.9 | fix in next patch release |
| Low | 0.1–3.9 | fix in next scheduled release |

## Response workflow

1. **Detect / intake** — register the incident in the problem-report log
   (`.fusa-problems.json`) with a `security` classification and link the
   originating report.
2. **Contain** — determine which supported versions are affected; if the
   threat is actively exploited, publish an interim mitigation advisory and
   notify known integrators.
3. **Eradicate** — develop and test the fix on a private branch. Before
   release, the **full go-FuSa lifecycle** (`check`, `trace -req-coverage 100`,
   `cyber`, `vuln`, `qualify`) and **RELAY conformance** (`relay conform
   --strict`, `relay interop --protocol LIN`) must pass.
4. **Recover** — cut a patched release, publish the GitHub Security Advisory,
   request a CVE where warranted, and refresh `vuln.json`, `cyber-report.json`
   and the TARA.
5. **Post-incident review** — record root cause and corrective actions in the
   safety case; if the attack surface or threat model changed, regenerate the
   TARA (`gofusa tara`) and, if a new hazard emerged, the HARA.

## Communication

- Private channel during embargo: GitHub Security Advisory + `matt@jellybaby.com`.
- Public disclosure: coordinated with the reporter, published as a GitHub
  Security Advisory and in the release notes of the patched version.

## Records

Every incident produces: a problem-report entry, a security advisory, an
updated `vuln.json`, and (where the model changed) a regenerated `tara.json`.
These are bundled into the release audit pack (`gofusa audit-pack`).

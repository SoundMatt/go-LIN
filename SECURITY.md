# Security Policy

This document is the vulnerability-management and cyber-incident-response
policy for **go-LIN**, satisfying IEC 62443-4-2 CR 6.2 / CR 6.2.1 and
ISO/SAE 21434 clause 6 (cybersecurity management) and clause 8 (vulnerability
management). go-LIN is a Safety Element Out Of Context (SEOOC); see
[SEOOC.md](SEOOC.md) for the assumptions placed on the integrating system.

## Supported versions

Security fixes are provided for the latest released minor version. Older
versions should be upgraded to receive fixes.

| Version | Supported |
|---------|-----------|
| 1.x     | ✅         |
| < 1.0   | ❌         |

## Reporting a vulnerability

Report suspected vulnerabilities **privately** — do not open a public issue.

- Preferred: GitHub **Security Advisories** → *Report a vulnerability* on
  <https://github.com/SoundMatt/go-LIN/security/advisories/new>.
- Alternative: email the maintainer at `matt@jellybaby.com` with subject
  `go-LIN security`.

Please include: affected version/commit, a description, reproduction steps or a
proof-of-concept, and the assessed impact. Do not include exploit code in any
public channel.

## Coordinated disclosure timeline

| Stage | Target |
|-------|--------|
| Acknowledge receipt | within 3 business days |
| Triage + severity (CVSS v3.1) | within 10 business days |
| Fix or mitigation for supported versions | within 90 days of triage |
| Public advisory + CVE (if warranted) | coordinated with the reporter on release |

## Cyber-incident response plan (IEC 62443-4-2 CR 6.2.1)

On confirmation of a security incident affecting go-LIN:

1. **Detect / intake** — log the report in the project's problem-report
   register (`.fusa-problems.json`) with a security classification.
2. **Contain** — assess exposure across supported versions; if actively
   exploited, publish an interim mitigation advisory.
3. **Eradicate** — develop and test a fix on a private branch; run the full
   go-FuSa lifecycle (`check`, `trace`, `cyber`, `vuln`, `qualify`) and RELAY
   conformance (`relay conform --strict`, `relay interop`) before release.
4. **Recover** — release a patched version, publish the GitHub Security
   Advisory, request a CVE where warranted, and update `vuln.json` / TARA.
5. **Post-incident** — record root cause and corrective actions in the safety
   case and, where the threat model changed, regenerate the TARA
   (`gofusa tara`).

## Vulnerability monitoring (ISO 21434 clause 8.3)

Dependencies are scanned on every CI run with `gofusa vuln` (OSV database);
the result is recorded in `vuln.json`. The Software Bill of Materials
(`sbom.json`) enables downstream impact analysis when a new advisory is
published against a transitive dependency.

## Threat model

See [tara.md](tara.md) for the ISO/SAE 21434 Threat Analysis and Risk
Assessment, and [HARA.md](HARA.md) for the functional-safety hazard analysis.

// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// This file declares the Safety Element Out Of Context (SEOOC) assumptions
// and integration requirements for go-LIN. These requirements describe what
// the integrating system MUST provide or ensure. go-LIN is compliant with
// ISO 26262 ASIL-B at the SEOOC level: the remaining ASIL-B obligations are
// allocated to the integrating system.
//
// Assumption requirements (REQ-SEOOC-002, 003, 007, 008, 009) are
// obligations on the system that uses go-LIN, not on go-LIN itself.
// They are recorded here so that go-FuSa can include them in the safety case
// and traceability report.
//
// Integration requirements (REQ-SEOOC-004, 005, 006) are verified by the
// integration tests in seooc_test.go.
//
//fusa:req REQ-SEOOC-001
//fusa:req REQ-SEOOC-002
//fusa:req REQ-SEOOC-003
//fusa:req REQ-SEOOC-004
//fusa:req REQ-SEOOC-005
//fusa:req REQ-SEOOC-006
//fusa:req REQ-SEOOC-007
//fusa:req REQ-SEOOC-008
//fusa:req REQ-SEOOC-009
package lin

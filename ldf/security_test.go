// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ldf_test

import (
	"strings"
	"testing"

	"github.com/SoundMatt/go-LIN/ldf"
)

// TestParse_rejectsMalformedInput is the requirement-based security test for
// REQ-SEC-001: the LDF parser treats its input as untrusted and must return an
// error (or an empty database) without panicking, for malformed, truncated, or
// hostile LDF text. A panic on untrusted input would be a denial-of-service
// vector (ISO/SAE 21434, CWE-20).
//
//fusa:test REQ-SEC-001
func TestParse_rejectsMalformedInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"garbage", "\x00\x01\x02 not an ldf at all \xff\xfe"},
		{"truncated block", "Nodes {\n  Master:"},
		{"unterminated string", `LIN_protocol_version = "2.1`},
		{"unbalanced braces", "Frames {{{{{{{{{{"},
		{"huge id", "Frames { f: 99999999999999999999, MASTER, 8 ;"},
		{"negative length", "Frames { f: 0x10, MASTER, -4 ;"},
		{"only braces", strings.Repeat("{", 1000) + strings.Repeat("}", 1000)},
		{"null bytes mid-keyword", "Sig\x00nals {}"},
		{"deeply nested", strings.Repeat("Frames {", 200)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The contract is "does not panic"; an error is acceptable and
			// expected. Parse must never crash on untrusted input.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Parse panicked on %q: %v", tc.name, r)
				}
			}()
			db, err := ldf.Parse(strings.NewReader(tc.input))
			if err != nil {
				return // expected for malformed input
			}
			// If it parsed without error, accessors must also be panic-safe.
			_ = db.Frames()
			_ = db.Signals()
			for id := uint8(0); id <= 0x3F; id++ {
				_ = db.Frame(id)
			}
		})
	}
}

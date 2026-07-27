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

// TestEncode_packsSignals packs two 8-bit signals into a two-byte payload,
// exercising Encode and packBits across a byte boundary.
func TestEncode_packsSignals(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := db.Encode(0x10, map[string]uint64{"Lo8": 0xAB, "Hi8": 0xCD})
	if out == nil {
		t.Fatal("Encode returned nil for a known frame")
	}
	want := []byte{0xAB, 0xCD}
	if len(out) != len(want) || out[0] != want[0] || out[1] != want[1] {
		t.Errorf("Encode = % X, want % X", out, want)
	}
}

// TestEncode_roundTripsWithDecode verifies Decode(Encode(id, signals)) is the
// identity for signal values that fit within their declared bit width.
func TestEncode_roundTripsWithDecode(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	in := map[string]uint64{"Lo8": 0x42, "Hi8": 0x99}
	payload := db.Encode(0x10, in)
	out := db.Decode(0x10, payload)
	for name, want := range in {
		if out[name] != want {
			t.Errorf("round-trip %s = 0x%02X, want 0x%02X", name, out[name], want)
		}
	}
}

// TestEncode_missingSignalUsesInitValue verifies a signal absent from the
// input map is packed with its declared InitValue rather than left undefined.
func TestEncode_missingSignalUsesInitValue(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Neither signal is supplied; both InitValues in decodeLDF are 0x00.
	out := db.Encode(0x10, nil)
	if out[0] != 0x00 || out[1] != 0x00 {
		t.Errorf("Encode(nil) = % X, want the signals' InitValue (00 00)", out)
	}
}

// TestEncode_unknownFrame returns nil for a frame ID that does not exist in
// the LDF, matching Decode's behaviour for the same case.
func TestEncode_unknownFrame(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if out := db.Encode(0x3F, map[string]uint64{"Lo8": 1}); out != nil {
		t.Errorf("Encode(unknown frame) = % X, want nil", out)
	}
}

// TestEncode_truncatesOverwideValues verifies a value wider than the
// signal's declared BitWidth is truncated to the low BitWidth bits rather
// than corrupting adjacent signals.
func TestEncode_truncatesOverwideValues(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out := db.Encode(0x10, map[string]uint64{"Lo8": 0x1FF, "Hi8": 0x00}) // Lo8 is only 8 bits wide
	if out[0] != 0xFF {
		t.Errorf("Lo8 truncated = 0x%02X, want 0xFF (low 8 bits of 0x1FF)", out[0])
	}
	if out[1] != 0x00 {
		t.Errorf("Hi8 = 0x%02X, want 0x00 (must not be corrupted by Lo8's overflow)", out[1])
	}
}

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

const decodeLDF = `
LIN_description_file;
LIN_protocol_version = "2.1";
LIN_speed = 19.2 kbps;

Nodes {
  Master: MasterNode, 1 ms, 0.1 ms;
  Slaves: SlaveA;
}

Signals {
  Lo8:  8, 0x00, SlaveA, MasterNode;
  Hi8:  8, 0x00, SlaveA, MasterNode;
}

Frames {
  DataFrame: 0x10, SlaveA, 2 {
    Lo8, 0;
    Hi8, 8;
  }
}
`

// TestDecode_extractsSignals decodes a two-byte frame into its two 8-bit
// signals, exercising Decode and extractBits across byte boundaries.
func TestDecode_extractsSignals(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := db.Decode(0x10, []byte{0xAB, 0xCD})
	if out == nil {
		t.Fatal("Decode returned nil for a known frame")
	}
	if out["Lo8"] != 0xAB {
		t.Errorf("Lo8 = 0x%02X, want 0xAB", out["Lo8"])
	}
	if out["Hi8"] != 0xCD {
		t.Errorf("Hi8 = 0x%02X, want 0xCD", out["Hi8"])
	}
}

// TestDecode_shortPayload exercises the extractBits out-of-range break path:
// a payload shorter than the signal layout yields zero for the missing bits
// without panicking.
func TestDecode_shortPayload(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out := db.Decode(0x10, []byte{0xFF}) // only the low byte present
	if out["Lo8"] != 0xFF {
		t.Errorf("Lo8 = 0x%02X, want 0xFF", out["Lo8"])
	}
	if out["Hi8"] != 0x00 {
		t.Errorf("Hi8 = 0x%02X, want 0x00 (bits beyond payload)", out["Hi8"])
	}
}

// TestSchedule_unknown returns nil for a schedule table that does not exist.
func TestSchedule_unknown(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(decodeLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s := db.Schedule("DoesNotExist"); s != nil {
		t.Errorf("Schedule(unknown) = %v, want nil", s)
	}
}

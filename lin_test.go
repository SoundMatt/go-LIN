// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin_test

import (
	"errors"
	"testing"

	lin "github.com/SoundMatt/go-LIN"
)

// ── ProtectID ────────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-004
//fusa:test REQ-LIN-005
//fusa:test REQ-LIN-018

func TestProtectID_P0(t *testing.T) {
	// P0 = ID0^ID1^ID2^ID4; verify bit 6 of PID matches
	cases := []struct{ id, pid uint8 }{
		{0x00, 0x80}, // P0=0, P1=1
		{0x01, 0xC1}, // P0=1, P1=1
		{0x10, 0x50}, // P0=1, P1=0
		{0x12, 0x92}, // P0=0, P1=1
		{0x3F, 0xBF}, // P0=0, P1=1
		{0x3C, 0x3C}, // P0=0, P1=0
	}
	for _, tt := range cases {
		got := lin.ProtectID(tt.id)
		if got != tt.pid {
			t.Errorf("ProtectID(0x%02X) = 0x%02X, want 0x%02X", tt.id, got, tt.pid)
		}
	}
}

func TestProtectID_allIDs(t *testing.T) {
	for id := uint8(0); id <= lin.MaxID; id++ {
		pid := lin.ProtectID(id)
		if pid&0x3F != id {
			t.Errorf("ProtectID(0x%02X): lower 6 bits 0x%02X != id", id, pid&0x3F)
		}
	}
}

// ── VerifyPID ────────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-006
//fusa:test REQ-LIN-007

func TestVerifyPID_valid(t *testing.T) {
	for id := uint8(0); id <= lin.MaxID; id++ {
		pid := lin.ProtectID(id)
		got, err := lin.VerifyPID(pid)
		if err != nil {
			t.Fatalf("VerifyPID(0x%02X): unexpected error: %v", pid, err)
		}
		if got != id {
			t.Errorf("VerifyPID(0x%02X) = 0x%02X, want 0x%02X", pid, got, id)
		}
	}
}

func TestVerifyPID_invalid(t *testing.T) {
	// Corrupt P0 bit of ID=0x10 PID=0x50 → 0x10 (P0 flipped)
	pid := lin.ProtectID(0x10) ^ 0x40
	if _, err := lin.VerifyPID(pid); err == nil {
		t.Error("VerifyPID: expected error for bad parity, got nil")
	}
}

func TestVerifyPID_corruptP1(t *testing.T) {
	// Corrupt P1 bit
	pid := lin.ProtectID(0x10) ^ 0x80
	if _, err := lin.VerifyPID(pid); err == nil {
		t.Error("VerifyPID: expected error for corrupt P1 bit")
	}
}

// ── CalcChecksum ─────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-008
//fusa:test REQ-LIN-009
//fusa:test REQ-LIN-010

func TestCalcChecksum_enhanced(t *testing.T) {
	pid := lin.ProtectID(0x10)
	data := []byte{0x01, 0x02}
	cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	// Verify: sum of PID + data + cs (with carry-around) == 0xFF
	var sum uint16 = uint16(pid)
	for _, b := range data {
		sum += uint16(b)
		if sum > 0xFF {
			sum -= 0xFF
		}
	}
	sum += uint16(cs)
	if sum > 0xFF {
		sum -= 0xFF
	}
	if sum != 0xFF {
		t.Errorf("enhanced checksum: sum+cs = 0x%02X, want 0xFF", sum)
	}
}

func TestCalcChecksum_classic_excludesPID(t *testing.T) {
	data := []byte{0xAA, 0x55}
	cs1 := lin.CalcChecksum(0x50, data, lin.ClassicChecksum)
	cs2 := lin.CalcChecksum(0x92, data, lin.ClassicChecksum)
	if cs1 != cs2 {
		t.Error("classic checksum must not depend on PID value")
	}
}

func TestCalcChecksum_enhanced_includesPID(t *testing.T) {
	data := []byte{0xAA, 0x55}
	cs1 := lin.CalcChecksum(0x50, data, lin.EnhancedChecksum)
	cs2 := lin.CalcChecksum(0x92, data, lin.EnhancedChecksum)
	if cs1 == cs2 {
		t.Error("enhanced checksum must differ when PID differs")
	}
}

func TestCalcChecksum_carryAround(t *testing.T) {
	// Data that forces a carry: 0xFF + 0x01 → carry
	data := []byte{0xFF, 0x01}
	pid := lin.ProtectID(0x00)
	cs := lin.CalcChecksum(pid, data, lin.ClassicChecksum)
	var sum uint16
	for _, b := range data {
		sum += uint16(b)
		if sum > 0xFF {
			sum -= 0xFF
		}
	}
	sum += uint16(cs)
	if sum > 0xFF {
		sum -= 0xFF
	}
	if sum != 0xFF {
		t.Errorf("carry-around: sum+cs = 0x%02X, want 0xFF", sum)
	}
}

// ── ValidateFrame ─────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-001
//fusa:test REQ-LIN-002
//fusa:test REQ-LIN-003
//fusa:test REQ-LIN-015
//fusa:test REQ-LIN-016
//fusa:test REQ-LIN-017

func TestValidateFrame_rejectsHighID(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x40, Data: []byte{0x01}}); err == nil {
		t.Error("expected error for ID=0x40 (> MaxID)")
	}
}

func TestValidateFrame_rejectsEmptyData(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x10, Data: nil}); err == nil {
		t.Error("expected error for nil data")
	}
	if err := lin.ValidateFrame(lin.Frame{ID: 0x10, Data: []byte{}}); err == nil {
		t.Error("expected error for zero-length data")
	}
}

func TestValidateFrame_rejectsOversizedData(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x10, Data: make([]byte, 9)}); err == nil {
		t.Error("expected error for 9-byte data (> MaxDataLen)")
	}
}

func TestValidateFrame_acceptsMaxID(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x3F, Data: []byte{0x01}}); err != nil {
		t.Errorf("unexpected error for ID=0x3F: %v", err)
	}
}

func TestValidateFrame_acceptsMaxData(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x10, Data: make([]byte, 8)}); err != nil {
		t.Errorf("unexpected error for 8-byte data: %v", err)
	}
}

func TestValidateFrame_acceptsMinData(t *testing.T) {
	if err := lin.ValidateFrame(lin.Frame{ID: 0x10, Data: []byte{0x01}}); err != nil {
		t.Errorf("unexpected error for 1-byte data: %v", err)
	}
}

// ── ErrNoResponse sentinel ───────────────────────────────────────────────────

//fusa:test REQ-LIN-021

func TestErrNoResponse_isSentinel(t *testing.T) {
	if lin.ErrNoResponse == nil {
		t.Fatal("ErrNoResponse must not be nil")
	}
	if !errors.Is(lin.ErrNoResponse, lin.ErrNoResponse) {
		t.Error("errors.Is(ErrNoResponse, ErrNoResponse) must return true")
	}
}

// ── Filter ────────────────────────────────────────────────────────────────────

func TestFilterMatches_exact(t *testing.T) {
	flt := lin.Filter{ID: 0x10}
	if !flt.Matches(lin.Frame{ID: 0x10, Data: []byte{1}}) {
		t.Error("exact filter should match ID 0x10")
	}
	if flt.Matches(lin.Frame{ID: 0x20, Data: []byte{1}}) {
		t.Error("exact filter should not match ID 0x20")
	}
}

func TestFilterMatches_all(t *testing.T) {
	flt := lin.Filter{All: true}
	for id := uint8(0); id <= lin.MaxID; id++ {
		if !flt.Matches(lin.Frame{ID: id, Data: []byte{1}}) {
			t.Errorf("all-filter should match ID 0x%02X", id)
		}
	}
}

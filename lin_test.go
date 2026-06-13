// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin_test

import (
	"testing"

	lin "github.com/SoundMatt/go-LIN"
)

//fusa:test REQ-LIN-003

func TestProtectID(t *testing.T) {
	tests := []struct {
		id   uint8
		want uint8
	}{
		{0x00, 0x80}, // P0=0, P1=1
		{0x01, 0xC1}, // P0=1, P1=1
		{0x10, 0x50}, // standard LIN example
		{0x12, 0x92},
		{0x3F, 0xBF},
		{0x3C, 0x3C}, // master request diagnostic
	}
	for _, tt := range tests {
		got := lin.ProtectID(tt.id)
		if got != tt.want {
			t.Errorf("ProtectID(0x%02X) = 0x%02X, want 0x%02X", tt.id, got, tt.want)
		}
	}
}

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
	// Corrupt the parity of ID=0x10
	pid := lin.ProtectID(0x10) ^ 0x40
	if _, err := lin.VerifyPID(pid); err == nil {
		t.Error("VerifyPID: expected error for bad parity, got nil")
	}
}

//fusa:test REQ-LIN-004

func TestCalcChecksum_enhanced(t *testing.T) {
	// LIN 2.x example: ID=0x10, data={0x01,0x02}
	// Enhanced: sum PID + data with carry-around 8-bit addition, inverted.
	pid := lin.ProtectID(0x10)
	data := []byte{0x01, 0x02}
	cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	// Reconstruct and verify (checksum + sum of PID + data, with carry) == 0xFF
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
		t.Errorf("enhanced checksum verification failed: sum+cs = 0x%02X", sum)
	}
}

func TestCalcChecksum_classic(t *testing.T) {
	data := []byte{0xAA, 0x55}
	cs := lin.CalcChecksum(0, data, lin.ClassicChecksum)
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
		t.Errorf("classic checksum verification failed: sum+cs = 0x%02X", sum)
	}
}

//fusa:test REQ-LIN-001

func TestValidateFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   lin.Frame
		wantErr bool
	}{
		{"valid", lin.Frame{ID: 0x10, Data: []byte{0x01}}, false},
		{"id too large", lin.Frame{ID: 0x40, Data: []byte{0x01}}, true},
		{"empty data", lin.Frame{ID: 0x10, Data: nil}, true},
		{"data too long", lin.Frame{ID: 0x10, Data: make([]byte, 9)}, true},
		{"max data", lin.Frame{ID: 0x3F, Data: make([]byte, 8)}, false},
	}
	for _, tt := range tests {
		err := lin.ValidateFrame(tt.frame)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: ValidateFrame() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestFilterMatches(t *testing.T) {
	f10 := lin.Frame{ID: 0x10, Data: []byte{1}}
	f20 := lin.Frame{ID: 0x20, Data: []byte{1}}

	exactFilt := lin.Filter{ID: 0x10}
	allFilt := lin.Filter{All: true}

	if !exactFilt.Matches(f10) {
		t.Error("exact filter should match ID 0x10")
	}
	if exactFilt.Matches(f20) {
		t.Error("exact filter should not match ID 0x20")
	}
	if !allFilt.Matches(f10) || !allFilt.Matches(f20) {
		t.Error("all-frames filter should match every frame")
	}
}

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

func TestMasterRequestFrame_toFromRoundTrip(t *testing.T) {
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2, Data: []byte{0x01, 0x02, 0x03}}
	f, err := req.ToFrame()
	if err != nil {
		t.Fatalf("ToFrame: %v", err)
	}
	if f.ID != lin.LINDiagRequestID {
		t.Errorf("f.ID = 0x%02X, want LINDiagRequestID (0x%02X)", f.ID, lin.LINDiagRequestID)
	}
	if len(f.Data) != lin.LINMaxDataLen {
		t.Fatalf("f.Data length = %d, want %d", len(f.Data), lin.LINMaxDataLen)
	}
	if f.ChecksumType != lin.ClassicChecksum {
		t.Errorf("f.ChecksumType = %v, want ClassicChecksum (diagnostic frames are always classic)", f.ChecksumType)
	}
	if err := lin.ValidateFrame(f); err != nil {
		t.Errorf("ValidateFrame(ToFrame output) = %v, want nil", err)
	}

	got, err := lin.ParseMasterRequestFrame(f)
	if err != nil {
		t.Fatalf("ParseMasterRequestFrame: %v", err)
	}
	if got.NAD != req.NAD || got.SID != req.SID {
		t.Errorf("round-trip NAD/SID = 0x%02X/0x%02X, want 0x%02X/0x%02X", got.NAD, got.SID, req.NAD, req.SID)
	}
	if string(got.Data) != string(req.Data) {
		t.Errorf("round-trip Data = % X, want % X", got.Data, req.Data)
	}
}

func TestSlaveResponseFrame_toFromRoundTrip(t *testing.T) {
	resp := lin.SlaveResponseFrame{NAD: 0x01, RSID: 0xF2, Data: []byte{0xAA, 0xBB}}
	f, err := resp.ToFrame()
	if err != nil {
		t.Fatalf("ToFrame: %v", err)
	}
	if f.ID != lin.LINDiagResponseID {
		t.Errorf("f.ID = 0x%02X, want LINDiagResponseID (0x%02X)", f.ID, lin.LINDiagResponseID)
	}

	got, err := lin.ParseSlaveResponseFrame(f)
	if err != nil {
		t.Fatalf("ParseSlaveResponseFrame: %v", err)
	}
	if got.NAD != resp.NAD || got.RSID != resp.RSID {
		t.Errorf("round-trip NAD/RSID = 0x%02X/0x%02X, want 0x%02X/0x%02X", got.NAD, got.RSID, resp.NAD, resp.RSID)
	}
	if string(got.Data) != string(resp.Data) {
		t.Errorf("round-trip Data = % X, want % X", got.Data, resp.Data)
	}
}

func TestMasterRequestFrame_emptyData(t *testing.T) {
	req := lin.MasterRequestFrame{NAD: 0x7F, SID: 0x10}
	f, err := req.ToFrame()
	if err != nil {
		t.Fatalf("ToFrame: %v", err)
	}
	got, err := lin.ParseMasterRequestFrame(f)
	if err != nil {
		t.Fatalf("ParseMasterRequestFrame: %v", err)
	}
	if len(got.Data) != 0 {
		t.Errorf("Data = % X, want empty", got.Data)
	}
}

func TestMasterRequestFrame_dataTooLong(t *testing.T) {
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2, Data: make([]byte, lin.DiagDataLen+1)}
	_, err := req.ToFrame()
	if !errors.Is(err, lin.ErrInvalidFrame) {
		t.Errorf("ToFrame(oversize data): errors.Is(err, lin.ErrInvalidFrame) = false, want true (err=%v)", err)
	}
}

func TestParseMasterRequestFrame_wrongLength(t *testing.T) {
	f := lin.Frame{ID: lin.LINDiagRequestID, Data: []byte{0x01, 0x02}}
	_, err := lin.ParseMasterRequestFrame(f)
	if !errors.Is(err, lin.ErrInvalidFrame) {
		t.Errorf("ParseMasterRequestFrame(short frame): errors.Is(err, lin.ErrInvalidFrame) = false, want true (err=%v)", err)
	}
}

func TestParseMasterRequestFrame_invalidPCI(t *testing.T) {
	data := make([]byte, lin.LINMaxDataLen)
	for i := range data {
		data[i] = 0xFF
	}
	data[1] = 0x00 // PCI = 0 is out of the single-frame range
	f := lin.Frame{ID: lin.LINDiagRequestID, Data: data}
	_, err := lin.ParseMasterRequestFrame(f)
	if !errors.Is(err, lin.ErrInvalidFrame) {
		t.Errorf("ParseMasterRequestFrame(PCI=0): errors.Is(err, lin.ErrInvalidFrame) = false, want true (err=%v)", err)
	}
}

// TestDiagPadding verifies unused data bytes are padded with 0xFF, per
// LIN 2.x §4.2.3, not left as zero.
func TestDiagPadding(t *testing.T) {
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0x10, Data: []byte{0x01}}
	f, err := req.ToFrame()
	if err != nil {
		t.Fatalf("ToFrame: %v", err)
	}
	// bytes 0..3 are NAD, PCI, SID, data[0]; bytes 4..7 must be padding.
	for i := 4; i < lin.LINMaxDataLen; i++ {
		if f.Data[i] != 0xFF {
			t.Errorf("f.Data[%d] = 0x%02X, want 0xFF padding", i, f.Data[i])
		}
	}
}

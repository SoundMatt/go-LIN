// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin

import "fmt"

// DiagDataLen is the number of application data bytes carried by a single-
// frame LIN diagnostic message, after the NAD, PCI, and SID/RSID bytes
// (LIN 2.x §4.2.3): 8 total frame bytes - 3 header bytes = 5.
const DiagDataLen = 5

// diagPadding fills unused diagnostic data bytes, per LIN 2.x §4.2.3.
const diagPadding = 0xFF

// MasterRequestFrame is the LIN master request diagnostic frame (ID 0x3C,
// LINDiagRequestID). It is always exactly 8 bytes on the wire: NAD, a
// single-frame PCI (length), SID, and up to 5 data bytes padded with 0xFF.
//
// This models the LIN 2.x single-frame diagnostic transport layer
// (LIN 2.x §4.2.3); multi-frame (first-frame/consecutive-frame) transport is
// not covered.
type MasterRequestFrame struct {
	// NAD is the Node Address of the target slave (0x01-0x7D individual,
	// 0x7E functional/broadcast-to-all-diagnostic-nodes, 0x7F broadcast).
	NAD uint8

	// SID is the application-defined Service Identifier.
	SID uint8

	// Data holds up to DiagDataLen bytes of request data. Len(Data) MUST be
	// <= DiagDataLen; ToFrame pads the remainder with 0xFF.
	Data []byte
}

// SlaveResponseFrame is the LIN slave response diagnostic frame (ID 0x3D,
// LINDiagResponseID). Layout mirrors MasterRequestFrame with RSID (the
// response service identifier, conventionally SID+0x40) in place of SID.
type SlaveResponseFrame struct {
	// NAD is the Node Address of the responding slave.
	NAD uint8

	// RSID is the Response Service Identifier.
	RSID uint8

	// Data holds up to DiagDataLen bytes of response data.
	Data []byte
}

// ToFrame packs a MasterRequestFrame into the wire-level 8-byte classic-
// checksum lin.Frame transmitted on LINDiagRequestID.
//
//fusa:req REQ-LIN-001
func (r MasterRequestFrame) ToFrame() (Frame, error) {
	data, err := packDiag(r.NAD, r.SID, r.Data)
	if err != nil {
		return Frame{}, fmt.Errorf("lin: MasterRequestFrame: %w", err)
	}
	pid := ProtectID(LINDiagRequestID)
	return Frame{
		ID:           LINDiagRequestID,
		Data:         data,
		Checksum:     CalcChecksum(pid, data, ClassicChecksum),
		ChecksumType: ClassicChecksum,
	}, nil
}

// ToFrame packs a SlaveResponseFrame into the wire-level 8-byte classic-
// checksum lin.Frame transmitted on LINDiagResponseID.
//
//fusa:req REQ-LIN-001
func (r SlaveResponseFrame) ToFrame() (Frame, error) {
	data, err := packDiag(r.NAD, r.RSID, r.Data)
	if err != nil {
		return Frame{}, fmt.Errorf("lin: SlaveResponseFrame: %w", err)
	}
	pid := ProtectID(LINDiagResponseID)
	return Frame{
		ID:           LINDiagResponseID,
		Data:         data,
		Checksum:     CalcChecksum(pid, data, ClassicChecksum),
		ChecksumType: ClassicChecksum,
	}, nil
}

// packDiag lays out [NAD, PCI, SID/RSID, data..., 0xFF-padding] as an
// 8-byte diagnostic payload per LIN 2.x §4.2.3.
func packDiag(nad, sidOrRsid uint8, data []byte) ([]byte, error) {
	if len(data) > DiagDataLen {
		return nil, fmt.Errorf("data length %d exceeds maximum %d: %w", len(data), DiagDataLen, ErrInvalidFrame)
	}
	out := make([]byte, LINMaxDataLen)
	for i := range out {
		out[i] = diagPadding
	}
	out[0] = nad
	out[1] = uint8(len(data) + 1) // PCI: single-frame length = SID/RSID byte + data bytes
	out[2] = sidOrRsid
	copy(out[3:], data)
	return out, nil
}

// ParseMasterRequestFrame extracts a MasterRequestFrame from the wire-level
// lin.Frame received on LINDiagRequestID. It returns ErrInvalidFrame if f is
// not an 8-byte frame with a valid single-frame PCI.
//
//fusa:req REQ-SEC-005
func ParseMasterRequestFrame(f Frame) (MasterRequestFrame, error) {
	nad, sid, data, err := unpackDiag(f)
	if err != nil {
		return MasterRequestFrame{}, fmt.Errorf("lin: MasterRequestFrame: %w", err)
	}
	return MasterRequestFrame{NAD: nad, SID: sid, Data: data}, nil
}

// ParseSlaveResponseFrame extracts a SlaveResponseFrame from the wire-level
// lin.Frame received on LINDiagResponseID. It returns ErrInvalidFrame if f
// is not an 8-byte frame with a valid single-frame PCI.
//
//fusa:req REQ-SEC-005
func ParseSlaveResponseFrame(f Frame) (SlaveResponseFrame, error) {
	nad, rsid, data, err := unpackDiag(f)
	if err != nil {
		return SlaveResponseFrame{}, fmt.Errorf("lin: SlaveResponseFrame: %w", err)
	}
	return SlaveResponseFrame{NAD: nad, RSID: rsid, Data: data}, nil
}

// unpackDiag is the shared decode path for both diagnostic frame types: it
// validates length and PCI, then splits [NAD, PCI, SID/RSID, data...] out of
// f.Data per LIN 2.x §4.2.3.
func unpackDiag(f Frame) (nad, sidOrRsid uint8, data []byte, err error) {
	if len(f.Data) != LINMaxDataLen {
		return 0, 0, nil, fmt.Errorf("diagnostic frame must be %d bytes, got %d: %w", LINMaxDataLen, len(f.Data), ErrInvalidFrame)
	}
	nad = f.Data[0]
	pci := f.Data[1]
	if pci == 0 || pci > uint8(DiagDataLen+1) {
		return 0, 0, nil, fmt.Errorf("diagnostic frame PCI 0x%02X out of single-frame range 0x01-0x%02X: %w", pci, DiagDataLen+1, ErrInvalidFrame)
	}
	sidOrRsid = f.Data[2]
	n := int(pci) - 1 // data length = PCI (SID/RSID + data bytes) minus the SID/RSID byte
	data = append([]byte(nil), f.Data[3:3+n]...)
	return nad, sidOrRsid, data, nil
}
